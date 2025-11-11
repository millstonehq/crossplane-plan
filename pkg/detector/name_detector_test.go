package detector

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNameDetector_DetectPR(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		xrName     string
		expectedPR int
	}{
		{
			name:       "matches pr-123-mill",
			pattern:    "pr-{number}-*",
			xrName:     "pr-123-mill",
			expectedPR: 123,
		},
		{
			name:       "matches pr-456-books",
			pattern:    "pr-{number}-*",
			xrName:     "pr-456-books",
			expectedPR: 456,
		},
		{
			name:       "no match - missing prefix",
			pattern:    "pr-{number}-*",
			xrName:     "mill",
			expectedPR: 0,
		},
		{
			name:       "no match - invalid number",
			pattern:    "pr-{number}-*",
			xrName:     "pr-abc-mill",
			expectedPR: 0,
		},
		{
			name:       "custom pattern",
			pattern:    "preview-{number}-*",
			xrName:     "preview-789-test",
			expectedPR: 789,
		},
		{
			name:       "multiple digits",
			pattern:    "pr-{number}-*",
			xrName:     "pr-12345-app",
			expectedPR: 12345,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewNameDetector(tt.pattern)
			xr := &unstructured.Unstructured{}
			xr.SetName(tt.xrName)

			got := detector.DetectPR(xr)
			if got != tt.expectedPR {
				t.Errorf("DetectPR() = %d, want %d", got, tt.expectedPR)
			}
		})
	}
}
