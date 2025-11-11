package github

import (
	"testing"
)

func TestNewClient_ValidRepo(t *testing.T) {
	client, err := NewClient("test-token", "owner/repo")
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.owner != "owner" {
		t.Errorf("owner = %s, want owner", client.owner)
	}

	if client.repo != "repo" {
		t.Errorf("repo = %s, want repo", client.repo)
	}
}

func TestNewClient_InvalidRepo(t *testing.T) {
	tests := []struct {
		name string
		repo string
	}{
		{
			name: "missing slash",
			repo: "ownerrepo",
		},
		{
			name: "too many slashes",
			repo: "owner/repo/extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient("token", tt.repo)
			if err == nil {
				t.Error("NewClient() error = nil, want error")
			}
		})
	}
}

func TestCommentIdentifier(t *testing.T) {
	expected := "<!-- crossplane-plan-comment -->"
	if CommentIdentifier != expected {
		t.Errorf("CommentIdentifier = %q, want %q", CommentIdentifier, expected)
	}
}

// Note: PostComment, findExistingComment, and DeleteComment are thin wrappers
// around go-github library calls. Testing these requires either:
// 1. Integration tests against real GitHub API
// 2. Complex mocking of go-github's internal HTTP transport
// 3. Dependency injection of the entire github.Client
//
// For a thin wrapper with minimal business logic, integration tests are more appropriate.
// See docs/testing.md for integration test strategy.
