package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/millstonehq/crossplane-plan/pkg/detector"
	"github.com/millstonehq/crossplane-plan/pkg/differ"
	"github.com/millstonehq/crossplane-plan/pkg/formatter"
	"github.com/millstonehq/crossplane-plan/pkg/vcs/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// XRWatcher watches Crossplane Composite Resources and posts diffs to GitHub
type XRWatcher struct {
	clientset              *kubernetes.Clientset
	dynamicClient          dynamic.Interface
	detector               detector.Detector
	differ                 *differ.Calculator
	formatter              *formatter.GitHubFormatter
	vcsClient              *github.Client
	logger                 logr.Logger
	processedXRs           map[string]string // name -> resource version
	reconciliationInterval int               // minutes
}

// NewXRWatcher creates a new XRWatcher
func NewXRWatcher(
	clientset *kubernetes.Clientset,
	detector detector.Detector,
	differ *differ.Calculator,
	formatter *formatter.GitHubFormatter,
	vcsClient *github.Client,
	logger logr.Logger,
	reconciliationInterval int,
) *XRWatcher {
	// Create dynamic client from the same config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Try building from kubeconfig if in-cluster fails
		panic(fmt.Sprintf("failed to get kubernetes config: %v", err))
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create dynamic client: %v", err))
	}

	return &XRWatcher{
		clientset:              clientset,
		dynamicClient:          dynamicClient,
		detector:               detector,
		differ:                 differ,
		formatter:              formatter,
		vcsClient:              vcsClient,
		logger:                 logger,
		processedXRs:           make(map[string]string),
		reconciliationInterval: reconciliationInterval,
	}
}

