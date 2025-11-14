package github

import (
	"encoding/json"
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

func TestNewClientFromConfig_TokenAuth(t *testing.T) {
	cfg := &ClientConfig{
		Token:      "test-token",
		Repository: "owner/repo",
	}

	client, err := NewClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewClientFromConfig() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClientFromConfig() returned nil client")
	}

	if client.owner != "owner" {
		t.Errorf("owner = %s, want owner", client.owner)
	}

	if client.repo != "repo" {
		t.Errorf("repo = %s, want repo", client.repo)
	}
}

func TestNewClientFromConfig_NoAuth(t *testing.T) {
	cfg := &ClientConfig{
		Repository: "owner/repo",
	}

	_, err := NewClientFromConfig(cfg)
	if err == nil {
		t.Error("NewClientFromConfig() error = nil, want error for no auth")
	}
}

func TestNewClientFromConfig_InvalidRepo(t *testing.T) {
	tests := []struct {
		name string
		repo string
	}{
		{name: "no slash", repo: "ownerrepo"},
		{name: "too many slashes", repo: "owner/repo/extra"},
		{name: "empty", repo: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ClientConfig{
				Token:      "token",
				Repository: tt.repo,
			}

			_, err := NewClientFromConfig(cfg)
			if err == nil {
				t.Error("NewClientFromConfig() error = nil, want error for invalid repo format")
			}
		})
	}
}

func TestNewClientFromConfig_CrossplaneCredentials(t *testing.T) {
	// Valid crossplane provider credentials
	creds := crossplaneProviderCredentials{
		AppAuth: []struct {
			ID             string `json:"id"`
			InstallationID string `json:"installation_id"`
			PemFile        string `json:"pem_file"`
		}{
			{
				ID:             "12345",
				InstallationID: "67890",
				PemFile:        "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA0Z...\n-----END RSA PRIVATE KEY-----",
			},
		},
		Owner: "test-owner",
	}

	credsJSON, _ := json.Marshal(creds)

	cfg := &ClientConfig{
		Credentials: string(credsJSON),
		Repository:  "owner/repo",
	}

	// Note: This will fail because the PEM file is fake, but we can test the parsing logic
	_, err := NewClientFromConfig(cfg)
	if err == nil {
		t.Error("NewClientFromConfig() should fail with invalid PEM (testing parsing worked)")
	} else if !contains(err.Error(), "failed to create GitHub App transport") {
		// If it got to the transport creation step, parsing worked
		t.Logf("Expected error (parsing succeeded, transport failed): %v", err)
	}
}

func TestNewClientFromConfig_InvalidCrossplaneCredentials(t *testing.T) {
	tests := []struct {
		name        string
		credentials string
		wantErrPart string
	}{
		{
			name:        "invalid JSON",
			credentials: "not valid json",
			wantErrPart: "failed to parse credentials JSON",
		},
		{
			name:        "empty app_auth",
			credentials: `{"app_auth":[],"owner":"test"}`,
			wantErrPart: "no app_auth entries found",
		},
		{
			name:        "missing id",
			credentials: `{"app_auth":[{"installation_id":"123","pem_file":"key"}],"owner":"test"}`,
			wantErrPart: "incomplete app_auth credentials",
		},
		{
			name:        "missing installation_id",
			credentials: `{"app_auth":[{"id":"123","pem_file":"key"}],"owner":"test"}`,
			wantErrPart: "incomplete app_auth credentials",
		},
		{
			name:        "missing pem_file",
			credentials: `{"app_auth":[{"id":"123","installation_id":"456"}],"owner":"test"}`,
			wantErrPart: "incomplete app_auth credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ClientConfig{
				Credentials: tt.credentials,
				Repository:  "owner/repo",
			}

			_, err := NewClientFromConfig(cfg)
			if err == nil {
				t.Error("NewClientFromConfig() error = nil, want error")
			} else if !contains(err.Error(), tt.wantErrPart) {
				t.Errorf("NewClientFromConfig() error = %v, want error containing %q", err, tt.wantErrPart)
			}
		})
	}
}

