package detector

import (
	"regexp"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NameDetector extracts PR numbers from XR names using a pattern
type NameDetector struct {
	pattern *regexp.Regexp
}

// NewNameDetector creates a NameDetector from a pattern string
// Pattern format: "pr-{number}-*" where {number} is replaced with (\d+)
func NewNameDetector(pattern string) *NameDetector {
	// Convert pattern to regex
	// "pr-{number}-*" becomes "^pr-(\d+)-.*$"
	regexPattern := regexp.MustCompile(`\{number\}`).ReplaceAllString(pattern, `(\d+)`)
	regexPattern = regexp.MustCompile(`\*`).ReplaceAllString(regexPattern, `.*`)
	regexPattern = "^" + regexPattern + "$"

	return &NameDetector{
		pattern: regexp.MustCompile(regexPattern),
	}
}

// DetectPR extracts the PR number from the XR name
func (d *NameDetector) DetectPR(xr *unstructured.Unstructured) int {
	name := xr.GetName()
	matches := d.pattern.FindStringSubmatch(name)

	if len(matches) < 2 {
		return 0
	}

	prNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}

	return prNumber
}

// GetBaseName strips the PR prefix from an XR name to get the production resource name
// Example: "pr-2-mill" -> "mill"
func (d *NameDetector) GetBaseName(xr *unstructured.Unstructured) string {
	name := xr.GetName()
	matches := d.pattern.FindStringSubmatch(name)

	if len(matches) < 2 {
		// Not a PR XR, return original name
		return name
	}

	// Pattern format: "pr-{number}-*" becomes "^pr-(\d+)-(.*)$"
	// matches[0] = full match (pr-2-mill)
	// matches[1] = PR number (2)
	// matches[2] = base name (mill)
	if len(matches) >= 3 {
		return matches[2]
	}

	return name
}
