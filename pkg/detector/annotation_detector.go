package detector

import (
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defaultAnnotationKey = "millstone.tech/preview-pr"
)

// AnnotationDetector extracts PR numbers from XR annotations
type AnnotationDetector struct {
	annotationKey string
}

// NewAnnotationDetector creates an AnnotationDetector with the default annotation key
func NewAnnotationDetector() *AnnotationDetector {
	return &AnnotationDetector{
		annotationKey: defaultAnnotationKey,
	}
}

// NewAnnotationDetectorWithKey creates an AnnotationDetector with a custom annotation key
func NewAnnotationDetectorWithKey(key string) *AnnotationDetector {
	return &AnnotationDetector{
		annotationKey: key,
	}
}

// DetectPR extracts the PR number from XR annotations
func (d *AnnotationDetector) DetectPR(xr *unstructured.Unstructured) int {
	annotations := xr.GetAnnotations()
	if annotations == nil {
		return 0
	}

	prValue, exists := annotations[d.annotationKey]
	if !exists {
		return 0
	}

	prNumber, err := strconv.Atoi(prValue)
	if err != nil {
		return 0
	}

	return prNumber
}

// GetBaseName returns the original name (annotation detector doesn't use name patterns)
func (d *AnnotationDetector) GetBaseName(xr *unstructured.Unstructured) string {
	return xr.GetName()
}
