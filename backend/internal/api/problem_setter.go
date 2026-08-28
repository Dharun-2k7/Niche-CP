package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Dharun-2k7/online-coding-platform/internal/db"
	"github.com/gin-gonic/gin"
)

// CodeConfig represents stored code configuration (generator, validator, solution, checker)
type CodeConfig struct {
	Code     string  `json:"code"`
	Language string  `json:"language"`
	Epsilon  float64 `json:"epsilon,omitempty"` // For floating point checker
}

// ProblemConfigPayload represents the full configuration payload for a problem
type ProblemConfigPayload struct {
	TimeLimitMs     int             `json:"time_limit_ms"`
	MemoryLimitMb   int             `json:"memory_limit_mb"`
	CheckerType     string          `json:"checker_type"`
	CheckerConfig   CodeConfig      `json:"checker_config"`
	GeneratorConfig GeneratorConfig `json:"generator_config"`
	ValidatorConfig CodeConfig      `json:"validator_config"`
	SolutionConfig  CodeConfig      `json:"solution_config"`
}

// SaveProblemConfig updates the problem setter configuration for a problem
func SaveProblemConfig(c *gin.Context) {
	idStr := c.Param("id")
	probID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID: " + idStr})
		return
	}

	var req ProblemConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	if req.TimeLimitMs <= 0 {
		req.TimeLimitMs = 2000
	}
	if req.MemoryLimitMb <= 0 {
		req.MemoryLimitMb = 256
	}
	if req.CheckerType == "" {
		req.CheckerType = "STANDARD"
	}

	chkJSON, _ := json.Marshal(req.CheckerConfig)
	genJSON, _ := json.Marshal(req.GeneratorConfig)
	valJSON, _ := json.Marshal(req.ValidatorConfig)
	solJSON, _ := json.Marshal(req.SolutionConfig)

	res, err := db.DB.Exec(`
		UPDATE problems 
		SET time_limit_ms = $1, memory_limit_mb = $2, checker_type = $3,
		    checker_config = $4::jsonb, generator_config = $5::jsonb,
		    validator_config = $6::jsonb, solution_config = $7::jsonb,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
	`, req.TimeLimitMs, req.MemoryLimitMb, req.CheckerType, string(chkJSON), string(genJSON), string(valJSON), string(solJSON), probID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save configuration: " + err.Error()})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem setter configuration saved successfully"})
}

// GetProblemConfig retrieves the problem setter configuration
func GetProblemConfig(c *gin.Context) {
	idStr := c.Param("id")
	probID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID: " + idStr})
		return
	}

	var status, checkerType string
	var timeLimitMs, memoryLimitMb int
	var chkStr, genStr, valStr, solStr string

	err = db.DB.QueryRow(`
		SELECT COALESCE(status, 'DRAFT'), COALESCE(time_limit_ms, 2000), COALESCE(memory_limit_mb, 256),
		       COALESCE(checker_type, 'STANDARD'), COALESCE(checker_config::text, '{}'),
		       COALESCE(generator_config::text, '{}'), COALESCE(validator_config::text, '{}'),
		       COALESCE(solution_config::text, '{}')
		FROM problems WHERE id = $1
	`, probID).Scan(&status, &timeLimitMs, &memoryLimitMb, &checkerType, &chkStr, &genStr, &valStr, &solStr)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed: " + err.Error()})
		return
	}

	var chkConfig, valConfig, solConfig CodeConfig
	var genConfig GeneratorConfig
	_ = json.Unmarshal([]byte(chkStr), &chkConfig)
	_ = json.Unmarshal([]byte(genStr), &genConfig)
	_ = json.Unmarshal([]byte(valStr), &valConfig)
	_ = json.Unmarshal([]byte(solStr), &solConfig)

	c.JSON(http.StatusOK, gin.H{
		"id":               probID,
		"status":           status,
		"time_limit_ms":    timeLimitMs,
		"memory_limit_mb":  memoryLimitMb,
		"checker_type":     checkerType,
		"checker_config":   chkConfig,
		"generator_config": genConfig,
		"validator_config": valConfig,
		"solution_config":  solConfig,
	})
}

