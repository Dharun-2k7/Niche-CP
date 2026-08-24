package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/Dharun-2k7/online-coding-platform/internal/judge"
	"github.com/joho/godotenv"
)

type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
}

var (
	activeSandboxes int32
	peakSandboxes   int32
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using default environment variables.")
	}

	db.InitPostgres()
	db.InitRedis()
	judge.InitSemaphore()

	maxWorkers := 2 // Default configuration
	if val := os.Getenv("MAX_EXECUTION_WORKERS"); val != "" {
		if w, err := strconv.Atoi(val); err == nil && w > 0 {
			maxWorkers = w
		}
	}

	log.Printf("Worker pool started with %d workers. Listening for submissions...", maxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received termination signal. Shutting down worker pool...")
		cancel()
	}()

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerLoop(ctx, workerID)
		}(i + 1)
	}

	wg.Wait()
	log.Printf("Worker pool stopped cleanly. PEAK ACTIVE SANDBOXES: %d", atomic.LoadInt32(&peakSandboxes))

	// Write to file for benchmark script to read
	os.WriteFile("peak_sandboxes.txt", []byte(fmt.Sprintf("%d", atomic.LoadInt32(&peakSandboxes))), 0666)
}

func workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Blocking pop from Redis queues (submissions_queue, run_queue, problem_setter_queue)
		result, err := db.RedisClient.BRPop(ctx, 1*time.Second, "submissions_queue", "run_queue", "problem_setter_queue").Result()
		if err != nil {
			if err == context.Canceled {
				return
			}
			if err.Error() == "redis: nil" { // Timeout
				continue
			}
			log.Printf("[Worker %d] Error pulling from queue: %v", workerID, err)
			continue
		}

		queueName := result[0]
		payload := result[1]

		switch queueName {
		case "run_queue":
			processRunJob(ctx, workerID, payload)
		case "problem_setter_queue":
			processProblemSetterJob(ctx, workerID, payload)
		default:
			processSubmission(ctx, workerID, payload)
		}
	}
}

func processRunJob(ctx context.Context, workerID int, payload string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker %d] Recovered from panic processing run job: %v\nPayload: %s", workerID, r, payload)
		}
	}()

	var job map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		log.Printf("[Worker %d] Failed to unmarshal run job payload: %v", workerID, err)
		return
	}

	requestID, okReq := job["request_id"].(string)
	code, okCode := job["code"].(string)
	language, okLang := job["language"].(string)
	input, _ := job["input"].(string)

	if !okReq || !okCode || !okLang {
		log.Printf("[Worker %d] Malformed run job payload: %s", workerID, payload)
		return
	}

	acqCtx, cancelAcq := context.WithTimeout(ctx, 10*time.Second)
	defer cancelAcq()

	err := judge.AcquireExecutionToken(acqCtx)
	if err != nil {
		log.Printf("[Worker %d] Semaphore acquisition failed for run job %s: %v", workerID, requestID, err)
		return
	}
	defer judge.ReleaseExecutionToken()

	compRes, err := judge.CompileCode(code, language)
	if err != nil || compRes.Error != "" {
		errMsg := compRes.Error
		if err != nil {
			errMsg = fmt.Sprintf("Sandbox Error: %v", err)
		}
		resJSON, _ := json.Marshal(map[string]string{"output": "", "stderr": errMsg})
		resKey := fmt.Sprintf("run_result:%s", requestID)
		db.RedisClient.LPush(ctx, resKey, resJSON)
		db.RedisClient.Expire(ctx, resKey, 10*time.Second)
		return
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, language)
	if err != nil {
		resJSON, _ := json.Marshal(map[string]string{"output": "", "stderr": fmt.Sprintf("Sandbox Session Error: %v", err)})
		resKey := fmt.Sprintf("run_result:%s", requestID)
		db.RedisClient.LPush(ctx, resKey, resJSON)
		db.RedisClient.Expire(ctx, resKey, 10*time.Second)
		return
	}
	defer session.Close()

	res, err := session.RunTestcase(input)
	stdout, stderr := "", ""
	if err != nil {
		stderr = fmt.Sprintf("Sandbox Error: %v", err)
	} else {
		stdout = res.Stdout
		stderr = res.Stderr
	}

	resJSON, _ := json.Marshal(map[string]string{"output": stdout, "stderr": stderr})
	resKey := fmt.Sprintf("run_result:%s", requestID)
	db.RedisClient.LPush(ctx, resKey, resJSON)
	db.RedisClient.Expire(ctx, resKey, 10*time.Second)
}

