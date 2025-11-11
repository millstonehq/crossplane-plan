package detector

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestLabelDetector_DetectPR(t *testing.T) {
	tests := []struct {
		name       string
		labels     map[string]string
		expectedPR int
	}{
		{
			name: "valid PR number in label",
			labels: map[string]string{
				"millstone.tech/pr-number": "123",
			},
			expectedPR: 123,
		},
		{
			name: "no PR label",
			labels: map[string]string{
				"app": "test",
			},
			expectedPR: 0,
		},
		{
			name:       "nil labels",
			labels:     nil,
			expectedPR: 0,
		},
		{
			name: "invalid PR number",
			labels: map[string]string{
				"millstone.tech/pr-number": "abc",
			},
			expectedPR: 0,
		},
		{
			name: "large PR number",
			labels: map[string]string{
				"millstone.tech/pr-number": "99999",
			},
			expectedPR: 99999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewLabelDetector()
			xr := &unstructured.Unstructured{}
			if tt.labels != nil {
				xr.SetLabels(tt.labels)
			}

			got := detector.DetectPR(xr)
			if got != tt.expectedPR {
				t.Errorf("DetectPR() = %d, want %d", got, tt.expectedPR)
			}
		})
	}
}

func TestLabelDetectorWithKey_DetectPR(t *testing.T) {
	customKey := "custom.io/pr"
	detector := NewLabelDetectorWithKey(customKey)

	xr := &unstructured.Unstructured{}
	xr.SetLabels(map[string]string{
		customKey: "456",
	})

	got := detector.DetectPR(xr)
	if got != 456 {
		t.Errorf("DetectPR() with custom key = %d, want 456", got)
	}
}