// GenerateTestsRequest defines the payload for invoking the generator pipeline
type GenerateTestsRequest struct {
	ArgLines   []string         `json:"arg_lines"`           // Advanced mode: one line per test
	Generator  *GeneratorConfig `json:"generator,omitempty"` // Optional simple or advanced override
	Validator  *CodeConfig      `json:"validator,omitempty"` // Optional override
	Solution   *CodeConfig      `json:"solution,omitempty"`  // Optional override
	ReplaceAll bool             `json:"replace_all"`         // If true, delete existing generated testcases first
}

// GenerateTests delegates generation to the Redis worker queue and persists testcases
func GenerateTests(c *gin.Context) {
	idStr := c.Param("id")
	probID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	var req GenerateTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load DB configs if overrides not provided
	var genStr, valStr, solStr string
	err = db.DB.QueryRow(`
		SELECT COALESCE(generator_config::text, '{}'), COALESCE(validator_config::text, '{}'), COALESCE(solution_config::text, '{}')
		FROM problems WHERE id = $1
	`, probID).Scan(&genStr, &valStr, &solStr)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	genConfig := req.Generator
	if genConfig == nil {
		var cfg GeneratorConfig
		_ = json.Unmarshal([]byte(genStr), &cfg)
		genConfig = &cfg
	}
	resolvedGenerator, simpleArgs, genErr := genConfig.resolve()
	if genErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": genErr.Error()})
		return
	}
	if genConfig.Mode == "simple" {
		req.ArgLines = simpleArgs
	} else if len(req.ArgLines) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "add at least one test plan entry"})
		return
	}

	valConfig := req.Validator
	if valConfig == nil || valConfig.Code == "" {
		var cfg CodeConfig
		_ = json.Unmarshal([]byte(valStr), &cfg)
		if cfg.Code != "" {
			valConfig = &cfg
		} else {
			valConfig = nil
		}
	}

	solConfig := req.Solution
	if solConfig == nil || solConfig.Code == "" {
		var cfg CodeConfig
		_ = json.Unmarshal([]byte(solStr), &cfg)
		solConfig = &cfg
	}

	if solConfig.Code == "" || solConfig.Language == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reference solution code and language are required"})
		return
	}

	requestID := fmt.Sprintf("ps_gen_%d_%d", time.Now().UnixNano(), c.GetInt("user_id"))

	jobPayload := map[string]interface{}{
		"request_id": requestID,
		"action":     "ps_generate",
		"generator":  resolvedGenerator,
		"solution":   solConfig,
		"arg_lines":  req.ArgLines,
	}
	if valConfig != nil && valConfig.Code != "" {
		jobPayload["validator"] = valConfig
	}

	jobData, _ := json.Marshal(jobPayload)

	// LPush to Redis queue
	ctx, cancel := context.WithTimeout(c.Request.Context(), 180*time.Second)
	defer cancel()

	if err := db.RedisClient.LPush(ctx, "problem_setter_queue", jobData).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue generation job: " + err.Error()})
		return
	}

	resKey := fmt.Sprintf("ps_result:%s", requestID)
	popResult, err := db.RedisClient.BRPop(ctx, 170*time.Second, resKey).Result()
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Generation request timed out or worker is busy"})
		return
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
		Data    []struct {
			Index          int    `json:"index"`
			Input          string `json:"input"`
			ExpectedOutput string `json:"expected_output"`
			Status         string `json:"status"`
			Diagnostic     string `json:"diagnostic"`
			Args           string `json:"args"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(popResult[1]), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse worker response"})
		return
	}

	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	// Persist generated testcases to DB
	if req.ReplaceAll {
		_, _ = db.DB.Exec("DELETE FROM testcases WHERE problem_id = $1 AND source = 'generated'", probID)
	}

	// Determine starting test_index
	var maxIdx sql.NullInt64
	_ = db.DB.QueryRow("SELECT MAX(test_index) FROM testcases WHERE problem_id = $1", probID).Scan(&maxIdx)
	startIdx := 1
	if maxIdx.Valid {
		startIdx = int(maxIdx.Int64) + 1
	}

	savedCount := 0
	for _, tc := range result.Data {
		if tc.Status == "READY" {
			valStatus := "valid"
			if valConfig != nil {
				valStatus = "valid"
			} else {
				valStatus = "pending"
			}

			_, insertErr := db.DB.Exec(`
				INSERT INTO testcases (problem_id, test_index, source, generator_args, input, expected_output, is_sample, validation_status, generation_status)
				VALUES ($1, $2, 'generated', $3, $4, $5, FALSE, $6, 'ready')
				ON CONFLICT (problem_id, test_index) DO UPDATE 
				SET input = EXCLUDED.input, expected_output = EXCLUDED.expected_output, generator_args = EXCLUDED.generator_args, updated_at = CURRENT_TIMESTAMP
			`, probID, startIdx, tc.Args, tc.Input, tc.ExpectedOutput, valStatus)

			if insertErr == nil {
				savedCount++
				startIdx++
			}
		}
	}

	// Count failed tests (status != READY)
	failedCount := 0
	for _, tc := range result.Data {
		if tc.Status != "READY" {
			failedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("Pipeline finished: %d testcases created/updated", savedCount),
		"saved_count":  savedCount,
		"failed_count": failedCount,
		"results":      result.Data,
	})
}

// ValidateTestInput validates a testcase input using the problem's validator
func ValidateTestInput(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID: " + probIDStr})
		return
	}

	var req struct {
		Input   string `json:"input" binding:"required"`
		ValCode string `json:"val_code"`
		ValLang string `json:"val_lang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load validator from DB if not provided
	if req.ValCode == "" {
		var valStr string
		err := db.DB.QueryRow("SELECT COALESCE(validator_config::text, '{}') FROM problems WHERE id = $1", probID).Scan(&valStr)
		if err == nil {
			var cfg CodeConfig
			_ = json.Unmarshal([]byte(valStr), &cfg)
			req.ValCode = cfg.Code
			req.ValLang = cfg.Language
		}
	}

	if req.ValCode == "" || req.ValLang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validator code and language are required"})
		return
	}

	requestID := fmt.Sprintf("ps_val_%d_%d", time.Now().UnixNano(), c.GetInt("user_id"))

	jobData, _ := json.Marshal(map[string]interface{}{
		"request_id": requestID,
		"action":     "ps_validate",
		"code":       req.ValCode,
		"language":   req.ValLang,
		"input":      req.Input,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := db.RedisClient.LPush(ctx, "problem_setter_queue", jobData).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue validation job"})
		return
	}

	resKey := fmt.Sprintf("ps_result:%s", requestID)
	popResult, err := db.RedisClient.BRPop(ctx, 25*time.Second, resKey).Result()
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Validation request timed out"})
		return
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			Valid      bool   `json:"valid"`
			Diagnostic string `json:"diagnostic"`
			Stdout     string `json:"stdout"`
		} `json:"data"`
	}

	_ = json.Unmarshal([]byte(popResult[1]), &result)

	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      result.Data.Valid,
		"diagnostic": result.Data.Diagnostic,
		"stdout":     result.Data.Stdout,
	})
}