func processSubmission(ctx context.Context, workerID int, payload string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker %d] Recovered from panic processing job: %v\nPayload: %s", workerID, r, payload)
		}
	}()

	var job map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		log.Printf("[Worker %d] Failed to unmarshal job payload: %v", workerID, err)
		return
	}

	// Defensive type assertions
	subIDRaw, okSub := job["submission_id"].(float64)
	code, okCode := job["code"].(string)
	language, okLang := job["language"].(string)
	probIDRaw, okProb := job["problem_id"].(float64)

	if !okSub || !okCode || !okLang || !okProb {
		log.Printf("[Worker %d] Malformed job payload missing required fields: %s", workerID, payload)
		if okSub {
			_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", int(subIDRaw))
		}
		return
	}

	submissionID := int(subIDRaw)
	problemID := int(probIDRaw)

	log.Printf("[Worker %d] Processing Submission %d (%s) for Problem %d...", workerID, submissionID, language, problemID)

	// Fetch checker config from problems table
	var checkerType string
	var checkerConfigJSON string
	err := db.DB.QueryRow(
		"SELECT COALESCE(checker_type, 'STANDARD'), COALESCE(checker_config::text, '{}') FROM problems WHERE id = $1",
		problemID,
	).Scan(&checkerType, &checkerConfigJSON)
	if err != nil {
		log.Printf("[Worker %d] Failed to fetch problem config for Problem %d: %v", workerID, problemID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}

	// Load testcases: prefer testcases table, fallback to hidden_testcases JSONB
	testCases, err := loadTestCases(problemID)
	if err != nil {
		log.Printf("[Worker %d] Failed to load test cases for Problem %d: %v", workerID, problemID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}

	if len(testCases) == 0 {
		log.Printf("[Worker %d] No test cases found for Problem %d", workerID, problemID)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}

	acqCtx, cancelAcq := context.WithTimeout(ctx, 5*time.Minute) // Wait up to 5 mins in queue
	defer cancelAcq()

	err = judge.AcquireExecutionToken(acqCtx)
	if err != nil {
		log.Printf("[Worker %d] Dropping Sub %d due to semaphore acquisition failure: %v", workerID, submissionID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}
	
	defer judge.ReleaseExecutionToken() // Guaranteed release even on panic

	// INSTRUMENTATION: Track concurrency
	currentActive := atomic.AddInt32(&activeSandboxes, 1)
	defer atomic.AddInt32(&activeSandboxes, -1)

	for {
		peak := atomic.LoadInt32(&peakSandboxes)
		if currentActive <= peak {
			break
		}
		if atomic.CompareAndSwapInt32(&peakSandboxes, peak, currentActive) {
			break
		}
	}
	log.Printf("[Worker %d] Acquired Token | Current Active Sandboxes: %d | Peak: %d", workerID, currentActive, atomic.LoadInt32(&peakSandboxes))

	// Compile submission code
	compRes, err := judge.CompileCode(code, language)
	if err != nil {
		log.Printf("[Worker %d] Compilation environment error for Sub %d: %v", workerID, submissionID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}
	if compRes.Error != "" {
		log.Printf("[Worker %d] Compilation error for Sub %d: %s", workerID, submissionID, compRes.Error)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "COMPILATION_ERROR", submissionID)
		return
	}

	// Compile custom checker if needed
	var checkerCompRes *judge.CompilationResult
	var checkerConfig judge.CheckerConfig
	if checkerType == "CUSTOM" {
		_ = json.Unmarshal([]byte(checkerConfigJSON), &checkerConfig)
		checkerConfig.Type = judge.CheckerCustom
		if checkerConfig.Code != "" && checkerConfig.Language != "" {
			cRes, cErr := judge.CompileCode(checkerConfig.Code, checkerConfig.Language)
			if cErr != nil || cRes.Error != "" {
				log.Printf("[Worker %d] Custom checker compilation failed for Sub %d", workerID, submissionID)
				_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
				return
			}
			checkerCompRes = cRes
		}
	} else if checkerType == "FLOATING_POINT" {
		_ = json.Unmarshal([]byte(checkerConfigJSON), &checkerConfig)
		checkerConfig.Type = judge.CheckerFloatingPoint
	} else {
		checkerConfig.Type = judge.CheckerStandard
	}

	status := "ACCEPTED"
	
	// Fetch the configured Sandbox Provider
	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, language)
	if err != nil {
		log.Printf("[Worker %d] Failed to start sandbox session for Sub %d: %v", workerID, submissionID, err)
		_, _ = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", "INTERNAL_ERROR", submissionID)
		return
	}
	defer session.Close()

	for i, tc := range testCases {
		res, err := session.RunTestcase(tc.Input)

		if err != nil {
			status = "RUNTIME_ERROR"
			log.Printf("[Worker %d] Execution failed for Sub %d: %v", workerID, submissionID, err)
			break
		} else if res.TimeExceeded {
			status = "TIME_LIMIT_EXCEEDED"
			break
		} else if res.Stderr != "" {
			status = "RUNTIME_ERROR"
			break
		}

		// Run checker based on type
		var checkResult *judge.CheckResult
		switch checkerConfig.Type {
		case judge.CheckerFloatingPoint:
			eps := checkerConfig.Epsilon
			if eps <= 0 {
				eps = 1e-6
			}
			checkResult = judge.RunFloatChecker(tc.ExpectedOutput, res.Stdout, eps)
		case judge.CheckerCustom:
			if checkerCompRes != nil {
				checkResult = runCustomCheckerInSandbox(ctx, provider, checkerCompRes, checkerConfig.Language, tc.Input, tc.ExpectedOutput, res.Stdout)
			} else {
				checkResult = judge.RunStandardChecker(tc.ExpectedOutput, res.Stdout)
			}
		default:
			checkResult = judge.RunStandardChecker(tc.ExpectedOutput, res.Stdout)
		}

		log.Printf("[Worker %d] Sub %d Test %d | Verdict: %s", workerID, submissionID, i+1, checkResult.Verdict)

		if checkResult.Verdict != "ACCEPTED" {
			status = "WRONG_ANSWER"
			log.Printf("[Worker %d] Submission %d failed on test case %d", workerID, submissionID, i+1)
			break
		}
	}

	// Update Database with the verdict
	_, err = db.DB.Exec("UPDATE submissions SET status = $1 WHERE id = $2", status, submissionID)
	if err != nil {
		log.Printf("[Worker %d] Failed to update database for Sub %d: %v", workerID, submissionID, err)
	} else {
		log.Printf("[Worker %d] Submission %d completed with status: %s", workerID, submissionID, status)
		if status == "ACCEPTED" {
			var userID int
			err = db.DB.QueryRow("SELECT user_id FROM submissions WHERE id = $1", submissionID).Scan(&userID)
			if err == nil {
				db.DB.Exec(`
					INSERT INTO user_problem_status (user_id, problem_id, status)
					VALUES ($1, $2, $3)
					ON CONFLICT (user_id, problem_id) DO UPDATE SET status = EXCLUDED.status
				`, userID, problemID, status)
			}
		}
	}
}

// loadTestCases reads testcases from the testcases table first,
// falling back to the legacy hidden_testcases JSONB column.
func loadTestCases(problemID int) ([]TestCase, error) {
	// Try testcases table first
	rows, err := db.DB.Query(
		`SELECT input, COALESCE(expected_output, '') FROM testcases 
		 WHERE problem_id = $1 AND is_sample = FALSE 
		 ORDER BY test_index ASC`, problemID)
	if err == nil {
		defer rows.Close()
		var cases []TestCase
		for rows.Next() {
			var tc TestCase
			if err := rows.Scan(&tc.Input, &tc.ExpectedOutput); err == nil {
				cases = append(cases, tc)
			}
		}
		if len(cases) > 0 {
			return cases, nil
		}
	}

	// Fallback: legacy hidden_testcases JSONB
	var testCasesJSON string
	err = db.DB.QueryRow("SELECT hidden_testcases FROM problems WHERE id = $1", problemID).Scan(&testCasesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch test cases: %v", err)
	}

	var testCases []TestCase
	if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test cases: %v", err)
	}
	return testCases, nil
}

