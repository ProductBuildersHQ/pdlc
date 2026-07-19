package readiness_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/plexusone/structured-evaluation/rubric"

	"github.com/ProductBuildersHQ/pdlc"
	"github.com/ProductBuildersHQ/pdlc/layout"
	"github.com/ProductBuildersHQ/pdlc/project"
	"github.com/ProductBuildersHQ/pdlc/readiness"
)

func TestMissingRequiredArtifactFailsAndBlocks(t *testing.T) {
	root := t.TempDir()
	proj := project.New("test", "Test", layout.ProfileMinimal)

	rep := evaluate(t, root, proj)

	if rep.Pass {
		t.Error("empty project should not pass readiness")
	}
	if len(rep.Blocking) == 0 {
		t.Error("expected blocking reason codes for missing required artifacts")
	}
	if !hasFindingCode(rep, readiness.CodeMissingRequired) {
		t.Error("expected a MISSING_REQUIRED finding")
	}
}

func TestAllArtifactsPresentButNoEvidenceIsConditional(t *testing.T) {
	root := t.TempDir()
	writeMinimalArtifacts(t, root)

	rep := evaluate(t, root, project.New("test", "Test", layout.ProfileMinimal))

	if len(rep.Blocking) != 0 {
		t.Errorf("presence is satisfied, so nothing should block; got %v", rep.Blocking)
	}
	if len(rep.Findings) != 0 {
		t.Errorf("presence is satisfied, so there should be no findings; got %d", len(rep.Findings))
	}
	// A baseline carries quality evidence, so unrun tools hold readiness back.
	if rep.Pass {
		t.Error("project with no tool evidence must not report ready for baseline")
	}
	if rep.OverallDecision != string(rubric.DecisionConditional) {
		t.Errorf("overallDecision = %q, want %q", rep.OverallDecision, rubric.DecisionConditional)
	}
}

func TestFullEvidencePasses(t *testing.T) {
	root := t.TempDir()
	m := pdlc.MustLayout()
	writeMinimalArtifacts(t, root)

	// Supply a passing leaf report for every in-scope domain that declares one.
	proj := project.New("test", "Test", layout.ProfileMinimal)
	for _, d := range m.Domains {
		if proj.LevelFor(m, d.ID) == layout.LevelExcluded {
			continue
		}
		for _, lr := range d.LeafReports {
			write(t, root, lr+"/report.json", `{"pass": true, "overallDecision": "pass"}`)
		}
	}

	rep := evaluate(t, root, proj)

	if !rep.Pass {
		t.Errorf("project with all artifacts and passing evidence should pass; blocking=%v", rep.Blocking)
	}
	if rep.OverallDecision != string(rubric.DecisionPass) {
		t.Errorf("overallDecision = %q, want %q", rep.OverallDecision, rubric.DecisionPass)
	}
}

func TestProfileScalesWhatIsRequired(t *testing.T) {
	// The same repository is judged against two profiles: satisfying minimal
	// leaves full with real gaps, which is what lets a community project start
	// small and grow without restructuring.
	root := t.TempDir()
	writeMinimalArtifacts(t, root)

	minimalRep := evaluate(t, root, project.New("test", "Test", layout.ProfileMinimal))
	fullRep := evaluate(t, root, project.New("test", "Test", layout.ProfileFull))

	if len(minimalRep.Findings) != 0 {
		t.Errorf("minimal profile should be satisfied, got %d findings", len(minimalRep.Findings))
	}
	if len(fullRep.Findings) == 0 {
		t.Error("full profile should report gaps that minimal does not")
	}
	if len(fullRep.Categories) <= len(minimalRep.Categories) {
		t.Errorf("full profile should score more domains than minimal: full=%d minimal=%d",
			len(fullRep.Categories), len(minimalRep.Categories))
	}
}

// writeMinimalArtifacts creates a placeholder for every artifact the minimal
// profile requires — the floor every profile builds on.
func writeMinimalArtifacts(t *testing.T, root string) {
	t.Helper()
	for _, a := range pdlc.MustLayout().Artifacts {
		if a.LevelFor(layout.ProfileMinimal) == layout.LevelRequired {
			write(t, root, a.Canonical, "placeholder\n")
		}
	}
}

func TestUnrunToolScoresPartialNotPass(t *testing.T) {
	root := t.TempDir()
	writeMinimalArtifacts(t, root)

	proj := project.New("test", "Test", layout.ProfileMinimal)
	rep := evaluate(t, root, proj)

	// The api domain declares a leaf report that has not been produced.
	cat, ok := category(rep, "api")
	if !ok {
		t.Fatal("expected an api category")
	}
	if cat.Score == rubric.ScorePass {
		t.Error("api scored pass with no api-style report; unrun tools must not pass silently")
	}
	if cat.Score != rubric.ScorePartial {
		t.Errorf("api score = %q, want %q", cat.Score, rubric.ScorePartial)
	}
	if !hasReasonCode(cat, readiness.CodeToolNotRun) {
		t.Errorf("expected TOOL_NOT_RUN reason code, got %v", cat.ReasonCodes)
	}
}

func TestFailingLeafReportFailsDomain(t *testing.T) {
	root := t.TempDir()
	writeMinimalArtifacts(t, root)
	write(t, root, "quality/api-style/report.json", `{"pass": false, "overallDecision": "fail"}`)

	rep := evaluate(t, root, project.New("test", "Test", layout.ProfileMinimal))

	cat, ok := category(rep, "api")
	if !ok {
		t.Fatal("expected an api category")
	}
	if cat.Score != rubric.ScoreFail {
		t.Errorf("api score = %q, want %q", cat.Score, rubric.ScoreFail)
	}
	if rep.Pass {
		t.Error("a failing leaf report must block overall readiness")
	}
}