// RunReferenceSolution executes the reference solution on input
func RunReferenceSolution(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID: " + probIDStr})
		return
	}

	var req struct {
		Input   string `json:"input"`
		SolCode string `json:"sol_code"`
		SolLang string `json:"sol_lang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SolCode == "" {
		var solStr string
		err := db.DB.QueryRow("SELECT COALESCE(solution_config::text, '{}') FROM problems WHERE id = $1", probID).Scan(&solStr)
		if err == nil {
			var cfg CodeConfig
			_ = json.Unmarshal([]byte(solStr), &cfg)
			req.SolCode = cfg.Code
			req.SolLang = cfg.Language
		}
	}

	if req.SolCode == "" || req.SolLang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solution code and language are required"})
		return
	}

	requestID := fmt.Sprintf("ps_sol_%d_%d", time.Now().UnixNano(), c.GetInt("user_id"))

	jobData, _ := json.Marshal(map[string]interface{}{
		"request_id": requestID,
		"action":     "ps_run_solution",
		"code":       req.SolCode,
		"language":   req.SolLang,
		"input":      req.Input,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := db.RedisClient.LPush(ctx, "problem_setter_queue", jobData).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue solution job"})
		return
	}

	resKey := fmt.Sprintf("ps_result:%s", requestID)
	popResult, err := db.RedisClient.BRPop(ctx, 25*time.Second, resKey).Result()
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Solution execution timed out"})
		return
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			Output string `json:"output"`
		} `json:"data"`
	}

	_ = json.Unmarshal([]byte(popResult[1]), &result)

	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output": result.Data.Output,
	})
}

// RunCheckerTest tests a custom checker against test values
func RunCheckerTest(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID: " + probIDStr})
		return
	}

	var req struct {
		Input          string `json:"input"`
		ExpectedOutput string `json:"expected_output"`
		ActualOutput   string `json:"actual_output"`
		ChkCode        string `json:"chk_code"`
		ChkLang        string `json:"chk_lang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ChkCode == "" {
		var chkStr string
		err := db.DB.QueryRow("SELECT COALESCE(checker_config::text, '{}') FROM problems WHERE id = $1", probID).Scan(&chkStr)
		if err == nil {
			var cfg CodeConfig
			_ = json.Unmarshal([]byte(chkStr), &cfg)
			req.ChkCode = cfg.Code
			req.ChkLang = cfg.Language
		}
	}

	if req.ChkCode == "" || req.ChkLang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Checker code and language are required"})
		return
	}

	requestID := fmt.Sprintf("ps_chk_%d_%d", time.Now().UnixNano(), c.GetInt("user_id"))

	jobData, _ := json.Marshal(map[string]interface{}{
		"request_id": requestID,
		"action":     "ps_run_checker",
		"checker": map[string]string{
			"code":     req.ChkCode,
			"language": req.ChkLang,
		},
		"input":           req.Input,
		"expected_output": req.ExpectedOutput,
		"actual_output":   req.ActualOutput,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := db.RedisClient.LPush(ctx, "problem_setter_queue", jobData).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue checker job"})
		return
	}

	resKey := fmt.Sprintf("ps_result:%s", requestID)
	popResult, err := db.RedisClient.BRPop(ctx, 25*time.Second, resKey).Result()
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Checker execution timed out"})
		return
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			Verdict  string `json:"verdict"`
			Feedback string `json:"feedback"`
		} `json:"data"`
	}

	_ = json.Unmarshal([]byte(popResult[1]), &result)

	if !result.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verdict":  result.Data.Verdict,
		"feedback": result.Data.Feedback,
	})
}

