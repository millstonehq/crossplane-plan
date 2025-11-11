package detector

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Detector extracts PR numbers from Crossplane XRs
type Detector interface {
	// DetectPR returns the PR number if found, or 0 if not found
	DetectPR(xr *unstructured.Unstructured) int
}
