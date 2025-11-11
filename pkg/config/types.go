package config

// Config holds the application configuration
type Config struct {
	// DetectionStrategy defines how to extract PR numbers from XRs
	// Supported values: "name", "label", "annotation"
	DetectionStrategy string

	// NamePattern is the pattern used for name-based detection
	// Example: "pr-{number}-*" matches "pr-123-mill"
	NamePattern string

	// GitHubRepo is the target repository for posting comments
	// Format: "owner/repo"
	GitHubRepo string

	// DryRun mode calculates diffs but doesn't post to GitHub
	DryRun bool

	// LabelKey is the label key for label-based detection
	// Default: "millstone.tech/pr-number"
	LabelKey string

	// AnnotationKey is the annotation key for annotation-based detection
	// Default: "millstone.tech/preview-pr"
	AnnotationKey string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		DetectionStrategy: "name",
		NamePattern:       "pr-{number}-*",
		LabelKey:          "millstone.tech/pr-number",
		AnnotationKey:     "millstone.tech/preview-pr",
	}
}