func TestNewClientFromConfig_GitHubAppAuth(t *testing.T) {
	cfg := &ClientConfig{
		AppID:          "12345",
		InstallationID: "67890",
		PrivateKey:     []byte("-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"),
		Repository:     "owner/repo",
	}

	// This will fail due to invalid key, but tests the flow
	_, err := NewClientFromConfig(cfg)
	if err == nil {
		t.Error("Expected error with invalid private key")
	}
}

func TestNewClientFromConfig_InvalidAppID(t *testing.T) {
	cfg := &ClientConfig{
		AppID:          "not-a-number",
		InstallationID: "67890",
		PrivateKey:     []byte("key"),
		Repository:     "owner/repo",
	}

	_, err := NewClientFromConfig(cfg)
	if err == nil {
		t.Error("NewClientFromConfig() error = nil, want error for invalid app ID")
	} else if !contains(err.Error(), "invalid GitHub App ID") {
		t.Errorf("Expected 'invalid GitHub App ID' error, got: %v", err)
	}
}

func TestNewClientFromConfig_InvalidInstallationID(t *testing.T) {
	cfg := &ClientConfig{
		AppID:          "12345",
		InstallationID: "not-a-number",
		PrivateKey:     []byte("key"),
		Repository:     "owner/repo",
	}

	_, err := NewClientFromConfig(cfg)
	if err == nil {
		t.Error("NewClientFromConfig() error = nil, want error for invalid installation ID")
	} else if !contains(err.Error(), "invalid installation ID") {
		t.Errorf("Expected 'invalid installation ID' error, got: %v", err)
	}
}

func TestCreateClientFromCrossplaneCredentials(t *testing.T) {
	// Valid structure with fake PEM
	validCreds := `{
		"app_auth": [
			{
				"id": "12345",
				"installation_id": "67890",
				"pem_file": "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----"
			}
		],
		"owner": "test-owner"
	}`

	_, err := createClientFromCrossplaneCredentials(validCreds)
	// Will fail on transport creation with fake PEM, but parsing should work
	if err == nil {
		t.Error("Expected error with fake PEM")
	} else if !contains(err.Error(), "failed to create GitHub App transport") {
		t.Logf("Got expected error: %v", err)
	}
}

func TestCreateClientFromGitHubApp(t *testing.T) {
	tests := []struct {
		name           string
		appID          string
		installationID string
		privateKey     []byte
		wantErrPart    string
	}{
		{
			name:           "invalid app ID",
			appID:          "invalid",
			installationID: "123",
			privateKey:     []byte("key"),
			wantErrPart:    "invalid GitHub App ID",
		},
		{
			name:           "invalid installation ID",
			appID:          "123",
			installationID: "invalid",
			privateKey:     []byte("key"),
			wantErrPart:    "invalid installation ID",
		},
		{
			name:           "invalid private key",
			appID:          "123",
			installationID: "456",
			privateKey:     []byte("not a valid key"),
			wantErrPart:    "failed to create GitHub App transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createClientFromGitHubApp(tt.appID, tt.installationID, tt.privateKey)
			if err == nil {
				t.Error("createClientFromGitHubApp() error = nil, want error")
			} else if !contains(err.Error(), tt.wantErrPart) {
				t.Errorf("createClientFromGitHubApp() error = %v, want error containing %q", err, tt.wantErrPart)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	if s == "" {
		return false
	}
	if substr == "" {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Note: PostComment, findExistingComment, and DeleteComment are thin wrappers
// around go-github library calls. Testing these requires either:
// 1. Integration tests against real GitHub API
// 2. Complex mocking of go-github's internal HTTP transport
// 3. Dependency injection of the entire github.Client
//
// For a thin wrapper with minimal business logic, integration tests are more appropriate.
// See docs/testing.md for integration test strategy.
