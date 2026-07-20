// Package readiness evaluates a project against its declared PDLC profile and
// emits a structured-evaluation report answering, per domain: does the artifact
// exist, and does its quality evaluation pass.
//
// The checker is deterministic. It reads leaf reports produced by other tools
// but never re-derives or overrules them, and a domain whose tool has not run
// scores partial with reason code TOOL_NOT_RUN rather than silently passing.
package readiness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/plexusone/structured-evaluation/rubric"

	"github.com/ProductBuildersHQ/pdlc/layout"
	"github.com/ProductBuildersHQ/pdlc/project"
)

// ReviewType identifies readiness reports among other structured evaluations.
const ReviewType = "pdlc-readiness"

// RubricID and RubricVersion identify the scoring contract, so reports remain
// comparable across projects and across executors (Go checker or AI agent).
const (
	RubricID      = "pdlc-readiness"
	RubricVersion = "0.1.0"
)

// ReportPath is where the readiness report is written, relative to the root.
const ReportPath = "quality/readiness.json"

// Reason codes specific to readiness evaluation.
const (
	// CodeMissingRequired marks a required artifact that is absent.
	CodeMissingRequired rubric.ReasonCode = "PDLC-MISSING_REQUIRED"

	// CodeToolNotRun marks a domain whose evaluation tool has not produced a
	// leaf report. Never treated as a pass.
	CodeToolNotRun rubric.ReasonCode = "PDLC-TOOL_NOT_RUN"

	// CodeLeafFailed marks a domain whose leaf report failed.
	CodeLeafFailed rubric.ReasonCode = "PDLC-LEAF_FAILED"
)

// Options configures an evaluation run.
type Options struct {
	// Root is the project repository root.
	Root string

	// GeneratedBy identifies the executor in report metadata.
	GeneratedBy string

	// Now supplies the report timestamp. Zero means time.Now().UTC().
	Now time.Time
}

// Evaluate checks the project at opts.Root against its declared profile and
// returns a structured-evaluation report.
func Evaluate(m *layout.Manifest, proj *project.Project, opts Options) (*rubric.Rubric, error) {
	if m == nil {
		return nil, fmt.Errorf("readiness: layout manifest is required")
	}
	if proj == nil {
		return nil, fmt.Errorf("readiness: project manifest is required")
	}
	root := opts.Root
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root %q: %w", root, err)
	}

	rep := rubric.NewRubric(ReviewType, abs)
	rep.RubricID = RubricID
	rep.RubricVersion = RubricVersion
	rep.Metadata.DocumentID = proj.Metadata.ID
	rep.Metadata.DocumentTitle = proj.Metadata.Title
	if !opts.Now.IsZero() {
		rep.Metadata.GeneratedAt = opts.Now.UTC()
	}
	if opts.GeneratedBy != "" {
		rep.Metadata.GeneratedBy = opts.GeneratedBy
	}

	var (
		excluded []string
		blocking []rubric.ReasonCode
	)

	for _, domain := range m.Domains {
		if proj.LevelFor(m, domain.ID) == layout.LevelExcluded {
			excluded = append(excluded, domain.ID)
			continue
		}
		override, hasOverride := proj.Spec.Artifacts[domain.ID]

		result, findings := evaluateDomain(abs, m, domain, proj.Spec.Profile, override, hasOverride)
		rep.Categories = append(rep.Categories, result)
		rep.Findings = append(rep.Findings, findings...)

		for _, code := range result.ReasonCodes {
			if code == CodeMissingRequired || code == CodeLeafFailed {
				blocking = appendUnique(blocking, code)
			}
		}
	}

	sort.Strings(excluded)
	rep.Extensions = map[string]any{
		"specVersion": RubricVersion,
		"profile":     string(proj.Spec.Profile),
		"excluded":    excluded,
	}
	if len(proj.Spec.Artifacts) > 0 {
		overrides := make(map[string]string, len(proj.Spec.Artifacts))
		for domain, lvl := range proj.Spec.Artifacts {
			overrides[domain] = string(lvl)
		}
		rep.Extensions["overrides"] = overrides
	}
	if cov, err := localeCoverage(abs, proj); err != nil {
		return nil, err
	} else if cov != nil {
		rep.Extensions["localeCoverage"] = cov
	}

	// A baseline carries quality evidence, so a domain whose tool has not run
	// holds readiness at conditional rather than passing it through.
	rep.SetBlocking(blocking)
	rep.Pass = len(blocking) == 0 && !hasScore(rep, rubric.ScorePartial)
	rep.Decision = rubric.EvaluateResults(rep.Categories, rep.Findings, rep.PassCriteria, nil)
	rep.OverallDecision = string(decisionFor(rep))
	rep.Summary = summarize(rep, proj)

	return rep, nil
}

