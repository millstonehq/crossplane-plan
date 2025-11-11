package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check defaults
	if cfg.DetectionStrategy != "name" {
		t.Errorf("DetectionStrategy = %s, want name", cfg.DetectionStrategy)
	}

	if cfg.NamePattern != "pr-{number}-*" {
		t.Errorf("NamePattern = %s, want pr-{number}-*", cfg.NamePattern)
	}

	if cfg.LabelKey != "millstone.tech/pr-number" {
		t.Errorf("LabelKey = %s, want millstone.tech/pr-number", cfg.LabelKey)
	}

	if cfg.AnnotationKey != "millstone.tech/preview-pr" {
		t.Errorf("AnnotationKey = %s, want millstone.tech/preview-pr", cfg.AnnotationKey)
	}
}
