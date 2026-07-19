// Package layout defines the PDLC layout contract: where every product-definition
// artifact belongs in a project repository, how to recognize a misplaced one, and
// what each project profile requires.
//
// Go types are the source of truth; JSON Schema is generated from them.
package layout

import (
	"fmt"
	"path"
	"strings"
)

// Profile names the breadth of product-definition work a project carries.
// The layout contract is identical at every profile; profiles select which
// domains are in scope.
type Profile string

const (
	// ProfileMinimal is specs, API, and baseline only.
	ProfileMinimal Profile = "minimal"

	// ProfileStandard adds prototype, guides, and descriptive requirements.
	ProfileStandard Profile = "standard"

	// ProfileFull is the reference practice: every domain including
	// localization, personas, and narrative artifacts.
	ProfileFull Profile = "full"

	// ProfileCustom relies entirely on per-domain overrides.
	ProfileCustom Profile = "custom"
)

// Profiles lists the built-in profiles in increasing breadth.
var Profiles = []Profile{ProfileMinimal, ProfileStandard, ProfileFull}

// Valid reports whether p is a known profile.
func (p Profile) Valid() bool {
	switch p {
	case ProfileMinimal, ProfileStandard, ProfileFull, ProfileCustom:
		return true
	}
	return false
}

// Level is how strongly a profile expects an artifact or domain.
type Level string

const (
	// LevelRequired means a stage gate fails when the artifact is absent.
	LevelRequired Level = "required"

	// LevelRecommended means the gate reports absence but does not fail.
	LevelRecommended Level = "recommended"

	// LevelOptional means the artifact is evaluated only when present.
	LevelOptional Level = "optional"

	// LevelExcluded means the domain is out of scope; it is reported as
	// excluded rather than as a failure.
	LevelExcluded Level = "excluded"
)

// Valid reports whether l is a known level.
func (l Level) Valid() bool {
	switch l {
	case LevelRequired, LevelRecommended, LevelOptional, LevelExcluded:
		return true
	}
	return false
}

// Rank orders levels by strength, so they can be compared and combined.
func (l Level) Rank() int {
	switch l {
	case LevelRequired:
		return 4
	case LevelRecommended:
		return 3
	case LevelOptional:
		return 2
	case LevelExcluded:
		return 1
	}
	return 0
}

// ApplyOverride combines a profile-derived level with a project override.
//
// An override of excluded removes the artifact from scope. An override of
// required promotes anything still in scope. A weaker override demotes, so a
// project can keep a domain in scope without letting it block a gate. An
// artifact excluded by the profile stays excluded: an override adjusts emphasis,
// it does not introduce artifacts the profile does not define.
func ApplyOverride(base, override Level) Level {
	switch override {
	case LevelExcluded:
		return LevelExcluded
	case LevelRequired:
		if base == LevelExcluded {
			return LevelExcluded
		}
		return LevelRequired
	case LevelRecommended, LevelOptional:
		if base.Rank() > override.Rank() {
			return override
		}
	}
	return base
}

// Authority is how binding an artifact is on the builder after handoff.
type Authority string

const (
	// AuthorityNormative is a binding requirement.
	AuthorityNormative Authority = "normative"

	// AuthorityInformative is context and rationale.
	AuthorityInformative Authority = "informative"

	// AuthorityAdvisory is a recommended approach the builder may replace.
	AuthorityAdvisory Authority = "advisory"

	// AuthorityEvidence is machine-written proof, not a requirement.
	AuthorityEvidence Authority = "evidence"
)

// Valid reports whether a is a known authority.
func (a Authority) Valid() bool {
	switch a {
	case AuthorityNormative, AuthorityInformative, AuthorityAdvisory, AuthorityEvidence:
		return true
	}
	return false
}

