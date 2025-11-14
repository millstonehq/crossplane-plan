#!/usr/bin/env bash
set -euo pipefail

# Coverage gate for business logic (excluding infrastructure components)
# Target: 70% coverage (realistic for codebase with API wrappers)
#
# Pure business logic (config, detector, formatter): ~99% coverage
# API wrappers (vcs/github methods): ~0% (integration-testable only)
# Infrastructure (differ, watcher): excluded

COVERAGE_THRESHOLD=40.0  # TODO: Increase to 70% after adding more tests

# Run tests with coverage
echo "Running tests with coverage..."
CGO_ENABLED=0 go test -coverprofile=coverage.out ./pkg/...

# Filter out excluded packages (differ and watcher)
echo "Filtering coverage data..."
grep -v "pkg/differ/" coverage.out | grep -v "pkg/watcher/" > coverage-filtered.out || {
    # If grep finds nothing, create file with just mode line
    echo "mode: set" > coverage-filtered.out
}

# Calculate coverage percentage for filtered packages
echo "Calculating coverage..."
COVERAGE=$(go tool cover -func=coverage-filtered.out | tail -1 | awk '{print $3}' | sed 's/%//')

echo ""
echo "========================================="
echo "Coverage Report (Business Logic Only)"
echo "========================================="
echo ""
go tool cover -func=coverage-filtered.out | grep -E "total:|config/|detector/|formatter/|vcs/"
echo ""
echo "========================================="
echo "Excluded from coverage: pkg/differ/, pkg/watcher/"
echo "These require integration tests"
echo "========================================="
echo "Total Coverage: ${COVERAGE}%"
echo "Threshold:      ${COVERAGE_THRESHOLD}%"
echo "========================================="

# Check if coverage meets threshold
if (( $(echo "$COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
    echo "✅ Coverage check PASSED"
    exit 0
else
    echo "❌ Coverage check FAILED"
    echo "Coverage ${COVERAGE}% is below threshold ${COVERAGE_THRESHOLD}%"
    exit 1
fi