func TestPassingLeafReportPassesDomain(t *testing.T) {
	root := t.TempDir()
	writeMinimalArtifacts(t, root)
	write(t, root, "quality/api-style/report.json", `{"pass": true, "overallDecision": "pass"}`)

	rep := evaluate(t, root, project.New("test", "Test", layout.ProfileMinimal))

	cat, _ := category(rep, "api")
	if cat.Score != rubric.ScorePass {
		t.Errorf("api score = %q, want %q (reasoning: %s)", cat.Score, rubric.ScorePass, cat.Reasoning)
	}
}

func TestExcludedDomainsAreNotFailures(t *testing.T) {
	root := t.TempDir()
	proj := project.New("test", "Test", layout.ProfileMinimal)

	rep := evaluate(t, root, proj)

	excluded, ok := rep.Extensions["excluded"].([]string)
	if !ok || len(excluded) == 0 {
		t.Fatalf("expected excluded domains under the minimal profile, got %v", rep.Extensions["excluded"])
	}
	for _, id := range excluded {
		if _, found := category(rep, id); found {
			t.Errorf("excluded domain %q must not appear as a scored category", id)
		}
	}
}

func TestOverrideCanExcludeADomain(t *testing.T) {
	root := t.TempDir()
	proj := project.New("test", "Test", layout.ProfileFull)
	proj.Spec.Artifacts = map[string]layout.Level{"guides": layout.LevelExcluded}

	rep := evaluate(t, root, proj)

	if _, found := category(rep, "guides"); found {
		t.Error("guides was excluded by override but still scored")
	}
	overrides, ok := rep.Extensions["overrides"].(map[string]string)
	if !ok || overrides["guides"] != string(layout.LevelExcluded) {
		t.Errorf("expected overrides to record the guides exclusion, got %v", rep.Extensions["overrides"])
	}
}

func TestLocaleCoverageIsComputed(t *testing.T) {
	root := t.TempDir()
	write(t, root, "locales/ui/en-US.json", `{"entries":{
		"a":{"value":"A","status":"translated"},
		"b":{"value":"B","status":"translated"},
		"c":{"value":"C","status":"translated"},
		"d":{"value":"D","status":"translated"}}}`)
	write(t, root, "locales/ui/de-DE.json", `{"entries":{
		"a":{"value":"A-de","status":"translated"},
		"b":{"value":"B-de","status":"translated"},
		"c":{"value":"","status":"missing"}}}`)

	proj := project.New("test", "Test", layout.ProfileFull)
	proj.Spec.Locales = project.Locales{Source: "en-US", Targets: []string{"de-DE", "ja-JP"}}

	rep := evaluate(t, root, proj)

	cov, ok := rep.Extensions["localeCoverage"].(map[string]float64)
	if !ok {
		t.Fatalf("expected localeCoverage extension, got %v", rep.Extensions["localeCoverage"])
	}
	if got, want := cov["de-DE"], 0.5; got != want {
		t.Errorf("de-DE coverage = %v, want %v", got, want)
	}
	if got, want := cov["ja-JP"], 0.0; got != want {
		t.Errorf("ja-JP coverage = %v, want %v (absent catalog is zero coverage)", got, want)
	}
}

func TestEvaluationIsIdempotent(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/specs/source/prd.md", "# Product Requirements\n")
	proj := project.New("test", "Test", layout.ProfileMinimal)

	first := evaluate(t, root, proj)
	second := evaluate(t, root, proj)

	// Compare everything except the generation timestamp.
	first.Metadata.GeneratedAt = second.Metadata.GeneratedAt
	a, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first report: %v", err)
	}
	b, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second report: %v", err)
	}
	if string(a) != string(b) {
		t.Error("two evaluations of an unchanged project produced different reports")
	}
}

func TestWriteProducesReadableReport(t *testing.T) {
	root := t.TempDir()
	rep := evaluate(t, root, project.New("test", "Test", layout.ProfileMinimal))

	dest, err := readiness.Write(root, rep)
	if err != nil {
		t.Fatalf("write report: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read written report: %v", err)
	}
	var round rubric.Rubric
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("written report is not valid JSON: %v", err)
	}
	if round.ReviewType != readiness.ReviewType {
		t.Errorf("reviewType = %q, want %q", round.ReviewType, readiness.ReviewType)
	}
	if round.RubricID != readiness.RubricID {
		t.Errorf("rubricId = %q, want %q", round.RubricID, readiness.RubricID)
	}
}

func evaluate(t *testing.T, root string, proj *project.Project) *rubric.Rubric {
	t.Helper()
	rep, err := readiness.Evaluate(pdlc.MustLayout(), proj, readiness.Options{Root: root})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return rep
}

func category(rep *rubric.Rubric, id string) (rubric.CategoryResult, bool) {
	for _, c := range rep.Categories {
		if c.Category == id {
			return c, true
		}
	}
	return rubric.CategoryResult{}, false
}

func hasReasonCode(c rubric.CategoryResult, want rubric.ReasonCode) bool {
	for _, code := range c.ReasonCodes {
		if code == want {
			return true
		}
	}
	return false
}

func hasFindingCode(rep *rubric.Rubric, want rubric.ReasonCode) bool {
	for _, f := range rep.Findings {
		if f.Code == want {
			return true
		}
	}
	return false
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