// evaluateDomain scores one domain on presence and, where a leaf report exists,
// on quality. Artifact requirement levels come from the project's profile,
// adjusted by an explicit domain override when the project declares one.
func evaluateDomain(
	root string,
	m *layout.Manifest,
	domain layout.Domain,
	profile layout.Profile,
	override layout.Level,
	hasOverride bool,
) (rubric.CategoryResult, []rubric.Finding) {
	result := rubric.CategoryResult{
		Category:         domain.ID,
		ChecklistResults: &rubric.ChecklistResults{},
	}
	var (
		findings []rubric.Finding
		reasons  []string
	)

	// Presence.
	for _, a := range m.ArtifactsInDomain(domain.ID) {
		artifactLevel := a.LevelFor(profile)
		if hasOverride {
			artifactLevel = layout.ApplyOverride(artifactLevel, override)
		}
		if artifactLevel == layout.LevelExcluded {
			continue
		}
		present := exists(filepath.Join(root, a.Canonical))

		switch {
		case present && required(artifactLevel):
			result.ChecklistResults.RequiredPresent = append(result.ChecklistResults.RequiredPresent, a.Canonical)
		case !present && required(artifactLevel):
			result.ChecklistResults.RequiredMissing = append(result.ChecklistResults.RequiredMissing, a.Canonical)
			findings = append(findings, rubric.Finding{
				ID:             fmt.Sprintf("%s-missing-%s", domain.ID, a.ID),
				Category:       domain.ID,
				Code:           CodeMissingRequired,
				Severity:       rubric.SeverityHigh,
				Title:          fmt.Sprintf("Missing required artifact: %s", a.ID),
				Description:    fmt.Sprintf("%s is required by this project's profile but was not found.", a.Canonical),
				Recommendation: fmt.Sprintf("Author %s, or adjust the profile if the domain is out of scope.", a.Canonical),
				Location:       a.Canonical,
			})
		case present:
			result.ChecklistResults.OptionalPresent = append(result.ChecklistResults.OptionalPresent, a.Canonical)
		default:
			result.ChecklistResults.OptionalMissing = append(result.ChecklistResults.OptionalMissing, a.Canonical)
		}
	}

	missing := len(result.ChecklistResults.RequiredMissing)
	presentCount := len(result.ChecklistResults.RequiredPresent) + len(result.ChecklistResults.OptionalPresent)

	// Quality, from leaf reports only.
	leaf, leafErr := readLeafReports(root, domain)
	switch {
	case len(domain.LeafReports) == 0:
		// Domain has no tool; presence is the whole story.
	case leafErr != nil:
		reasons = append(reasons, fmt.Sprintf("leaf report unreadable: %v", leafErr))
		result.ReasonCodes = append(result.ReasonCodes, CodeToolNotRun)
	case leaf == nil:
		reasons = append(reasons, fmt.Sprintf("no evaluation found under %s", strings.Join(domain.LeafReports, ", ")))
		result.ReasonCodes = append(result.ReasonCodes, CodeToolNotRun)
	case !leaf.pass:
		reasons = append(reasons, fmt.Sprintf("evaluation failed (%s)", leaf.source))
		result.ReasonCodes = append(result.ReasonCodes, CodeLeafFailed)
		findings = append(findings, rubric.Finding{
			ID:             fmt.Sprintf("%s-leaf-failed", domain.ID),
			Category:       domain.ID,
			Code:           CodeLeafFailed,
			Severity:       rubric.SeverityHigh,
			Title:          fmt.Sprintf("%s evaluation failed", domain.ID),
			Description:    fmt.Sprintf("The evaluation report at %s did not pass.", leaf.source),
			Recommendation: "Address the findings in the linked report, then re-run the tool.",
			Location:       leaf.source,
		})
	default:
		reasons = append(reasons, fmt.Sprintf("evaluation passed (%s)", leaf.source))
	}

	// Score.
	switch {
	case missing > 0:
		result.Score = rubric.ScoreFail
		result.ReasonCodes = appendUnique(result.ReasonCodes, CodeMissingRequired)
		reasons = append([]string{fmt.Sprintf("%d required artifact(s) missing", missing)}, reasons...)
	case hasCode(result.ReasonCodes, CodeLeafFailed):
		result.Score = rubric.ScoreFail
	case hasCode(result.ReasonCodes, CodeToolNotRun):
		result.Score = rubric.ScorePartial
	case presentCount == 0 && len(m.ArtifactsInDomain(domain.ID)) > 0:
		// Nothing required and nothing authored: in scope but untouched.
		result.Score = rubric.ScorePass
		reasons = append([]string{"nothing required by this profile; no artifacts authored"}, reasons...)
	default:
		result.Score = rubric.ScorePass
		reasons = append([]string{fmt.Sprintf("%d artifact(s) present", presentCount)}, reasons...)
	}

	result.Reasoning = strings.Join(reasons, "; ")
	if result.Reasoning == "" {
		result.Reasoning = "no artifacts or evaluations configured for this domain"
	}
	return result, findings
}

