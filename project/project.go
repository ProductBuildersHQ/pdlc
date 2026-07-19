// Package project models pdlc.yaml, the manifest declaring which product-definition
// domains a project carries and what quality thresholds apply to them.
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ProductBuildersHQ/pdlc/layout"
)

// ManifestFilename is the canonical name of the project manifest.
const ManifestFilename = "pdlc.yaml"

// APIVersion is the manifest contract version this package reads and writes.
const APIVersion = "pdlc.productbuildershq.org/v1"

// Kind is the manifest kind.
const Kind = "ProductProject"

// ErrNotFound reports that a project has no pdlc.yaml.
var ErrNotFound = errors.New("pdlc.yaml not found")

// Project is the pdlc.yaml manifest.
type Project struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion" jsonschema:"required"`
	Kind       string   `json:"kind" yaml:"kind" jsonschema:"required"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       Spec     `json:"spec" yaml:"spec"`
}

// Metadata identifies the product.
type Metadata struct {
	ID    string `json:"id" yaml:"id" jsonschema:"required"`
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
}

// Spec declares the project's scope and thresholds.
type Spec struct {
	// Profile selects which domains are in scope.
	Profile layout.Profile `json:"profile" yaml:"profile" jsonschema:"required"`

	// Artifacts overrides the profile's level for individual domains,
	// keyed by domain ID (for example "guides" or "a11y").
	Artifacts map[string]layout.Level `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`

	// Locales declares the source and target locales.
	Locales Locales `json:"locales,omitempty" yaml:"locales,omitempty"`

	// Quality holds per-tool conformance targets.
	Quality Quality `json:"quality,omitempty" yaml:"quality,omitempty"`

	// Builder names the downstream engineering methodology.
	Builder Builder `json:"builder,omitempty" yaml:"builder,omitempty"`
}

// Locales declares localization scope.
type Locales struct {
	Source  string   `json:"source,omitempty" yaml:"source,omitempty"`
	Targets []string `json:"targets,omitempty" yaml:"targets,omitempty"`
}

// Quality holds conformance targets per evaluation tool.
type Quality struct {
	DesignSystem *DesignSystemTarget `json:"designSystem,omitempty" yaml:"designSystem,omitempty"`
	APIStyle     *APIStyleTarget     `json:"apiStyle,omitempty" yaml:"apiStyle,omitempty"`
	A11y         *A11yTarget         `json:"a11y,omitempty" yaml:"a11y,omitempty"`
	L10n         *L10nTarget         `json:"l10n,omitempty" yaml:"l10n,omitempty"`
}

// DesignSystemTarget configures design-system conformance.
type DesignSystemTarget struct {
	// Spec is a path or external reference to the design-system definition.
	Spec      string `json:"spec,omitempty" yaml:"spec,omitempty"`
	Threshold string `json:"threshold,omitempty" yaml:"threshold,omitempty"`
}

// APIStyleTarget configures API style conformance.
type APIStyleTarget struct {
	// Profile is a named api-style profile; Spec is an in-repo definition.
	Profile string `json:"profile,omitempty" yaml:"profile,omitempty"`
	Spec    string `json:"spec,omitempty" yaml:"spec,omitempty"`
	Level   string `json:"level,omitempty" yaml:"level,omitempty"`
}

// A11yTarget configures the optional prototype accessibility audit.
type A11yTarget struct {
	WCAGVersion string `json:"wcagVersion,omitempty" yaml:"wcagVersion,omitempty"`
	Level       string `json:"level,omitempty" yaml:"level,omitempty"`
}

// L10nTarget configures localization coverage requirements.
type L10nTarget struct {
	MinimumCoverage float64 `json:"minimumCoverage,omitempty" yaml:"minimumCoverage,omitempty"`
}

// Builder names the downstream engineering methodology consuming the baseline.
type Builder struct {
	Methodology string `json:"methodology,omitempty" yaml:"methodology,omitempty"`
}

// New returns a project manifest with the given identity and profile.
func New(id, title string, profile layout.Profile) *Project {
	return &Project{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   Metadata{ID: id, Title: title},
		Spec:       Spec{Profile: profile},
	}
}

// Load reads the project manifest from a repository root. It returns
// ErrNotFound (wrapped) when the repository has no pdlc.yaml.
func Load(root string) (*Project, error) {
	p := filepath.Join(root, ManifestFilename)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w at %s", ErrNotFound, p)
		}
		return nil, fmt.Errorf("read %s: %w", p, err)
	}

	var proj Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if err := proj.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return &proj, nil
}

// Save writes the project manifest to a repository root.
func Save(root string, proj *Project) error {
	if err := proj.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(proj)
	if err != nil {
		return fmt.Errorf("marshal project manifest: %w", err)
	}
	p := filepath.Join(root, ManifestFilename)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}

// Validate checks the manifest for internal consistency.
func (p *Project) Validate() error {
	var problems []string

	if p.Kind != Kind {
		problems = append(problems, fmt.Sprintf("kind must be %q, got %q", Kind, p.Kind))
	}
	if p.APIVersion == "" {
		problems = append(problems, "apiVersion is required")
	}
	if p.Metadata.ID == "" {
		problems = append(problems, "metadata.id is required")
	}
	if !p.Spec.Profile.Valid() {
		problems = append(problems, fmt.Sprintf("unknown profile %q", p.Spec.Profile))
	}
	for domain, lvl := range p.Spec.Artifacts {
		if !lvl.Valid() {
			problems = append(problems, fmt.Sprintf("override %q: unknown level %q", domain, lvl))
		}
	}
	if t := p.Spec.Quality.L10n; t != nil && (t.MinimumCoverage < 0 || t.MinimumCoverage > 1) {
		problems = append(problems, fmt.Sprintf("quality.l10n.minimumCoverage must be between 0 and 1, got %v", t.MinimumCoverage))
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid project manifest: %s", strings.Join(problems, "; "))
	}
	return nil
}

// LevelFor resolves the effective requirement level for a domain: the project's
// explicit override when present, otherwise the level the layout manifest
// assigns under the project's profile.
func (p *Project) LevelFor(m *layout.Manifest, domainID string) layout.Level {
	if lvl, ok := p.Spec.Artifacts[domainID]; ok {
		return lvl
	}
	return DomainLevel(m, domainID, p.Spec.Profile)
}

// DomainLevel returns the level a profile assigns to a domain. Artifact-backed
// domains take the strongest level among their artifacts, so a domain counts as
// required when any of its artifacts is required.
func DomainLevel(m *layout.Manifest, domainID string, profile layout.Profile) layout.Level {
	d, ok := m.Domain(domainID)
	if !ok {
		return layout.LevelOptional
	}
	if d.QualityOnly {
		return d.LevelFor(profile)
	}

	strongest := layout.LevelExcluded
	for _, a := range m.ArtifactsInDomain(domainID) {
		if lvl := a.LevelFor(profile); lvl.Rank() > strongest.Rank() {
			strongest = lvl
		}
	}
	return strongest
}
