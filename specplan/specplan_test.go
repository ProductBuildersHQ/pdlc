package specplan_test

import (
	"slices"
	"testing"

	"github.com/ProductBuildersHQ/pdlc/specplan"
)

func TestListProfilesIncludesBigTech(t *testing.T) {
	names := specplan.ListProfiles()
	if len(names) == 0 {
		t.Fatal("no spec profiles listed")
	}
	if !slices.Contains(names, "big-tech-product") {
		t.Errorf("expected big-tech-product among profiles, got %v", names)
	}
}

func TestResolveBigTechProduct(t *testing.T) {
	plan, err := specplan.Resolve("big-tech-product", "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if plan.SpecsRoot != specplan.DefaultSpecsRoot {
		t.Errorf("specsRoot = %q, want %q", plan.SpecsRoot, specplan.DefaultSpecsRoot)
	}

	byType := map[string]specplan.Artifact{}
	for _, a := range plan.Artifacts {
		byType[a.SpecType] = a
	}

	// The standard specs must route to the agreed subdirectories and have both
	// a template and a rubric available.
	cases := map[string]string{
		"prd":   "docs/specs/source/prd.md",
		"uxd":   "docs/specs/source/uxd.md",
		"press": "docs/specs/gtm/press.md",
		"trd":   "docs/specs/technical/trd.md",
		"tpd":   "docs/specs/technical/tpd.md",
	}
	for specType, wantPath := range cases {
		a, ok := byType[specType]
		if !ok {
			t.Errorf("big-tech-product plan missing %q", specType)
			continue
		}
		if a.Path != wantPath {
			t.Errorf("%s path = %q, want %q", specType, a.Path, wantPath)
		}
		if !a.HasTemplate {
			t.Errorf("%s should have a template", specType)
		}
		if !a.HasRubric {
			t.Errorf("%s should have a rubric", specType)
		}
	}
}

func TestResolveUnknownProfileErrors(t *testing.T) {
	if _, err := specplan.Resolve("does-not-exist", ""); err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestTemplateReturnsContent(t *testing.T) {
	content, err := specplan.Template("big-tech-product", "prd")
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	if len(content) == 0 {
		t.Error("prd template is empty")
	}
}
