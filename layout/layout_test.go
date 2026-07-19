package layout_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ProductBuildersHQ/pdlc"
	"github.com/ProductBuildersHQ/pdlc/layout"
)

func TestEmbeddedManifestIsValid(t *testing.T) {
	m, err := pdlc.Layout()
	if err != nil {
		t.Fatalf("load embedded layout: %v", err)
	}
	if len(m.Artifacts) == 0 {
		t.Fatal("embedded layout has no artifacts")
	}
	if len(m.Domains) == 0 {
		t.Fatal("embedded layout has no domains")
	}
}

func TestEveryArtifactHasAKnownDomain(t *testing.T) {
	m := pdlc.MustLayout()
	for _, a := range m.Artifacts {
		if _, ok := m.Domain(a.Domain); !ok {
			t.Errorf("artifact %q references unknown domain %q", a.ID, a.Domain)
		}
	}
}

func TestRequiredArtifactsExistAtEveryProfile(t *testing.T) {
	m := pdlc.MustLayout()
	for _, p := range layout.Profiles {
		var required int
		for _, a := range m.Artifacts {
			if a.LevelFor(p) == layout.LevelRequired {
				required++
			}
		}
		if required == 0 {
			t.Errorf("profile %q requires no artifacts; every profile needs a floor", p)
		}
	}
}

func TestIsImmovable(t *testing.T) {
	m := pdlc.MustLayout()

	immovable := []string{
		"aidlc-docs",
		"aidlc-docs/inception/requirements.md",
		".visionspec/context-cache.json",
		"specs/001-feature/spec.md",
		"docs/_generated/aidlc/index.md",
		"node_modules/react/index.js",
	}
	for _, p := range immovable {
		if !m.IsImmovable(p) {
			t.Errorf("IsImmovable(%q) = false, want true", p)
		}
	}

	movable := []string{
		"docs/specs/source/prd.md",
		"prototype/src/App.tsx",
		"locales/ui/de-DE.json",
		"docs/api/openapi.yaml",
		// A path that merely starts with the same letters must not match.
		"specifications/lifecycle.md",
	}
	for _, p := range movable {
		if m.IsImmovable(p) {
			t.Errorf("IsImmovable(%q) = true, want false", p)
		}
	}
}

func TestClassifyFindsMisplacedArtifacts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "product-requirements.md", "# Product Requirements\n\nFR-1: something.\n")
	writeFile(t, root, "api/openapi.yaml", "openapi: 3.1.0\ninfo:\n  title: Test\n")
	writeFile(t, root, "docs/api/../guides/user/getting-started.md", "# Getting started\n")
	// Immovable content must never be classified.
	writeFile(t, root, "aidlc-docs/inception/requirements.md", "# Product Requirements\n")

	m := pdlc.MustLayout()
	inv, err := m.Classify(root)
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	prd, ok := inv.Found("prd")
	if !ok {
		t.Fatal("expected product-requirements.md to classify as prd")
	}
	if prd.Conformant {
		t.Errorf("prd at %q should not be conformant", prd.Path)
	}
	if prd.Canonical != "docs/specs/source/prd.md" {
		t.Errorf("prd canonical = %q, want docs/specs/source/prd.md", prd.Canonical)
	}

	if _, ok := inv.Found("openapi"); !ok {
		t.Error("expected api/openapi.yaml to classify as openapi")
	}

	for _, e := range inv.Entries {
		if len(e.Path) >= len("aidlc-docs") && e.Path[:len("aidlc-docs")] == "aidlc-docs" {
			t.Errorf("immovable path %q was classified", e.Path)
		}
	}

	if len(inv.Moves()) == 0 {
		t.Error("expected at least one proposed move")
	}
}

func TestClassifyMarksConformantPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/specs/source/prd.md", "# Product Requirements\n")

	inv, err := pdlc.MustLayout().Classify(root)
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	prd, ok := inv.Found("prd")
	if !ok {
		t.Fatal("expected prd to be found")
	}
	if !prd.Conformant {
		t.Errorf("prd at canonical path should be conformant, got %+v", prd)
	}
	if len(inv.Moves()) != 0 {
		t.Errorf("expected no moves, got %v", inv.Moves())
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
