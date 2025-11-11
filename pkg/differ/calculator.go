package differ

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/crossplane-contrib/crossplane-diff/cmd/diff/client/core"
	xp "github.com/crossplane-contrib/crossplane-diff/cmd/diff/client/crossplane"
	k8 "github.com/crossplane-contrib/crossplane-diff/cmd/diff/client/kubernetes"
	"github.com/crossplane-contrib/crossplane-diff/cmd/diff/diffprocessor"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

// DiffResult represents the structured diff output
type DiffResult struct {
	// XR is the Composite Resource being diffed
	XR *unstructured.Unstructured

	// RawDiff is the raw diff output from crossplane-diff
	RawDiff string

	// HasChanges indicates if there are any changes
	HasChanges bool

	// Summary provides a high-level summary of changes
	Summary string
}

// Calculator uses crossplane-diff library to calculate diffs
type Calculator struct {
	config      *rest.Config
	logger      logging.Logger
	k8sClients  k8.Clients
	xpClients   xp.Clients
	processor   diffprocessor.DiffProcessor
	initialized bool
}

// NewCalculator creates a new Calculator
func NewCalculator(config *rest.Config, logger logging.Logger) *Calculator {
	return &Calculator{
		config: config,
		logger: logger,
	}
}

// Initialize sets up the Kubernetes and Crossplane clients
func (c *Calculator) Initialize(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	// Create core clients
	coreClients, err := core.NewClients(c.config)
	if err != nil {
		return fmt.Errorf("failed to create core clients: %w", err)
	}

	// Create type converter
	tc := k8.NewTypeConverter(coreClients, c.logger)

	// Create K8s clients
	c.k8sClients = k8.Clients{
		Type:     tc,
		Apply:    k8.NewApplyClient(coreClients, tc, c.logger),
		Resource: k8.NewResourceClient(coreClients, tc, c.logger),
		Schema:   k8.NewSchemaClient(coreClients, tc, c.logger),
	}

	// Create Crossplane clients
	defClient := xp.NewDefinitionClient(c.k8sClients.Resource, c.logger)
	c.xpClients = xp.Clients{
		Definition:   defClient,
		Composition:  xp.NewCompositionClient(c.k8sClients.Resource, defClient, c.logger),
		Environment:  xp.NewEnvironmentClient(c.k8sClients.Resource, c.logger),
		Function:     xp.NewFunctionClient(c.k8sClients.Resource, c.logger),
		ResourceTree: xp.NewResourceTreeClient(coreClients.Tree, c.logger),
	}

	// Initialize Crossplane clients
	if err := c.xpClients.Initialize(ctx, c.logger); err != nil {
		return fmt.Errorf("failed to initialize crossplane clients: %w", err)
	}

	// Create diff processor
	c.processor = diffprocessor.NewDiffProcessor(
		c.k8sClients,
		c.xpClients,
		diffprocessor.WithLogger(c.logger),
		diffprocessor.WithNamespace("default"),
		diffprocessor.WithColorize(false), // No colors for structured output
		diffprocessor.WithCompact(false),
	)

	// Initialize processor
	if err := c.processor.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize diff processor: %w", err)
	}

	c.initialized = true
	return nil
}

// CalculateDiff calculates the diff for an XR using crossplane-diff library
func (c *Calculator) CalculateDiff(ctx context.Context, xr *unstructured.Unstructured) (*DiffResult, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize calculator: %w", err)
		}
	}

	// Use a buffer to capture diff output
	var buf bytes.Buffer

	// Perform diff - PerformDiff writes to io.Writer
	resources := []*unstructured.Unstructured{xr}
	err := c.processor.PerformDiff(ctx, &buf, resources, c.xpClients.Composition.FindMatchingComposition)
	
	diffOutput := buf.String()
	hasChanges := len(strings.TrimSpace(diffOutput)) > 0

	if err != nil {
		return nil, fmt.Errorf("failed to calculate diff: %w", err)
	}

	return &DiffResult{
		XR:         xr,
		RawDiff:    diffOutput,
		HasChanges: hasChanges,
		Summary:    c.generateSummary(xr, diffOutput, hasChanges),
	}, nil
}

// generateSummary creates a high-level summary of the diff
func (c *Calculator) generateSummary(xr *unstructured.Unstructured, diff string, hasChanges bool) string {
	if !hasChanges {
		return fmt.Sprintf("No changes detected for %s/%s", xr.GetKind(), xr.GetName())
	}

	// Count additions and deletions
	additions := 0
	deletions := 0
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "+") {
			additions++
		} else if strings.HasPrefix(line, "-") {
			deletions++
		}
	}

	return fmt.Sprintf("Changes detected for %s/%s: +%d -%d lines",
		xr.GetKind(), xr.GetName(), additions, deletions)
}
