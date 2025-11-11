package detector

import (
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defaultLabelKey = "millstone.tech/pr-number"
)

// LabelDetector extracts PR numbers from XR labels
type LabelDetector struct {
	labelKey string
}

// NewLabelDetector creates a LabelDetector with the default label key
func NewLabelDetector() *LabelDetector {
	return &LabelDetector{
		labelKey: defaultLabelKey,
	}
}

// NewLabelDetectorWithKey creates a LabelDetector with a custom label key
func NewLabelDetectorWithKey(key string) *LabelDetector {
	return &LabelDetector{
		labelKey: key,
	}
}

// DetectPR extracts the PR number from XR labels
func (d *LabelDetector) DetectPR(xr *unstructured.Unstructured) int {
	labels := xr.GetLabels()
	if labels == nil {
		return 0
	}

	prValue, exists := labels[d.labelKey]
	if !exists {
		return 0
	}

	prNumber, err := strconv.Atoi(prValue)
	if err != nil {
		return 0
	}

	return prNumber
}

// GetBaseName returns the original name (label detector doesn't use name patterns)
func (d *LabelDetector) GetBaseName(xr *unstructured.Unstructured) string {
	return xr.GetName()
}