// CreateManualTestcase adds a manually authored testcase
func CreateManualTestcase(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	var req struct {
		Input          string `json:"input" binding:"required"`
		ExpectedOutput string `json:"expected_output"`
		IsSample       bool   `json:"is_sample"`
		GeneratorArgs  string `json:"generator_args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var maxIdx sql.NullInt64
	_ = db.DB.QueryRow("SELECT MAX(test_index) FROM testcases WHERE problem_id = $1", probID).Scan(&maxIdx)
	nextIdx := 1
	if maxIdx.Valid {
		nextIdx = int(maxIdx.Int64) + 1
	}

	var testID int
	err = db.DB.QueryRow(`
		INSERT INTO testcases (problem_id, test_index, source, generator_args, input, expected_output, is_sample, validation_status, generation_status)
		VALUES ($1, $2, 'manual', $3, $4, $5, $6, 'pending', 'ready')
		RETURNING id
	`, probID, nextIdx, req.GeneratorArgs, req.Input, req.ExpectedOutput, req.IsSample).Scan(&testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert testcase: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Testcase created successfully",
		"id":         testID,
		"test_index": nextIdx,
	})
}

// ListTestcases returns all testcases for a problem
func ListTestcases(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, test_index, source, COALESCE(generator_args, ''), input, COALESCE(expected_output, ''),
		       is_sample, COALESCE(validation_status, 'pending'), COALESCE(validation_message, ''),
		       COALESCE(generation_status, 'ready'), created_at
		FROM testcases WHERE problem_id = $1 ORDER BY test_index ASC
	`, probID)

	if err == nil {
		defer rows.Close()
		testcases := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id, testIndex int
			var source, args, input, expected, valStatus, valMsg, genStatus string
			var isSample bool
			var createdAt time.Time
			if scanErr := rows.Scan(&id, &testIndex, &source, &args, &input, &expected, &isSample, &valStatus, &valMsg, &genStatus, &createdAt); scanErr == nil {
				testcases = append(testcases, map[string]interface{}{
					"id":                 id,
					"test_index":         testIndex,
					"source":             source,
					"generator_args":     args,
					"input":              input,
					"expected_output":    expected,
					"is_sample":          isSample,
					"validation_status":  valStatus,
					"validation_message": valMsg,
					"generation_status":  genStatus,
					"created_at":         createdAt,
				})
			}
		}

		if len(testcases) > 0 {
			c.JSON(http.StatusOK, testcases)
			return
		}
	}

	// Fallback to hidden_testcases JSONB for legacy problems
	var hiddenJSON string
	_ = db.DB.QueryRow("SELECT hidden_testcases FROM problems WHERE id = $1", probID).Scan(&hiddenJSON)

	var legacyCases []struct {
		Input          string `json:"input"`
		ExpectedOutput string `json:"expected_output"`
	}
	_ = json.Unmarshal([]byte(hiddenJSON), &legacyCases)

	fallbackCases := make([]map[string]interface{}, 0)
	for i, tc := range legacyCases {
		fallbackCases = append(fallbackCases, map[string]interface{}{
			"id":                -(i + 1), // Negative ID indicates legacy JSONB source
			"test_index":        i + 1,
			"source":            "legacy",
			"input":             tc.Input,
			"expected_output":   tc.ExpectedOutput,
			"is_sample":         false,
			"validation_status": "valid",
		})
	}

	c.JSON(http.StatusOK, fallbackCases)
}

