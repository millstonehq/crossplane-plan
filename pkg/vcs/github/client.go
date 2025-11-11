package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

const (
	// CommentIdentifier is used to identify crossplane-plan comments
	CommentIdentifier = "<!-- crossplane-plan-comment -->"
)

// Client is a GitHub API client for posting PR comments
type Client struct {
	client *github.Client
	owner  string
	repo   string
}

// ClientConfig holds authentication configuration for GitHub
type ClientConfig struct {
	// Token-based authentication (PAT or OAuth token)
	Token string

	// GitHub App authentication
	AppID          string // GitHub App ID
	InstallationID string // Installation ID for the app
	PrivateKey     []byte // Private key for the GitHub App

	// Crossplane provider credentials format (JSON)
	// This is base64-encoded JSON in the format used by crossplane-provider-github
	Credentials string

	// Repository (required)
	Repository string // Format: owner/repo
}

// crossplaneProviderCredentials represents the JSON structure used by crossplane-provider-github
type crossplaneProviderCredentials struct {
	AppAuth []struct {
		ID             string `json:"id"`
		InstallationID string `json:"installation_id"`
		PemFile        string `json:"pem_file"`
	} `json:"app_auth"`
	Owner string `json:"owner"`
}

// NewClient creates a new GitHub client with either token or GitHub App authentication
func NewClient(token, repository string) (*Client, error) {
	return NewClientFromConfig(&ClientConfig{
		Token:      token,
		Repository: repository,
	})
}

// NewClientFromConfig creates a new GitHub client from configuration
// Supports multiple authentication methods:
// 1. Token authentication (PAT or OAuth)
// 2. Crossplane provider credentials format (base64-encoded JSON)
// 3. GitHub App authentication (direct credentials)
func NewClientFromConfig(config *ClientConfig) (*Client, error) {
	// Parse repository (format: owner/repo)
	parts := strings.Split(config.Repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s (expected owner/repo)", config.Repository)
	}
	owner, repo := parts[0], parts[1]

	var httpClient *http.Client

	// Determine authentication method (in priority order)
	if config.Token != "" {
		// Token-based authentication (PAT or OAuth)
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: config.Token},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	} else if config.Credentials != "" {
		// Crossplane provider credentials format (base64-encoded JSON)
		client, err := createClientFromCrossplaneCredentials(config.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to parse crossplane credentials: %w", err)
		}
		httpClient = client
	} else if config.AppID != "" && config.InstallationID != "" && len(config.PrivateKey) > 0 {
		// GitHub App authentication (direct credentials)
		client, err := createClientFromGitHubApp(config.AppID, config.InstallationID, config.PrivateKey)
		if err != nil {
			return nil, err
		}
		httpClient = client
	} else {
		return nil, fmt.Errorf("no valid authentication provided: either token, credentials, or GitHub App credentials (appID, installationID, privateKey) required")
	}

	return &Client{
		client: github.NewClient(httpClient),
		owner:  owner,
		repo:   repo,
	}, nil
}

// createClientFromCrossplaneCredentials parses crossplane provider credentials and creates HTTP client
func createClientFromCrossplaneCredentials(credentialsB64 string) (*http.Client, error) {
	// Decode base64
	credentialsJSON, err := base64.StdEncoding.DecodeString(credentialsB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 credentials: %w", err)
	}

	// Parse JSON
	var creds crossplaneProviderCredentials
	if err := json.Unmarshal(credentialsJSON, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials JSON: %w", err)
	}

	// Validate
	if len(creds.AppAuth) == 0 {
		return nil, fmt.Errorf("no app_auth entries found in credentials")
	}

	appAuth := creds.AppAuth[0]
	if appAuth.ID == "" || appAuth.InstallationID == "" || appAuth.PemFile == "" {
		return nil, fmt.Errorf("incomplete app_auth credentials")
	}

	// Create GitHub App client
	return createClientFromGitHubApp(appAuth.ID, appAuth.InstallationID, []byte(appAuth.PemFile))
}

// createClientFromGitHubApp creates an HTTP client using GitHub App credentials
func createClientFromGitHubApp(appID, installationID string, privateKey []byte) (*http.Client, error) {
	appIDInt, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub App ID: %w", err)
	}

	installationIDInt, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid installation ID: %w", err)
	}

	// Create GitHub App transport
	itr, err := ghinstallation.New(http.DefaultTransport, appIDInt, installationIDInt, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub App transport: %w", err)
	}

	return &http.Client{Transport: itr}, nil
}

// PostComment posts or updates a comment on a PR
// If a crossplane-plan comment already exists, it updates it; otherwise creates a new one
func (c *Client) PostComment(ctx context.Context, prNumber int, body string) error {
	// Add identifier to comment body
	commentBody := CommentIdentifier + "\n\n" + body

	// Find existing crossplane-plan comment
	existingCommentID, err := c.findExistingComment(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to find existing comment: %w", err)
	}

	if existingCommentID != nil {
		// Update existing comment
		comment := &github.IssueComment{
			Body: &commentBody,
		}
		_, _, err := c.client.Issues.EditComment(ctx, c.owner, c.repo, *existingCommentID, comment)
		if err != nil {
			return fmt.Errorf("failed to update comment: %w", err)
		}
		return nil
	}

	// Create new comment
	comment := &github.IssueComment{
		Body: &commentBody,
	}
	_, _, err = c.client.Issues.CreateComment(ctx, c.owner, c.repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// findExistingComment finds an existing crossplane-plan comment on the PR
func (c *Client) findExistingComment(ctx context.Context, prNumber int) (*int64, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, c.owner, c.repo, prNumber, opts)
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.HasPrefix(*comment.Body, CommentIdentifier) {
				return comment.ID, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil, nil
}

// DeleteComment deletes a crossplane-plan comment from a PR
func (c *Client) DeleteComment(ctx context.Context, prNumber int) error {
	commentID, err := c.findExistingComment(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to find existing comment: %w", err)
	}

	if commentID == nil {
		// No comment to delete
		return nil
	}

	_, err = c.client.Issues.DeleteComment(ctx, c.owner, c.repo, *commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	return nil
}
