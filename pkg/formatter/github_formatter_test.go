package formatter

import (
	"strings"
	"testing"

	"github.com/millstonehq/crossplane-plan/pkg/differ"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGitHubFormatter_FormatDiff_NoChanges(t *testing.T) {
	formatter := NewGitHubFormatter()

	xr := &unstructured.Unstructured{}
	xr.SetKind("XGitHubRepository")
	xr.SetName("mill")

	result := &differ.DiffResult{
		XR:         xr,
		RawDiff:    "",
		HasChanges: false,
		Summary:    "No changes detected for XGitHubRepository/mill",
	}

	output := formatter.FormatDiff(xr, result)

	// Check key elements
	if !strings.Contains(output, "🔄 Crossplane Preview") {
		t.Error("Missing header")
	}
	if !strings.Contains(output, "XGitHubRepository/mill") {
		t.Error("Missing resource name")
	}
	if !strings.Contains(output, "✅ No Changes") {
		t.Error("Missing no changes indicator")
	}
	if !strings.Contains(output, "crossplane-plan") {
		t.Error("Missing footer attribution")
	}
}

func TestGitHubFormatter_FormatDiff_WithChanges(t *testing.T) {
	formatter := NewGitHubFormatter()

	xr := &unstructured.Unstructured{}
	xr.SetKind("XGitHubRepository")
	xr.SetName("pr-123-mill")

	result := &differ.DiffResult{
		XR:         xr,
		RawDiff:    "+ added line\n- removed line\n  context line",
		HasChanges: true,
		Summary:    "Changes detected for XGitHubRepository/pr-123-mill: +1 -1 lines",
	}

	output := formatter.FormatDiff(xr, result)

	// Check key elements
	if !strings.Contains(output, "📋 Changes Detected") {
		t.Error("Missing changes detected header")
	}
	if !strings.Contains(output, "Changes detected for XGitHubRepository/pr-123-mill: +1 -1 lines") {
		t.Error("Missing summary")
	}
	if !strings.Contains(output, "<details>") {
		t.Error("Missing collapsible details")
	}
	if !strings.Contains(output, "```diff") {
		t.Error("Missing diff code block")
	}
	if !strings.Contains(output, "+ added line") {
		t.Error("Missing diff content")
	}
}

func TestGitHubFormatter_FormatDiff_WithNamespace(t *testing.T) {
	formatter := NewGitHubFormatter()

	xr := &unstructured.Unstructured{}
	xr.SetKind("XGitHubRepository")
	xr.SetName("mill")
	xr.SetNamespace("millstone-prod")

	result := &differ.DiffResult{
		XR:         xr,
		RawDiff:    "",
		HasChanges: false,
		Summary:    "No changes",
	}

	output := formatter.FormatDiff(xr, result)

	if !strings.Contains(output, "**Namespace:** `millstone-prod`") {
		t.Error("Missing namespace in output")
	}
}

func TestGitHubFormatter_FormatMultipleDiffs_NoChanges(t *testing.T) {
	formatter := NewGitHubFormatter()

	results := map[string]*differ.DiffResult{
		"mill": {
			HasChanges: false,
			Summary:    "No changes",
		},
		"books": {
			HasChanges: false,
			Summary:    "No changes",
		},
	}

	output := formatter.FormatMultipleDiffs(results)

	if !strings.Contains(output, "**Resources:** 2 total, 0 with changes") {
		t.Error("Missing resource count")
	}
	if !strings.Contains(output, "✅ No Changes") {
		t.Error("Missing no changes message")
	}
}

func TestGitHubFormatter_FormatMultipleDiffs_WithChanges(t *testing.T) {
	formatter := NewGitHubFormatter()

	results := map[string]*differ.DiffResult{
		"mill": {
			RawDiff:    "+ change",
			HasChanges: true,
			Summary:    "Changes: +1 lines",
		},
		"books": {
			HasChanges: false,
			Summary:    "No changes",
		},
	}

	output := formatter.FormatMultipleDiffs(results)

	if !strings.Contains(output, "**Resources:** 2 total, 1 with changes") {
		t.Error("Missing resource count")
	}
	if !strings.Contains(output, "📋 Modified Resources") {
		t.Error("Missing modified resources section")
	}
	if !strings.Contains(output, "**mill**: Changes: +1 lines") {
		t.Error("Missing changed resource")
	}
	if strings.Contains(output, "**books**") {
		t.Error("Should not include unchanged resources in changes list")
	}
}

func TestGitHubFormatter_FormatMultipleDiffs_WithDeletions(t *testing.T) {
	formatter := NewGitHubFormatter()

	xr := &unstructured.Unstructured{}
	xr.SetKind("XGitHubRepository")
	xr.SetName("provider-tailscale")
	xr.Object["spec"] = map[string]interface{}{
		"name": "provider-tailscale",
	}

	results := map[string]*differ.DiffResult{
		"pr-5-provider-upjet-tailscale": {
			RawDiff:    "+ new resource",
			HasChanges: true,
			Summary:    "Changes detected",
		},
		"DELETED-provider-tailscale": {
			XR:         xr,
			RawDiff:    "Resource will be deleted",
			HasChanges: true,
			Summary:    "⚠️  Resource will be **DELETED**",
		},
	}

	output := formatter.FormatMultipleDiffs(results)

	if !strings.Contains(output, "**Resources:** 2 total, 2 with changes") {
		t.Error("Missing resource count")
	}
	if !strings.Contains(output, "📋 Modified Resources") {
		t.Error("Missing modified resources section")
	}
	if !strings.Contains(output, "🗑️ Deleted Resources") {
		t.Error("Missing deleted resources section")
	}
	if !strings.Contains(output, "**provider-tailscale**: ⚠️  Resource will be **DELETED**") {
		t.Error("Missing deletion in deleted resources list")
	}
	if !strings.Contains(output, "`provider-tailscale` (DELETION)") {
		t.Error("Missing deletion details header")
	}
	if !strings.Contains(output, "⚠️ WARNING:** This resource will be **DELETED**") {
		t.Error("Missing deletion warning")
	}
}

func TestGitHubFormatter_FormatMultipleDiffs_MixedChanges(t *testing.T) {
	formatter := NewGitHubFormatter()

	deletedXR := &unstructured.Unstructured{}
	deletedXR.SetKind("XGitHubRepository")
	deletedXR.SetName("old-repo")

	results := map[string]*differ.DiffResult{
		"modified-repo": {
			RawDiff:    "+ modified",
			HasChanges: true,
			Summary:    "Modified",
		},
		"DELETED-old-repo": {
			XR:         deletedXR,
			RawDiff:    "Deleted",
			HasChanges: true,
			Summary:    "⚠️  Resource will be **DELETED**",
		},
		"no-change-repo": {
			HasChanges: false,
			Summary:    "No changes",
		},
	}

	output := formatter.FormatMultipleDiffs(results)

	if !strings.Contains(output, "**Resources:** 3 total, 2 with changes") {
		t.Error("Missing resource count")
	}
	if !strings.Contains(output, "📋 Modified Resources") {
		t.Error("Missing modified resources section")
	}
	if !strings.Contains(output, "🗑️ Deleted Resources") {
		t.Error("Missing deleted resources section")
	}
	// Ensure no-change repo is not in either section
	modifiedSection := strings.Index(output, "📋 Modified Resources")
	deletedSection := strings.Index(output, "🗑️ Deleted Resources")

	if modifiedSection > 0 && deletedSection > 0 {
		betweenSections := output[modifiedSection:deletedSection]
		if strings.Contains(betweenSections, "no-change-repo") {
			t.Error("Unchanged resource should not appear in modified section")
		}
	}
}
