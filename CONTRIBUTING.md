# Contributing to crossplane-plan

Thank you for your interest in contributing to crossplane-plan! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Architecture Overview](#architecture-overview)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/crossplane-plan.git
   cd crossplane-plan
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/millstonehq/crossplane-plan.git
   ```

## Development Setup

### Prerequisites

- Go 1.22 or later
- Docker (for building container images)
- Kubernetes cluster (kind, minikube, or similar for testing)
- Crossplane v2.0.0+ installed in your cluster
- kubectl configured to access your cluster

### Install Dependencies

```bash
# Download Go dependencies
go mod download

# Install testing tools (optional)
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

### Building Locally

```bash
# Build the binary
go build -o bin/crossplane-plan ./cmd/crossplane-plan

# Run tests
go test -v ./...

# Run with coverage
go test -cover ./pkg/...

# Build container image locally
docker build -t crossplane-plan:dev .
```

### Running Locally (Development Mode)

```bash
# Run against your kubeconfig context
go run cmd/crossplane-plan/main.go \
  --detection-strategy=name \
  --name-pattern='pr-{number}-*' \
  --github-repo=YOUR_ORG/YOUR_REPO \
  --github-token=$GITHUB_TOKEN
```

## Making Changes

### Branch Naming

Use descriptive branch names with prefixes:
- `feat/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or modifications

Example: `feat/add-gitlab-support`

### Commit Messages

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Maintenance tasks

**Example:**
```
feat(vcs): add GitLab merge request support

Add GitLab client implementation for posting plan comments
to merge requests, following the same pattern as GitHub.

Closes #42
```

## Testing

### Unit Tests

```bash
# Run all tests
go test -v ./...

# Run tests for specific package
go test -v ./pkg/watcher/...

# Run with coverage
go test -cover ./pkg/...

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests

To test against a live Crossplane cluster:

1. Deploy crossplane-plan to your cluster:
   ```bash
   kubectl apply -f deploy/kubernetes/
   ```

2. Create test XRs with preview naming:
   ```bash
   kubectl apply -f examples/test-xr.yaml
   ```

3. Verify plan comments appear in your test PR

### Manual Testing

```bash
# Install in local cluster
kubectl apply -f deploy/kubernetes/

# Create test XR with preview naming
cat <<EOF | kubectl apply -f -
apiVersion: example.org/v1alpha1
kind: XComposite
metadata:
  name: pr-123-test
spec:
  field: value
EOF

# Check logs
kubectl logs -n crossplane-system -l app=crossplane-plan -f

# Verify GitHub comment posted to PR #123
```

## Submitting Changes

### Pull Request Process

1. **Update your fork** with latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your changes** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Open a Pull Request** on GitHub with:
   - Clear title and description
   - Reference to related issues (e.g., "Closes #123")
   - Description of changes and testing performed
   - Screenshots or examples if applicable

4. **Address review feedback** promptly

5. **Ensure CI passes** - all checks must pass before merge

### PR Requirements

- [ ] Code follows project style guidelines
- [ ] Tests added/updated for new functionality
- [ ] Documentation updated (README, examples, etc.)
- [ ] Commit messages follow conventional format
- [ ] No breaking changes (or clearly documented if unavoidable)
- [ ] Coverage meets minimum threshold (40%)

## Code Style

### Go Code

- Follow standard Go conventions and [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` and `goimports` for formatting
- Keep functions small and focused
- Add comments for exported types and functions
- Use meaningful variable names

### Package Structure

```
crossplane-plan/
├── cmd/
│   └── crossplane-plan/     # Main entrypoint
├── pkg/
│   ├── watcher/             # XR watching logic
│   ├── detector/            # PR detection strategies
│   ├── differ/              # Diff calculation
│   ├── vcs/                 # VCS clients (GitHub/GitLab/etc)
│   ├── formatter/           # Output formatting
│   └── config/              # Configuration loading
├── deploy/
│   └── kubernetes/          # Kubernetes manifests
└── examples/                # Example configurations
```

### Configuration

When adding configuration options:

```go
// config/config.go
type Config struct {
    // Detection strategy: name, label, annotation, namespace
    DetectionStrategy string `yaml:"strategy"`
    
    // Pattern for name-based detection (e.g., "pr-{number}-*")
    NamePattern string `yaml:"namePattern"`
    
    // Add documentation for each field
}
```

## Architecture Overview

crossplane-plan consists of five main components:

1. **XR Watcher** (`pkg/watcher/`)
   - Watches Crossplane XRs via Kubernetes API
   - Filters for preview resources
   - Triggers diff calculation on changes

2. **PR Detector** (`pkg/detector/`)
   - Extracts PR number from XR metadata
   - Supports multiple strategies (name/label/annotation/namespace)
   - Maps XRs to pull requests

3. **Diff Calculator** (`pkg/differ/`)
   - Wraps crossplane-diff library
   - Calculates composition-aware diffs
   - Returns full managed resource tree

4. **VCS Client** (`pkg/vcs/`)
   - Posts/updates PR comments
   - Supports GitHub, GitLab, Bitbucket
   - Handles authentication and rate limiting

5. **Formatter** (`pkg/formatter/`)
   - Converts diffs to markdown
   - Generates user-friendly output
   - Handles syntax highlighting

### Adding a New VCS Platform

To add support for a new VCS platform:

1. **Implement the VCS interface** in `pkg/vcs/`:
   ```go
   type Client interface {
       PostComment(ctx context.Context, pr int, comment string) error
       UpdateComment(ctx context.Context, pr int, commentID string, comment string) error
   }
   ```

2. **Add platform-specific client** (e.g., `pkg/vcs/gitlab/`):
   ```go
   type GitLabClient struct {
       token string
       repo  string
   }
   
   func (c *GitLabClient) PostComment(ctx context.Context, pr int, comment string) error {
       // Implementation
   }
   ```

3. **Update configuration** to support new platform:
   ```yaml
   vcs:
     platform: gitlab  # github, gitlab, bitbucket
     repo: org/repo
     token: $GITLAB_TOKEN
   ```

4. **Add tests** for the new client

5. **Update documentation** with examples

### Adding a New Detection Strategy

To add a new detection strategy:

1. **Implement the Detector interface** in `pkg/detector/`:
   ```go
   type Detector interface {
       DetectPR(xr *unstructured.Unstructured) (int, error)
   }
   ```

2. **Add strategy implementation**:
   ```go
   type CustomDetector struct {
       pattern string
   }
   
   func (d *CustomDetector) DetectPR(xr *unstructured.Unstructured) (int, error) {
       // Implementation
   }
   ```

3. **Register in detector factory**

4. **Add configuration support**

5. **Add tests and documentation**

## Questions?

- Open a [Discussion](https://github.com/millstonehq/crossplane-plan/discussions) for general questions
- Create an [Issue](https://github.com/millstonehq/crossplane-plan/issues) for bugs or feature requests
- Check existing documentation in the [README](README.md)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
