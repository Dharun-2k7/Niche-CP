package judge

import (
	"math"
	"strconv"
	"strings"
)

// CheckerType defines the type of output comparison
type CheckerType string

const (
	CheckerStandard      CheckerType = "STANDARD"
	CheckerFloatingPoint CheckerType = "FLOATING_POINT"
	CheckerCustom        CheckerType = "CUSTOM"
)

// CheckerConfig holds the configuration for a problem's checker
type CheckerConfig struct {
	Type     CheckerType `json:"type"`
	Epsilon  float64     `json:"epsilon,omitempty"`  // For FLOATING_POINT
	Code     string      `json:"code,omitempty"`     // For CUSTOM
	Language string      `json:"language,omitempty"` // For CUSTOM
}

// CheckResult holds the result of a checker execution
type CheckResult struct {
	Verdict  string `json:"verdict"`  // "ACCEPTED" or "WRONG_ANSWER"
	Feedback string `json:"feedback"` // Optional diagnostic message
}

// RunStandardChecker performs token-by-token comparison with normalized whitespace.
// This handles trailing newlines, multiple spaces, and trailing whitespace —
// the most common source of false WRONG_ANSWER verdicts on competitive programming platforms.
func RunStandardChecker(expectedOutput, contestantOutput string) *CheckResult {
	expectedTokens := tokenize(expectedOutput)
	actualTokens := tokenize(contestantOutput)

	if len(expectedTokens) != len(actualTokens) {
		return &CheckResult{
			Verdict:  "WRONG_ANSWER",
			Feedback: "",
		}
	}

	for i := range expectedTokens {
		if expectedTokens[i] != actualTokens[i] {
			return &CheckResult{
				Verdict:  "WRONG_ANSWER",
				Feedback: "",
			}
		}
	}

	return &CheckResult{
		Verdict: "ACCEPTED",
	}
}

// RunFloatChecker compares outputs token-by-token, treating numeric tokens
// as floating-point values with epsilon tolerance. Non-numeric tokens are
// compared exactly.
func RunFloatChecker(expectedOutput, contestantOutput string, epsilon float64) *CheckResult {
	expectedTokens := tokenize(expectedOutput)
	actualTokens := tokenize(contestantOutput)

	if len(expectedTokens) != len(actualTokens) {
		return &CheckResult{
			Verdict:  "WRONG_ANSWER",
			Feedback: "",
		}
	}

	for i := range expectedTokens {
		// Try to parse both as floats
		expectedVal, errE := strconv.ParseFloat(expectedTokens[i], 64)
		actualVal, errA := strconv.ParseFloat(actualTokens[i], 64)

		if errE == nil && errA == nil {
			// Both are valid floats — compare with epsilon
			diff := math.Abs(expectedVal - actualVal)
			// Check both absolute and relative error
			absOK := diff <= epsilon
			relOK := false
			if math.Abs(expectedVal) > 1e-9 {
				relOK = diff/math.Abs(expectedVal) <= epsilon
			}
			if !absOK && !relOK {
				return &CheckResult{
					Verdict:  "WRONG_ANSWER",
					Feedback: "",
				}
			}
		} else {
			// Non-numeric tokens — exact comparison
			if expectedTokens[i] != actualTokens[i] {
				return &CheckResult{
					Verdict:  "WRONG_ANSWER",
					Feedback: "",
				}
			}
		}
	}

	return &CheckResult{
		Verdict: "ACCEPTED",
	}
}

// tokenize splits a string into whitespace-delimited tokens.
// This normalizes all whitespace variations (multiple spaces, tabs, trailing newlines).
func tokenize(s string) []string {
	return strings.Fields(strings.TrimSpace(s))
}

// EvaluateVerdict compares the actual stdout against the expected output
// and returns the final submission status. This is the backward-compatible
// entry point used by the worker for problems without checker config.
func EvaluateVerdict(actualOutput, expectedOutput string, timeExceeded bool, stderr string) string {
	if timeExceeded {
		return "TIME_LIMIT_EXCEEDED"
	}

	if stderr != "" {
		return "RUNTIME_ERROR"
	}

	result := RunStandardChecker(expectedOutput, actualOutput)
	return result.Verdict
}