// runCustomCheckerInSandbox executes a custom checker program in the sandbox.
// The checker receives 3 file paths as arguments: input, expected output, contestant output.
// Exit code 0 = ACCEPTED, non-zero = WRONG_ANSWER. Stderr = feedback.
func runCustomCheckerInSandbox(ctx context.Context, provider judge.SandboxProvider, checkerComp *judge.CompilationResult, checkerLang, input, expected, actual string) *judge.CheckResult {
	chkSession, err := provider.StartSession(checkerComp.ArtifactDir, checkerLang)
	if err != nil {
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Checker sandbox error: " + err.Error()}
	}
	defer chkSession.Close()

	// Use type assertion to access Docker-specific methods
	dockerSession, ok := chkSession.(*judge.DockerSandboxSession)
	if !ok {
		// Fallback for non-Docker sandbox: pass combined data via stdin
		combined := fmt.Sprintf("%s\n---SEPARATOR---\n%s\n---SEPARATOR---\n%s", input, expected, actual)
		res, err := chkSession.RunTestcase(combined)
		if err != nil {
			return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Checker error: " + err.Error()}
		}
		if res.ExitCode == 0 {
			return &judge.CheckResult{Verdict: "ACCEPTED", Feedback: strings.TrimSpace(res.Stderr)}
		}
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: strings.TrimSpace(res.Stderr)}
	}

	// Inject 3 files into /tmp
	if err := dockerSession.InjectFile("/tmp/input.txt", input); err != nil {
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Failed to inject input: " + err.Error()}
	}
	if err := dockerSession.InjectFile("/tmp/expected.txt", expected); err != nil {
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Failed to inject expected: " + err.Error()}
	}
	if err := dockerSession.InjectFile("/tmp/actual.txt", actual); err != nil {
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Failed to inject actual: " + err.Error()}
	}

	res, err := dockerSession.RunWithArgs("", []string{"/tmp/input.txt", "/tmp/expected.txt", "/tmp/actual.txt"}, 5.0)
	if err != nil {
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Checker execution error: " + err.Error()}
	}
	if res.TimeExceeded {
		return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: "Checker Time Limit Exceeded"}
	}

	feedback := strings.TrimSpace(res.Stderr)
	if len(feedback) > 1000 {
		feedback = feedback[:1000] + "..."
	}

	if res.ExitCode == 0 {
		return &judge.CheckResult{Verdict: "ACCEPTED", Feedback: feedback}
	}
	return &judge.CheckResult{Verdict: "WRONG_ANSWER", Feedback: feedback}
}

