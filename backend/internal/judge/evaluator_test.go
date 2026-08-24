package judge

import (
	"testing"
)

func TestStandardChecker(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		want     string
	}{
		{
			name:     "Exact match",
			expected: "1 2 3\n4 5 6",
			actual:   "1 2 3\n4 5 6\n",
			want:     "ACCEPTED",
		},
		{
			name:     "Whitespace variations",
			expected: "1   2   3\n",
			actual:   "1 2 3 \n",
			want:     "ACCEPTED",
		},
		{
			name:     "Mismatched tokens",
			expected: "1 2 3",
			actual:   "1 2 4",
			want:     "WRONG_ANSWER",
		},
		{
			name:     "Extra token",
			expected: "1 2",
			actual:   "1 2 3",
			want:     "WRONG_ANSWER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RunStandardChecker(tt.expected, tt.actual)
			if res.Verdict != tt.want {
				t.Errorf("RunStandardChecker() = %v, want %v", res.Verdict, tt.want)
			}
		})
	}
}

func TestFloatChecker(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		eps      float64
		want     string
	}{
		{
			name:     "Within absolute epsilon",
			expected: "3.14159265",
			actual:   "3.14159200",
			eps:      1e-5,
			want:     "ACCEPTED",
		},
		{
			name:     "Outside epsilon",
			expected: "3.14159265",
			actual:   "3.14000000",
			eps:      1e-5,
			want:     "WRONG_ANSWER",
		},
		{
			name:     "Mixed text and float",
			expected: "Case #1: 2.500",
			actual:   "Case #1: 2.5000001",
			eps:      1e-5,
			want:     "ACCEPTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RunFloatChecker(tt.expected, tt.actual, tt.eps)
			if res.Verdict != tt.want {
				t.Errorf("RunFloatChecker() = %v, want %v", res.Verdict, tt.want)
			}
		})
	}
}