// UpdateTestcase updates an existing testcase
func UpdateTestcase(c *gin.Context) {
	tid := c.Param("tid")

	var req struct {
		Input          string `json:"input"`
		ExpectedOutput string `json:"expected_output"`
		IsSample       bool   `json:"is_sample"`
		GeneratorArgs  string `json:"generator_args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := db.DB.Exec(`
		UPDATE testcases 
		SET input = $1, expected_output = $2, is_sample = $3, generator_args = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`, req.Input, req.ExpectedOutput, req.IsSample, req.GeneratorArgs, tid)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update testcase"})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Testcase not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Testcase updated successfully"})
}

// DeleteTestcase removes a testcase and re-indexes remaining testcases
func DeleteTestcase(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID: " + probIDStr})
		return
	}
	tid := c.Param("tid")

	res, err := db.DB.Exec("DELETE FROM testcases WHERE id = $1 AND problem_id = $2", tid, probID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete testcase: " + err.Error()})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Testcase not found"})
		return
	}

	// Re-index remaining testcases
	_, _ = db.DB.Exec(`
		WITH reindexed AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY test_index ASC) as new_index
			FROM testcases WHERE problem_id = $1
		)
		UPDATE testcases SET test_index = reindexed.new_index
		FROM reindexed WHERE testcases.id = reindexed.id
	`, probID)

	c.JSON(http.StatusOK, gin.H{"message": "Testcase deleted and remaining testcases re-indexed"})
}

// ChecklistItem represents a single readiness requirement
type ChecklistItem struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Passed     bool   `json:"passed"`
	Diagnostic string `json:"diagnostic"`
}

// ReviewProblem evaluates the problem against automated readiness requirements
func ReviewProblem(c *gin.Context) {
	idStr := c.Param("id")
	probID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	var title, description, inputFmt, outputFmt, constraints, status, checkerType string
	var timeLimitMs, memoryLimitMb int
	var solStr, chkStr string

	err = db.DB.QueryRow(`
		SELECT title, description, COALESCE(input_format, ''), COALESCE(output_format, ''),
		       COALESCE(constraints, ''), COALESCE(status, 'DRAFT'), COALESCE(time_limit_ms, 2000),
		       COALESCE(memory_limit_mb, 256), COALESCE(checker_type, 'STANDARD'),
		       COALESCE(solution_config::text, '{}'), COALESCE(checker_config::text, '{}')
		FROM problems WHERE id = $1
	`, probID).Scan(&title, &description, &inputFmt, &outputFmt, &constraints, &status, &timeLimitMs, &memoryLimitMb, &checkerType, &solStr, &chkStr)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	checklist := make([]ChecklistItem, 0)
	allPassed := true

	// 1. Statement & Description
	hasDesc := len(strings.TrimSpace(description)) >= 20
	diagDesc := "Description is set (" + strconv.Itoa(len(description)) + " chars)"
	if !hasDesc {
		diagDesc = "Description is missing or too short (minimum 20 characters)"
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "description", Label: "Problem Statement", Passed: hasDesc, Diagnostic: diagDesc})

	// 2. Input Format
	hasInputFmt := len(strings.TrimSpace(inputFmt)) > 0
	diagInput := "Input format specified"
	if !hasInputFmt {
		diagInput = "Input format specification is empty"
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "input_format", Label: "Input Format", Passed: hasInputFmt, Diagnostic: diagInput})

	// 3. Output Format
	hasOutputFmt := len(strings.TrimSpace(outputFmt)) > 0
	diagOutput := "Output format specified"
	if !hasOutputFmt {
		diagOutput = "Output format specification is empty"
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "output_format", Label: "Output Format", Passed: hasOutputFmt, Diagnostic: diagOutput})

	// 4. Constraints
	hasConstraints := len(strings.TrimSpace(constraints)) > 0
	diagConstraints := "Constraints specified"
	if !hasConstraints {
		diagConstraints = "Constraints specification is empty"
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "constraints", Label: "Constraints", Passed: hasConstraints, Diagnostic: diagConstraints})

	// 5. Reference Solution
	var solConfig CodeConfig
	_ = json.Unmarshal([]byte(solStr), &solConfig)
	hasSolution := len(strings.TrimSpace(solConfig.Code)) > 0 && solConfig.Language != ""
	diagSol := "Reference solution present (" + solConfig.Language + ")"
	if !hasSolution {
		diagSol = "Reference solution code or language is missing"
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "reference_solution", Label: "Reference Solution", Passed: hasSolution, Diagnostic: diagSol})

	// 6. Testcases Count
	var tcCount int
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM testcases WHERE problem_id = $1", probID).Scan(&tcCount)
	if tcCount == 0 {
		// Fallback check legacy hidden_testcases
		var hiddenStr string
		_ = db.DB.QueryRow("SELECT hidden_testcases FROM problems WHERE id = $1", probID).Scan(&hiddenStr)
		var legacy []interface{}
		_ = json.Unmarshal([]byte(hiddenStr), &legacy)
		tcCount = len(legacy)
	}
	hasTestcases := tcCount >= 1
	diagTc := fmt.Sprintf("%d testcase(s) configured", tcCount)
	if !hasTestcases {
		diagTc = "At least 1 testcase is required before publication"
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "testcases", Label: "Testcases Quantity", Passed: hasTestcases, Diagnostic: diagTc})

	// 7. Expected Outputs Present
	var missingOutputs int
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM testcases WHERE problem_id = $1 AND (expected_output IS NULL OR expected_output = '')", probID).Scan(&missingOutputs)
	hasAllOutputs := missingOutputs == 0
	diagOutputs := "All testcases have expected outputs"
	if !hasAllOutputs {
		diagOutputs = fmt.Sprintf("%d testcase(s) are missing expected output from reference solution", missingOutputs)
		allPassed = false
	}
	checklist = append(checklist, ChecklistItem{Key: "expected_outputs", Label: "Expected Outputs", Passed: hasAllOutputs, Diagnostic: diagOutputs})

	// 8. Checker Configuration
	checkerValid := true
	diagChecker := "Checker type: " + checkerType
	if checkerType == "CUSTOM" {
		var chkConfig CodeConfig
		_ = json.Unmarshal([]byte(chkStr), &chkConfig)
		if strings.TrimSpace(chkConfig.Code) == "" || chkConfig.Language == "" {
			checkerValid = false
			diagChecker = "Custom checker code or language is missing"
			allPassed = false
		}
	} else if checkerType == "FLOATING_POINT" {
		var chkConfig CodeConfig
		_ = json.Unmarshal([]byte(chkStr), &chkConfig)
		if chkConfig.Epsilon <= 0 {
			diagChecker = "Floating point checker epsilon defaulting to 1e-6"
		}
	}
	checklist = append(checklist, ChecklistItem{Key: "checker", Label: "Checker Configuration", Passed: checkerValid, Diagnostic: diagChecker})

	// 9. Time & Memory Limits
	limitsValid := timeLimitMs > 0 && memoryLimitMb > 0
	diagLimits := fmt.Sprintf("Time Limit: %dms, Memory Limit: %dMB", timeLimitMs, memoryLimitMb)
	checklist = append(checklist, ChecklistItem{Key: "limits", Label: "Execution Limits", Passed: limitsValid, Diagnostic: diagLimits})

	// Update status if passed and currently DRAFT
	if allPassed && status == "DRAFT" {
		_, _ = db.DB.Exec("UPDATE problems SET status = 'READY_FOR_REVIEW' WHERE id = $1", probID)
		status = "READY_FOR_REVIEW"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      status,
		"can_publish": allPassed,
		"checklist":   checklist,
	})
}

// PublishProblem transitions problem status to PUBLISHED and syncs JSONB testcases for legacy compatibility
func PublishProblem(c *gin.Context) {
	probIDStr := c.Param("id")
	probID, err := strconv.Atoi(probIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	// 1. Sync sample and hidden testcases into JSONB columns for backwards compatibility
	rows, err := db.DB.Query(`
		SELECT input, COALESCE(expected_output, ''), is_sample
		FROM testcases WHERE problem_id = $1 ORDER BY test_index ASC
	`, probID)

	if err == nil {
		defer rows.Close()
		samples := make([]map[string]string, 0)
		hiddens := make([]map[string]string, 0)

		for rows.Next() {
			var inp, exp string
			var isSample bool
			if scanErr := rows.Scan(&inp, &exp, &isSample); scanErr == nil {
				item := map[string]string{"input": inp, "expected_output": exp}
				if isSample {
					samples = append(samples, item)
				}
				hiddens = append(hiddens, item)
			}
		}

		if len(hiddens) > 0 {
			sampleJSON, _ := json.Marshal(samples)
			hiddenJSON, _ := json.Marshal(hiddens)
			_, _ = db.DB.Exec(`
				UPDATE problems SET sample_testcases = $1::jsonb, hidden_testcases = $2::jsonb WHERE id = $3
			`, string(sampleJSON), string(hiddenJSON), probID)
		}
	}

	// 2. Set status to PUBLISHED
	res, err := db.DB.Exec("UPDATE problems SET status = 'PUBLISHED', updated_at = CURRENT_TIMESTAMP WHERE id = $1", probID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish problem: " + err.Error()})
		return
	}

	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Problem published successfully! It is now live for platform participants.",
		"status":  "PUBLISHED",
	})
}
