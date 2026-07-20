package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ProductBuildersHQ/pdlc/layout"
	"github.com/ProductBuildersHQ/pdlc/project"
)

func TestLoadYAMLWithSpecProfileShorthandAndObject(t *testing.T) {
	root := t.TempDir()
	write(t, root, "pdlc.yaml", `
apiVersion: pdlc.productbuildershq.org/v1
kind: ProductProject
metadata:
  id: demo
spec:
  profile: full
  specProfiles:
    - big-tech-product
    - name: aws-aidlc
      provider: aidlc
      role: builder
`)

	proj, err := project.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(proj.Spec.SpecProfiles) != 2 {
		t.Fatalf("want 2 spec profiles, got %d", len(proj.Spec.SpecProfiles))
	}

	// Shorthand string → visionspec authoring by default.
	sp0 := proj.Spec.SpecProfiles[0]
	if sp0.Name != "big-tech-product" {
		t.Errorf("sp0 name = %q", sp0.Name)
	}
	if sp0.ResolvedProvider() != project.ProviderVisionSpec {
		t.Errorf("sp0 provider = %q, want visionspec default", sp0.ResolvedProvider())
	}
	if sp0.ResolvedRole() != project.RoleAuthoring {
		t.Errorf("sp0 role = %q, want authoring default", sp0.ResolvedRole())
	}

	// Object form carries explicit provider/role.
	sp1 := proj.Spec.SpecProfiles[1]
	if sp1.Name != "aws-aidlc" || sp1.ResolvedProvider() != project.ProviderAIDLC || sp1.ResolvedRole() != project.RoleBuilder {
		t.Errorf("sp1 = %+v", sp1)
	}
}

func TestLoadJSONManifest(t *testing.T) {
	root := t.TempDir()
	write(t, root, "pdlc.json", `{
  "apiVersion": "pdlc.productbuildershq.org/v1",
  "kind": "ProductProject",
  "metadata": { "id": "demo" },
  "spec": {
    "profile": "standard",
    "specProfiles": ["big-tech-feature", {"name": "github-speckit", "provider": "speckit", "role": "builder"}]
  }
}`)

	proj, err := project.Load(root)
	if err != nil {
		t.Fatalf("load json: %v", err)
	}
	if proj.Spec.Profile != layout.ProfileStandard {
		t.Errorf("profile = %q", proj.Spec.Profile)
	}
	if len(proj.Spec.SpecProfiles) != 2 {
		t.Fatalf("want 2 spec profiles, got %d", len(proj.Spec.SpecProfiles))
	}
	if proj.Spec.SpecProfiles[0].Name != "big-tech-feature" {
		t.Errorf("first profile = %q", proj.Spec.SpecProfiles[0].Name)
	}
	if proj.Spec.SpecProfiles[1].ResolvedProvider() != project.ProviderSpecKit {
		t.Errorf("second provider = %q", proj.Spec.SpecProfiles[1].ResolvedProvider())
	}
}

func TestSaveRoundTripYAMLAndJSON(t *testing.T) {
	for _, name := range []string{"pdlc.yaml", "pdlc.json"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			proj := project.New("demo", "Demo", layout.ProfileFull)
			proj.Spec.SpecProfiles = []project.SpecProfile{
				{Name: "big-tech-product"},
				{Name: "aws-aidlc", Provider: project.ProviderAIDLC, Role: project.RoleBuilder},
			}

			if err := project.Save(root, name, proj); err != nil {
				t.Fatalf("save %s: %v", name, err)
			}

			got, err := project.Load(root)
			if err != nil {
				t.Fatalf("reload %s: %v", name, err)
			}
			if len(got.Spec.SpecProfiles) != 2 {
				t.Fatalf("%s: want 2 spec profiles, got %d", name, len(got.Spec.SpecProfiles))
			}
			if got.Spec.SpecProfiles[1].Provider != project.ProviderAIDLC {
				t.Errorf("%s: provider not preserved: %+v", name, got.Spec.SpecProfiles[1])
			}
		})
	}
}

func TestLoadMissingManifest(t *testing.T) {
	_, err := project.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected ErrNotFound for a directory with no manifest")
	}
}

func TestEmptySpecProfileNameIsInvalid(t *testing.T) {
	root := t.TempDir()
	write(t, root, "pdlc.yaml", `
apiVersion: pdlc.productbuildershq.org/v1
kind: ProductProject
metadata:
  id: demo
spec:
  profile: full
  specProfiles:
    - name: ""
      provider: visionspec
`)
	if _, err := project.Load(root); err == nil {
		t.Fatal("expected validation error for empty spec profile name")
	}
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