// ============================================
// Problem Setter Job Processor
// ============================================

type ProblemSetterJob struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"` // ps_run, ps_generate, ps_validate, ps_run_solution, ps_run_checker

	// Common fields
	Code     string `json:"code,omitempty"`
	Language string `json:"language,omitempty"`
	Input    string `json:"input,omitempty"`
	Args     []string `json:"args,omitempty"`

	// Generate pipeline fields
	Generator *CodePayload `json:"generator,omitempty"`
	Validator *CodePayload `json:"validator,omitempty"`
	Solution  *CodePayload `json:"solution,omitempty"`
	ArgLines  []string     `json:"arg_lines,omitempty"` // One line per test

	// Checker test fields
	Checker        *CodePayload `json:"checker,omitempty"`
	ExpectedOutput string       `json:"expected_output,omitempty"`
	ActualOutput   string       `json:"actual_output,omitempty"`
}

type CodePayload struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type PSResult struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type GeneratedTestResult struct {
	Index         int    `json:"index"`
	Input         string `json:"input"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	Status        string `json:"status"` // READY, GEN_FAILED, INVALID, SOL_FAILED
	Diagnostic    string `json:"diagnostic,omitempty"`
	Args          string `json:"args"`
}

func processProblemSetterJob(ctx context.Context, workerID int, payload string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker %d][PS] Recovered from panic: %v\nPayload: %s", workerID, r, payload)
		}
	}()

	var job ProblemSetterJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		log.Printf("[Worker %d][PS] Failed to unmarshal job: %v", workerID, err)
		return
	}

	if job.RequestID == "" {
		log.Printf("[Worker %d][PS] Missing request_id in job", workerID)
		return
	}

	var result PSResult

	switch job.Action {
	case "ps_run":
		result = psRun(ctx, workerID, &job)
	case "ps_generate":
		result = psGenerate(ctx, workerID, &job)
	case "ps_validate":
		result = psValidate(ctx, workerID, &job)
	case "ps_run_solution":
		result = psRunSolution(ctx, workerID, &job)
	case "ps_run_checker":
		result = psRunChecker(ctx, workerID, &job)
	default:
		result = PSResult{Success: false, Error: fmt.Sprintf("Unknown action: %s", job.Action)}
	}

	resJSON, _ := json.Marshal(result)
	resKey := fmt.Sprintf("ps_result:%s", job.RequestID)
	db.RedisClient.LPush(ctx, resKey, resJSON)
	db.RedisClient.Expire(ctx, resKey, 60*time.Second)
}

// psRun executes a single program with optional stdin and args
func psRun(ctx context.Context, workerID int, job *ProblemSetterJob) PSResult {
	acqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := judge.AcquireExecutionToken(acqCtx); err != nil {
		return PSResult{Success: false, Error: "Execution capacity full, try again later"}
	}
	defer judge.ReleaseExecutionToken()

	compRes, err := judge.CompileCode(job.Code, job.Language)
	if err != nil {
		return PSResult{Success: false, Error: fmt.Sprintf("Sandbox error: %v", err)}
	}
	if compRes.Error != "" {
		return PSResult{Success: false, Error: compRes.Error}
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, job.Language)
	if err != nil {
		return PSResult{Success: false, Error: fmt.Sprintf("Session error: %v", err)}
	}
	defer session.Close()

	// Use RunWithArgs if available (Docker), else RunTestcase
	var res *judge.SandboxResult
	if dockerSess, ok := session.(*judge.DockerSandboxSession); ok && len(job.Args) > 0 {
		res, err = dockerSess.RunWithArgs(job.Input, job.Args, 10.0)
	} else {
		res, err = session.RunTestcase(job.Input)
	}

	if err != nil {
		return PSResult{Success: false, Error: fmt.Sprintf("Execution error: %v", err)}
	}

	return PSResult{Success: true, Data: map[string]interface{}{
		"stdout":        res.Stdout,
		"stderr":        res.Stderr,
		"time_exceeded": res.TimeExceeded,
		"exit_code":     res.ExitCode,
	}}
}

// psGenerate runs the full generator → validator → reference solution pipeline
func psGenerate(ctx context.Context, workerID int, job *ProblemSetterJob) PSResult {
	if job.Generator == nil || job.Solution == nil {
		return PSResult{Success: false, Error: "Generator and solution are required"}
	}
	if len(job.ArgLines) == 0 {
		return PSResult{Success: false, Error: "At least one argument line is required"}
	}

	acqCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := judge.AcquireExecutionToken(acqCtx); err != nil {
		return PSResult{Success: false, Error: "Execution capacity full"}
	}
	defer judge.ReleaseExecutionToken()

	log.Printf("[Worker %d][PS] Starting generate pipeline: %d tests", workerID, len(job.ArgLines))

	// Compile all programs upfront
	genComp, err := judge.CompileCode(job.Generator.Code, job.Generator.Language)
	if err != nil || genComp.Error != "" {
		errMsg := "Generator compilation failed"
		if genComp != nil && genComp.Error != "" {
			errMsg = genComp.Error
		}
		if err != nil {
			errMsg = err.Error()
		}
		return PSResult{Success: false, Error: errMsg}
	}

	solComp, err := judge.CompileCode(job.Solution.Code, job.Solution.Language)
	if err != nil || solComp.Error != "" {
		errMsg := "Solution compilation failed"
		if solComp != nil && solComp.Error != "" {
			errMsg = solComp.Error
		}
		if err != nil {
			errMsg = err.Error()
		}
		return PSResult{Success: false, Error: errMsg}
	}

	var valComp *judge.CompilationResult
	if job.Validator != nil && job.Validator.Code != "" {
		vc, err := judge.CompileCode(job.Validator.Code, job.Validator.Language)
		if err != nil || vc.Error != "" {
			errMsg := "Validator compilation failed"
			if vc != nil && vc.Error != "" {
				errMsg = vc.Error
			}
			if err != nil {
				errMsg = err.Error()
			}
			return PSResult{Success: false, Error: errMsg}
		}
		valComp = vc
	}

	provider := judge.GetSandboxProvider()
	results := make([]GeneratedTestResult, 0, len(job.ArgLines))

	for i, argLine := range job.ArgLines {
		testResult := GeneratedTestResult{
			Index: i + 1,
			Args:  argLine,
		}

		// Step 1: Run generator
		genSession, err := provider.StartSession(genComp.ArtifactDir, job.Generator.Language)
		if err != nil {
			testResult.Status = "GEN_FAILED"
			testResult.Diagnostic = fmt.Sprintf("Failed to start generator session: %v", err)
			results = append(results, testResult)
			continue
		}

		args := strings.Fields(argLine)
		var genRes *judge.SandboxResult
		if dockerSess, ok := genSession.(*judge.DockerSandboxSession); ok {
			genRes, err = dockerSess.RunWithArgs("", args, 10.0)
		} else {
			// Fallback: pass args via stdin as a single line
			genRes, err = genSession.RunTestcase(argLine)
		}
		genSession.Close()

		if err != nil || genRes.TimeExceeded {
			testResult.Status = "GEN_FAILED"
			diag := "Generator execution failed"
			if genRes != nil && genRes.TimeExceeded {
				diag = "Generator Time Limit Exceeded"
			}
			if err != nil {
				diag = err.Error()
			}
			testResult.Diagnostic = diag
			results = append(results, testResult)
			continue
		}

		input := genRes.Stdout
		testResult.Input = input

		// Step 2: Validate (optional)
		if valComp != nil {
			valSession, err := provider.StartSession(valComp.ArtifactDir, job.Validator.Language)
			if err != nil {
				testResult.Status = "INVALID"
				testResult.Diagnostic = fmt.Sprintf("Failed to start validator session: %v", err)
				results = append(results, testResult)
				continue
			}

			valRes, err := valSession.RunTestcase(input)
			valSession.Close()

			if err != nil || valRes.ExitCode != 0 || valRes.TimeExceeded {
				testResult.Status = "INVALID"
				diag := "Validator rejected input"
				if valRes != nil {
					if valRes.Stderr != "" {
						diag = strings.TrimSpace(valRes.Stderr)
					}
					if valRes.TimeExceeded {
						diag = "Validator Time Limit Exceeded"
					}
				}
				testResult.Diagnostic = diag
				results = append(results, testResult)
				continue
			}
		}

		// Step 3: Run reference solution
		solSession, err := provider.StartSession(solComp.ArtifactDir, job.Solution.Language)
		if err != nil {
			testResult.Status = "SOL_FAILED"
			testResult.Diagnostic = fmt.Sprintf("Failed to start solution session: %v", err)
			results = append(results, testResult)
			continue
		}

		solRes, err := solSession.RunTestcase(input)
		solSession.Close()

		if err != nil || solRes.TimeExceeded || solRes.Stderr != "" {
			testResult.Status = "SOL_FAILED"
			diag := "Reference solution failed"
			if solRes != nil {
				if solRes.TimeExceeded {
					diag = "Reference Solution Time Limit Exceeded"
				} else if solRes.Stderr != "" {
					diag = "Runtime Error: " + strings.TrimSpace(solRes.Stderr)
				}
			}
			if err != nil {
				diag = err.Error()
			}
			testResult.Diagnostic = diag
			results = append(results, testResult)
			continue
		}

		testResult.ExpectedOutput = solRes.Stdout
		testResult.Status = "READY"
		results = append(results, testResult)
		log.Printf("[Worker %d][PS] Test #%d generated successfully", workerID, i+1)
	}

	return PSResult{Success: true, Data: results}
}

// psValidate runs the validator on a single input
func psValidate(ctx context.Context, workerID int, job *ProblemSetterJob) PSResult {
	acqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := judge.AcquireExecutionToken(acqCtx); err != nil {
		return PSResult{Success: false, Error: "Execution capacity full"}
	}
	defer judge.ReleaseExecutionToken()

	compRes, err := judge.CompileCode(job.Code, job.Language)
	if err != nil || compRes.Error != "" {
		errMsg := "Validator compilation failed"
		if compRes != nil && compRes.Error != "" {
			errMsg = compRes.Error
		}
		return PSResult{Success: false, Error: errMsg}
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, job.Language)
	if err != nil {
		return PSResult{Success: false, Error: "Session error: " + err.Error()}
	}
	defer session.Close()

	res, err := session.RunTestcase(job.Input)
	if err != nil {
		return PSResult{Success: false, Error: "Execution error: " + err.Error()}
	}

	if res.TimeExceeded {
		return PSResult{Success: true, Data: map[string]interface{}{
			"valid":      false,
			"diagnostic": "Validator Time Limit Exceeded",
		}}
	}

	valid := res.ExitCode == 0 && res.Stderr == ""
	diag := ""
	if !valid {
		diag = strings.TrimSpace(res.Stderr)
		if diag == "" {
			diag = strings.TrimSpace(res.Stdout)
		}
		if diag == "" {
			diag = "Validator rejected input (non-zero exit code)"
		}
	}

	return PSResult{Success: true, Data: map[string]interface{}{
		"valid":      valid,
		"diagnostic": diag,
		"stdout":     res.Stdout,
	}}
}

// psRunSolution runs a reference solution on given input
func psRunSolution(ctx context.Context, workerID int, job *ProblemSetterJob) PSResult {
	acqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := judge.AcquireExecutionToken(acqCtx); err != nil {
		return PSResult{Success: false, Error: "Execution capacity full"}
	}
	defer judge.ReleaseExecutionToken()

	compRes, err := judge.CompileCode(job.Code, job.Language)
	if err != nil || compRes.Error != "" {
		errMsg := "Solution compilation failed"
		if compRes != nil && compRes.Error != "" {
			errMsg = compRes.Error
		}
		return PSResult{Success: false, Error: errMsg}
	}

	provider := judge.GetSandboxProvider()
	session, err := provider.StartSession(compRes.ArtifactDir, job.Language)
	if err != nil {
		return PSResult{Success: false, Error: "Session error: " + err.Error()}
	}
	defer session.Close()

	res, err := session.RunTestcase(job.Input)
	if err != nil {
		return PSResult{Success: false, Error: "Execution error: " + err.Error()}
	}

	if res.TimeExceeded {
		return PSResult{Success: false, Error: "Reference Solution Time Limit Exceeded"}
	}
	if res.Stderr != "" {
		return PSResult{Success: false, Error: "Runtime Error: " + strings.TrimSpace(res.Stderr)}
	}

	return PSResult{Success: true, Data: map[string]interface{}{
		"output": res.Stdout,
	}}
}

// psRunChecker tests a custom checker with provided inputs
func psRunChecker(ctx context.Context, workerID int, job *ProblemSetterJob) PSResult {
	if job.Checker == nil {
		return PSResult{Success: false, Error: "Checker code is required"}
	}

	acqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := judge.AcquireExecutionToken(acqCtx); err != nil {
		return PSResult{Success: false, Error: "Execution capacity full"}
	}
	defer judge.ReleaseExecutionToken()

	compRes, err := judge.CompileCode(job.Checker.Code, job.Checker.Language)
	if err != nil || compRes.Error != "" {
		errMsg := "Checker compilation failed"
		if compRes != nil && compRes.Error != "" {
			errMsg = compRes.Error
		}
		return PSResult{Success: false, Error: errMsg}
	}

	provider := judge.GetSandboxProvider()
	result := runCustomCheckerInSandbox(ctx, provider, compRes, job.Checker.Language, job.Input, job.ExpectedOutput, job.ActualOutput)

	return PSResult{Success: true, Data: map[string]interface{}{
		"verdict":  result.Verdict,
		"feedback": result.Feedback,
	}}
}