// Manifest is the complete layout contract.
type Manifest struct {
	// APIVersion identifies the contract version.
	APIVersion string `json:"apiVersion" yaml:"apiVersion" jsonschema:"required"`

	// Kind is always "LayoutManifest".
	Kind string `json:"kind" yaml:"kind" jsonschema:"required"`

	// Version is the manifest revision.
	Version int `json:"version" yaml:"version"`

	// Immovable lists repository-relative directories that must never be
	// moved, reorganized, or written to. These are tool-native contracts
	// (AWS AI-DLC, Spec Kit, VisionSpec working state) plus machine-written
	// projections.
	Immovable []string `json:"immovable" yaml:"immovable"`

	// Ignore lists directory base names that never hold a project's own
	// product artifacts, such as templates, examples, and test fixtures.
	// Classification skips them entirely.
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`

	// Domains are the evaluation groupings. The readiness report emits one
	// category per non-excluded domain.
	Domains []Domain `json:"domains" yaml:"domains"`

	// Artifacts are the artifact kinds with their canonical locations.
	Artifacts []Artifact `json:"artifacts" yaml:"artifacts"`
}

// Domain groups artifacts into an evaluation category.
type Domain struct {
	// ID is the domain identifier, used as the readiness category name.
	ID string `json:"id" yaml:"id" jsonschema:"required"`

	// Title is the human-readable name.
	Title string `json:"title,omitempty" yaml:"title,omitempty"`

	// QualityOnly marks a domain that has no canonical artifacts and exists
	// only as an evaluation (for example accessibility or consistency).
	QualityOnly bool `json:"qualityOnly,omitempty" yaml:"qualityOnly,omitempty"`

	// LeafReports are repository-relative paths holding this domain's
	// tool-produced evaluation evidence.
	LeafReports []string `json:"leafReports,omitempty" yaml:"leafReports,omitempty"`

	// Profiles sets the requirement level per profile. Used for quality-only
	// domains; artifact-backed domains derive their level from artifacts.
	Profiles map[Profile]Level `json:"profiles,omitempty" yaml:"profiles,omitempty"`
}

// Artifact is one kind of product-definition artifact.
type Artifact struct {
	// ID uniquely identifies the artifact kind.
	ID string `json:"id" yaml:"id" jsonschema:"required"`

	// Domain is the evaluation domain this artifact belongs to.
	Domain string `json:"domain" yaml:"domain" jsonschema:"required"`

	// Canonical is the repository-relative path where the artifact belongs.
	Canonical string `json:"canonical" yaml:"canonical" jsonschema:"required"`

	// Authority is how binding the artifact is after handoff.
	Authority Authority `json:"authority" yaml:"authority"`

	// Profiles sets the requirement level per profile.
	Profiles map[Profile]Level `json:"profiles" yaml:"profiles"`

	// Detect holds heuristics for recognizing a misplaced instance.
	Detect Detect `json:"detect,omitempty" yaml:"detect,omitempty"`
}

// Detect holds heuristics for classifying an existing file or directory as an
// artifact kind. Rules are additive: any match proposes the artifact, while
// Under and NotUnder constrain where a match is credible.
type Detect struct {
	// Filenames are base names that identify the artifact (case-insensitive).
	Filenames []string `json:"filenames,omitempty" yaml:"filenames,omitempty"`

	// Dirnames are directory base names that identify the artifact.
	Dirnames []string `json:"dirnames,omitempty" yaml:"dirnames,omitempty"`

	// Content are substrings whose presence identifies the artifact.
	Content []string `json:"content,omitempty" yaml:"content,omitempty"`

	// Under restricts matches to paths containing one of these segments.
	Under []string `json:"under,omitempty" yaml:"under,omitempty"`

	// NotUnder rejects matches under any of these segments.
	NotUnder []string `json:"notUnder,omitempty" yaml:"not_under,omitempty"`
}

// Empty reports whether no detection heuristics are configured.
func (d Detect) Empty() bool {
	return len(d.Filenames) == 0 && len(d.Dirnames) == 0 && len(d.Content) == 0
}

// LevelFor returns the requirement level of the artifact under profile.
// Unlisted profiles default to LevelOptional.
func (a Artifact) LevelFor(p Profile) Level {
	if lvl, ok := a.Profiles[p]; ok {
		return lvl
	}
	return LevelOptional
}

// LevelFor returns the requirement level of the domain under profile.
// Unlisted profiles default to LevelOptional.
func (d Domain) LevelFor(p Profile) Level {
	if lvl, ok := d.Profiles[p]; ok {
		return lvl
	}
	return LevelOptional
}

// Artifact returns the artifact with the given ID.
func (m *Manifest) Artifact(id string) (Artifact, bool) {
	for _, a := range m.Artifacts {
		if a.ID == id {
			return a, true
		}
	}
	return Artifact{}, false
}

// Domain returns the domain with the given ID.
func (m *Manifest) Domain(id string) (Domain, bool) {
	for _, d := range m.Domains {
		if d.ID == id {
			return d, true
		}
	}
	return Domain{}, false
}

// ArtifactsInDomain returns every artifact belonging to domain id.
func (m *Manifest) ArtifactsInDomain(id string) []Artifact {
	var out []Artifact
	for _, a := range m.Artifacts {
		if a.Domain == id {
			out = append(out, a)
		}
	}
	return out
}

// IsImmovable reports whether the repository-relative path lies within a
// directory that must never be moved or written to.
func (m *Manifest) IsImmovable(relPath string) bool {
	clean := path.Clean(strings.ReplaceAll(relPath, "\\", "/"))
	clean = strings.TrimPrefix(clean, "./")
	for _, dir := range m.Immovable {
		d := strings.Trim(path.Clean(dir), "/")
		if clean == d || strings.HasPrefix(clean, d+"/") {
			return true
		}
	}
	return false
}

// IsIgnored reports whether any segment of the repository-relative path is an
// ignored directory name, meaning the path cannot hold a product artifact.
func (m *Manifest) IsIgnored(relPath string) bool {
	segments := strings.Split(strings.ReplaceAll(relPath, "\\", "/"), "/")
	for _, seg := range segments {
		for _, ign := range m.Ignore {
			if strings.EqualFold(seg, ign) {
				return true
			}
		}
	}
	return false
}

// Validate checks the manifest for internal consistency. It reports every
// problem found rather than stopping at the first.
func (m *Manifest) Validate() error {
	var problems []string

	if m.Kind != "LayoutManifest" {
		problems = append(problems, fmt.Sprintf("kind must be %q, got %q", "LayoutManifest", m.Kind))
	}
	if m.APIVersion == "" {
		problems = append(problems, "apiVersion is required")
	}

	domains := make(map[string]bool, len(m.Domains))
	for _, d := range m.Domains {
		if d.ID == "" {
			problems = append(problems, "domain with empty id")
			continue
		}
		if domains[d.ID] {
			problems = append(problems, fmt.Sprintf("duplicate domain id %q", d.ID))
		}
		domains[d.ID] = true
		for p, lvl := range d.Profiles {
			if !p.Valid() {
				problems = append(problems, fmt.Sprintf("domain %q: unknown profile %q", d.ID, p))
			}
			if !lvl.Valid() {
				problems = append(problems, fmt.Sprintf("domain %q: unknown level %q", d.ID, lvl))
			}
		}
	}

	ids := make(map[string]bool, len(m.Artifacts))
	for _, a := range m.Artifacts {
		switch {
		case a.ID == "":
			problems = append(problems, "artifact with empty id")
			continue
		case ids[a.ID]:
			problems = append(problems, fmt.Sprintf("duplicate artifact id %q", a.ID))
		}
		ids[a.ID] = true

		if a.Canonical == "" {
			problems = append(problems, fmt.Sprintf("artifact %q: canonical path is required", a.ID))
		}
		if a.Domain == "" {
			problems = append(problems, fmt.Sprintf("artifact %q: domain is required", a.ID))
		} else if !domains[a.Domain] {
			problems = append(problems, fmt.Sprintf("artifact %q: unknown domain %q", a.ID, a.Domain))
		}
		if a.Authority != "" && !a.Authority.Valid() {
			problems = append(problems, fmt.Sprintf("artifact %q: unknown authority %q", a.ID, a.Authority))
		}
		for p, lvl := range a.Profiles {
			if !p.Valid() {
				problems = append(problems, fmt.Sprintf("artifact %q: unknown profile %q", a.ID, p))
			}
			if !lvl.Valid() {
				problems = append(problems, fmt.Sprintf("artifact %q: unknown level %q", a.ID, lvl))
			}
		}
		if m.IsImmovable(a.Canonical) {
			problems = append(problems, fmt.Sprintf("artifact %q: canonical path %q is inside an immovable directory", a.ID, a.Canonical))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid layout manifest: %s", strings.Join(problems, "; "))
	}
	return nil
}