// Start begins watching Crossplane XRs
func (w *XRWatcher) Start(ctx context.Context) error {
	w.logger.Info("Starting XR watcher")

	// Discover Crossplane XRD GVRs
	gvrs, err := w.discoverXRDGVRs(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover XRDs: %w", err)
	}

	w.logger.Info("Discovered XRDs", "count", len(gvrs))

	// Initial reconciliation - process existing PR XRs
	w.logger.Info("Starting initial reconciliation of existing PR XRs")
	for _, gvr := range gvrs {
		if err := w.reconcileExistingXRs(ctx, gvr); err != nil {
			w.logger.Error(err, "failed initial reconciliation", "gvr", gvr.String())
			// Don't fail startup, just log and continue
		}
	}
	w.logger.Info("Initial reconciliation complete")

	// Watch each GVR for changes
	for _, gvr := range gvrs {
		go w.watchGVR(ctx, gvr)
	}

	// Start periodic reconciliation if enabled
	if w.reconciliationInterval > 0 {
		ticker := time.NewTicker(time.Duration(w.reconciliationInterval) * time.Minute)
		defer ticker.Stop()

		w.logger.Info("Starting periodic reconciliation", "interval", fmt.Sprintf("%dm", w.reconciliationInterval))

		go func() {
			for {
				select {
				case <-ticker.C:
					w.logger.Info("Running periodic reconciliation")
					for _, gvr := range gvrs {
						if err := w.reconcileExistingXRs(ctx, gvr); err != nil {
							w.logger.Error(err, "periodic reconciliation failed", "gvr", gvr.String())
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Block until context is cancelled
	<-ctx.Done()
	return nil
}

// discoverXRDGVRs discovers all Crossplane XRDs in the cluster
func (w *XRWatcher) discoverXRDGVRs(ctx context.Context) ([]schema.GroupVersionResource, error) {
	// XRDs are defined by apiextensions.crossplane.io/v1 CompositeResourceDefinition
	xrdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.crossplane.io",
		Version:  "v1",
		Resource: "compositeresourcedefinitions",
	}

	xrds, err := w.dynamicClient.Resource(xrdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list XRDs: %w", err)
	}

	var gvrs []schema.GroupVersionResource
	for _, xrd := range xrds.Items {
		// Extract group from spec.group
		group, found, err := unstructured.NestedString(xrd.Object, "spec", "group")
		if err != nil || !found {
			w.logger.Error(err, "failed to get group from XRD", "name", xrd.GetName())
			continue
		}

		// Extract plural from spec.names.plural
		plural, found, err := unstructured.NestedString(xrd.Object, "spec", "names", "plural")
		if err != nil || !found {
			w.logger.Error(err, "failed to get plural from XRD", "name", xrd.GetName())
			continue
		}

		// Get served versions from spec.versions
		versions, found, err := unstructured.NestedSlice(xrd.Object, "spec", "versions")
		if err != nil || !found {
			w.logger.Error(err, "failed to get versions from XRD", "name", xrd.GetName())
			continue
		}

		// Find first served+referenceable version
		for _, v := range versions {
			versionMap, ok := v.(map[string]interface{})
			if !ok {
				continue
			}

			served, _, _ := unstructured.NestedBool(versionMap, "served")
			referenceable, _, _ := unstructured.NestedBool(versionMap, "referenceable")
			versionName, _, _ := unstructured.NestedString(versionMap, "name")

			if served && referenceable && versionName != "" {
				gvrs = append(gvrs, schema.GroupVersionResource{
					Group:    group,
					Version:  versionName,
					Resource: plural,
				})
				break
			}
		}
	}

	return gvrs, nil
}

// reconcileExistingXRs performs initial reconciliation of existing XRs for a GVR
func (w *XRWatcher) reconcileExistingXRs(ctx context.Context, gvr schema.GroupVersionResource) error {
	list, err := w.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	w.logger.Info("Checking for existing PR XRs", "gvr", gvr.String(), "totalCount", len(list.Items))

	// Group XRs by PR number
	prXRs := make(map[int][]*unstructured.Unstructured)
	for _, item := range list.Items {
		xr := item.DeepCopy()

		// Only process PR XRs
		prNumber := w.detector.DetectPR(xr)
		if prNumber == 0 {
			continue
		}

		prXRs[prNumber] = append(prXRs[prNumber], xr)
	}

	// Process each PR's XRs as a batch
	for prNumber, xrs := range prXRs {
		w.logger.Info("Reconciling PR XRs", "prNumber", prNumber, "count", len(xrs))
		if err := w.handlePRBatch(ctx, prNumber, xrs); err != nil {
			w.logger.Error(err, "failed to process PR batch", "prNumber", prNumber)
			// Continue with other PRs
		}
	}

	if len(prXRs) > 0 {
		w.logger.Info("Reconciled existing PR XRs", "gvr", gvr.String(), "prCount", len(prXRs))
	}

	return nil
}

// watchGVR watches a specific GVR for changes
func (w *XRWatcher) watchGVR(ctx context.Context, gvr schema.GroupVersionResource) {
	w.logger.Info("Watching GVR", "gvr", gvr.String())

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := w.watchGVROnce(ctx, gvr); err != nil {
				w.logger.Error(err, "watch failed, retrying in 5s", "gvr", gvr.String())
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// watchGVROnce performs a single watch operation
func (w *XRWatcher) watchGVROnce(ctx context.Context, gvr schema.GroupVersionResource) error {
	watcher, err := w.dynamicClient.Resource(gvr).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			if event.Type == watch.Error {
				w.logger.Error(nil, "watch error event", "gvr", gvr.String())
				continue
			}

			xr, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				w.logger.Error(nil, "unexpected object type", "gvr", gvr.String())
				continue
			}

			w.handleXREvent(ctx, event.Type, xr)
		}
	}
}

// handlePRBatch processes all XRs for a single PR and posts one combined comment
func (w *XRWatcher) handlePRBatch(ctx context.Context, prNumber int, xrs []*unstructured.Unstructured) error {
	results := make(map[string]*differ.DiffResult)

	for _, xr := range xrs {
		name := xr.GetName()
		namespace := xr.GetNamespace()
		resourceVersion := xr.GetResourceVersion()

		// Check if we've already processed this version
		key := fmt.Sprintf("%s/%s", namespace, name)
		if namespace == "" {
			key = name
		}

		if lastVersion, exists := w.processedXRs[key]; exists && lastVersion == resourceVersion {
			continue // Already processed
		}

		w.logger.Info("Processing XR in batch",
			"name", name,
			"namespace", namespace,
			"prNumber", prNumber,
		)

		// Clone the XR and rename it to the production name
		baseName := w.detector.GetBaseName(xr)
		xrForDiff := xr.DeepCopy()
		xrForDiff.SetName(baseName)

		// Clear immutable metadata fields
		xrForDiff.SetUID("")
		xrForDiff.SetResourceVersion("")
		xrForDiff.SetGeneration(0)
		xrForDiff.SetCreationTimestamp(metav1.Time{})
		xrForDiff.SetManagedFields(nil)

		w.logger.Info("Comparing PR XR against production",
			"prName", name,
			"productionName", baseName,
		)

		// Calculate diff
		diff, err := w.differ.CalculateDiff(ctx, xrForDiff)
		if err != nil {
			w.logger.Error(err, "failed to calculate diff", "name", name)
			continue
		}

		// Store result using original XR name as key
		results[name] = diff

		// Mark as processed
		w.processedXRs[key] = resourceVersion
	}

	// If no results, nothing to post
	if len(results) == 0 {
		return nil
	}

	// Format combined comment
	var comment string
	if len(results) == 1 {
		// Single XR - use simple format
		for _, diff := range results {
			comment = w.formatter.FormatDiff(xrs[0], diff)
		}
	} else {
		// Multiple XRs - use combined format
		comment = w.formatter.FormatMultipleDiffs(results)
	}

	// Post to GitHub
	if w.vcsClient != nil {
		if err := w.vcsClient.PostComment(ctx, prNumber, comment); err != nil {
			return fmt.Errorf("failed to post GitHub comment: %w", err)
		}
		w.logger.Info("Posted GitHub comment", "prNumber", prNumber, "resourceCount", len(results))
	} else {
		// Dry-run mode
		w.logger.Info("Dry-run: would post comment", "prNumber", prNumber, "resourceCount", len(results))
	}

	return nil
}

// handleXREvent processes an XR event
func (w *XRWatcher) handleXREvent(ctx context.Context, eventType watch.EventType, xr *unstructured.Unstructured) {
	name := xr.GetName()
	namespace := xr.GetNamespace()
	resourceVersion := xr.GetResourceVersion()

	// Check if we've already processed this version
	key := fmt.Sprintf("%s/%s", namespace, name)
	if namespace == "" {
		key = name
	}

	if lastVersion, exists := w.processedXRs[key]; exists && lastVersion == resourceVersion {
		return // Already processed
	}

	// Detect PR number
	prNumber := w.detector.DetectPR(xr)
	if prNumber == 0 {
		// Not a PR preview XR, skip
		return
	}

	w.logger.Info("Processing XR event",
		"type", eventType,
		"name", name,
		"namespace", namespace,
		"prNumber", prNumber,
	)

	// Clone the XR and rename it to the production name
	// This allows crossplane-diff to compare against production resources
	baseName := w.detector.GetBaseName(xr)
	xrForDiff := xr.DeepCopy()
	xrForDiff.SetName(baseName)

	// Clear immutable metadata fields that would cause dry-run apply to fail
	xrForDiff.SetUID("")
	xrForDiff.SetResourceVersion("")
	xrForDiff.SetGeneration(0)
	xrForDiff.SetCreationTimestamp(metav1.Time{})
	xrForDiff.SetManagedFields(nil)

	w.logger.Info("Comparing PR XR against production",
		"prName", name,
		"productionName", baseName,
	)

	// Calculate diff using renamed XR
	// crossplane-diff will look for managed resources labeled with the production name
	diff, err := w.differ.CalculateDiff(ctx, xrForDiff)
	if err != nil {
		w.logger.Error(err, "failed to calculate diff", "name", name)
		return
	}

	// Format for GitHub
	comment := w.formatter.FormatDiff(xr, diff)

	// Post to GitHub (if VCS client is configured)
	if w.vcsClient != nil {
		if err := w.vcsClient.PostComment(ctx, prNumber, comment); err != nil {
			w.logger.Error(err, "failed to post GitHub comment", "prNumber", prNumber)
			return
		}
		w.logger.Info("Posted GitHub comment", "prNumber", prNumber)
	} else {
		// Dry-run mode: log the comment
		w.logger.Info("Dry-run: would post comment", "prNumber", prNumber, "comment", comment)
	}

	// Mark as processed
	w.processedXRs[key] = resourceVersion
}