// leafResult is the outcome extracted from a domain's tool report.
type leafResult struct {
	pass   bool
	source string
}

// readLeafReports finds the most relevant evaluation report for a domain and
// extracts its pass/fail outcome. It returns nil when no report exists.
func readLeafReports(root string, domain layout.Domain) (*leafResult, error) {
	for _, rel := range domain.LeafReports {
		dir := filepath.Join(root, rel)
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", rel, err)
		}

		var files []string
		if info.IsDir() {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", rel, err)
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
					files = append(files, filepath.Join(dir, e.Name()))
				}
			}
			sort.Strings(files)
		} else {
			files = []string{dir}
		}

		// A domain passes only if every report under it passes.
		var (
			found  bool
			allOK  = true
			source string
		)
		for _, f := range files {
			ok, err := reportPasses(f)
			if err != nil {
				return nil, err
			}
			found = true
			if source == "" || !ok {
				source = mustRel(root, f)
			}
			allOK = allOK && ok
		}
		if found {
			return &leafResult{pass: allOK, source: source}, nil
		}
	}
	return nil, nil
}

// reportPasses reads a structured-evaluation report and returns its outcome.
// Reports that do not carry an explicit decision are treated as not passing,
// so an unrecognized format never yields a false green.
func reportPasses(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}

	var probe struct {
		Pass            *bool  `json:"pass"`
		OverallDecision string `json:"overallDecision"`
		Decision        struct {
			Passed *bool  `json:"passed"`
			Status string `json:"status"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}

	switch {
	case probe.Pass != nil:
		return *probe.Pass, nil
	case probe.Decision.Passed != nil:
		return *probe.Decision.Passed, nil
	case probe.OverallDecision != "":
		return probe.OverallDecision == string(rubric.DecisionPass), nil
	case probe.Decision.Status != "":
		return probe.Decision.Status == string(rubric.DecisionPass), nil
	}
	return false, nil
}

// localeCoverage computes translated-key coverage per target locale by diffing
// each catalog against the source catalog. It returns nil when the project has
// no locale catalogs to compare.
func localeCoverage(root string, proj *project.Project) (map[string]float64, error) {
	src := proj.Spec.Locales.Source
	if src == "" || len(proj.Spec.Locales.Targets) == 0 {
		return nil, nil
	}

	sourceKeys, err := catalogKeys(filepath.Join(root, "locales", "ui", src+".json"))
	if err != nil {
		return nil, err
	}
	if len(sourceKeys) == 0 {
		return nil, nil
	}

	coverage := make(map[string]float64, len(proj.Spec.Locales.Targets))
	for _, target := range proj.Spec.Locales.Targets {
		targetKeys, err := catalogKeys(filepath.Join(root, "locales", "ui", target+".json"))
		if err != nil {
			return nil, err
		}
		var translated int
		for k := range sourceKeys {
			if targetKeys[k] {
				translated++
			}
		}
		coverage[target] = float64(translated) / float64(len(sourceKeys))
	}
	return coverage, nil
}

// catalogKeys returns the translated keys in a locale catalog. It accepts the
// PDLC locale IR (entries keyed by string, counting only entries with a value)
// and degrades gracefully to a flat key/value catalog. A missing file yields no
// keys rather than an error, since an absent translation is a coverage fact.
func catalogKeys(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var ir struct {
		Entries map[string]struct {
			Value  string `json:"value"`
			Status string `json:"status"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ir); err == nil && len(ir.Entries) > 0 {
		keys := make(map[string]bool, len(ir.Entries))
		for k, e := range ir.Entries {
			if e.Value != "" && e.Status != "missing" {
				keys[k] = true
			}
		}
		return keys, nil
	}

	var flat map[string]json.RawMessage
	if err := json.Unmarshal(data, &flat); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	keys := make(map[string]bool, len(flat))
	for k := range flat {
		keys[k] = true
	}
	return keys, nil
}

