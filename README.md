# crossplane-plan

> Terraform plan for Crossplane - Preview infrastructure changes in pull requests

## Overview

`crossplane-plan` watches Crossplane Composite Resources (XRs) and automatically posts infrastructure change previews to GitHub pull requests, similar to how Terraform Cloud provides plan comments.

## Architecture

- **XR Watcher**: Monitors XRs via Kubernetes watch API
- **PR Detector**: Extracts PR number from XR name/labels/annotations
- **Diff Calculator**: Uses [crossplane-diff](https://github.com/crossplane-contrib/crossplane-diff) library for accurate composition rendering
- **VCS Client**: Posts formatted diffs to GitHub/GitLab/Bitbucket

## Detection Strategies

### Name-based (Default)
Extracts PR number from XR name using pattern: `pr-{number}-*`
```yaml
# Example: pr-123-mill → PR #123
apiVersion: platform.millstone.tech/v1alpha1
kind: XGitHubRepository
metadata:
  name: pr-123-mill
```

### Label-based
Reads PR number from label: `millstone.tech/pr-number: "123"`

### Annotation-based
Reads PR number from annotation: `millstone.tech/preview-pr: "123"`

## Usage

### Deployment
```bash
# Build and push image
earthly +docker --tag=v0.1.0

# Deploy to Kubernetes
kubectl apply -f deploy/kubernetes/
```

### Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: crossplane-plan-config
data:
  detection-strategy: name
  name-pattern: pr-{number}-*
  github-repo: millstonehq/mill
```

## Development

### Prerequisites
- Go 1.22+
- Kubernetes cluster with Crossplane installed
- GitHub App credentials for posting comments

### Local Development
```bash
# Install dependencies
go mod download

# Run locally (requires kubeconfig)
go run cmd/crossplane-plan/main.go \
  --detection-strategy=name \
  --name-pattern='pr-{number}-*' \
  --github-repo=millstonehq/mill
```

### Testing
```bash
# Run unit tests
go test ./...

# Run with verbose output
go test -v ./...
```

## How It Works

1. **Watch XRs**: Monitors all Crossplane XRs in the cluster
2. **Detect PR**: Extracts PR number using configured strategy
3. **Calculate Diff**: Uses crossplane-diff to render full composition tree
4. **Format Output**: Generates markdown-formatted diff
5. **Post Comment**: Creates/updates GitHub PR comment with preview

## Why crossplane-diff Library?

- **Maintained by community**: crossplane-contrib maintains the complex logic
- **Complete rendering**: Shows XR + all managed resources
- **Accurate composition**: Uses actual Crossplane composition engine
- **Handles edge cases**: Nested XRs, composition changes, schema validation
- **Provider-agnostic**: Works with ANY Crossplane provider

## Roadmap

- [x] Phase 1: Name-based detection, GitHub integration
- [ ] Phase 2: Label/annotation detection strategies
- [ ] Phase 3: GitLab and Bitbucket support
- [ ] Phase 4: Open source release

## License

Apache 2.0