func decisionFor(rep *rubric.Rubric) rubric.DecisionStatus {
	switch {
	case hasScore(rep, rubric.ScoreFail):
		return rubric.DecisionFail
	case hasScore(rep, rubric.ScorePartial):
		return rubric.DecisionConditional
	}
	return rubric.DecisionPass
}

func hasScore(rep *rubric.Rubric, want rubric.ScoreValue) bool {
	for _, c := range rep.Categories {
		if c.Score == want {
			return true
		}
	}
	return false
}

func summarize(rep *rubric.Rubric, proj *project.Project) string {
	var pass, partial, fail int
	for _, c := range rep.Categories {
		switch c.Score {
		case rubric.ScorePass:
			pass++
		case rubric.ScorePartial:
			partial++
		case rubric.ScoreFail:
			fail++
		}
	}
	var verdict string
	switch {
	case fail > 0:
		verdict = "not ready for baseline"
	case partial > 0:
		verdict = "not ready for baseline: quality evidence missing"
	default:
		verdict = "ready for baseline"
	}
	return fmt.Sprintf(
		"Profile %q: %d domain(s) passing, %d partial, %d failing — %s.",
		proj.Spec.Profile, pass, partial, fail, verdict,
	)
}

func required(l layout.Level) bool { return l == layout.LevelRequired }

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasCode(codes []rubric.ReasonCode, want rubric.ReasonCode) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

func appendUnique(codes []rubric.ReasonCode, add rubric.ReasonCode) []rubric.ReasonCode {
	if hasCode(codes, add) {
		return codes
	}
	return append(codes, add)
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// Write saves a readiness report to the canonical location under root.
func Write(root string, rep *rubric.Rubric) (string, error) {
	dest := filepath.Join(root, filepath.FromSlash(ReportPath))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", filepath.Dir(dest), err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal readiness report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(dest, data, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", dest, err)
	}
	return dest, nil
}
