// Package project models pdlc.yaml, the manifest declaring which product-definition
// domains a project carries and what quality thresholds apply to them.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ProductBuildersHQ/pdlc/layout"
)

// ManifestFilename is the default project manifest name.
const ManifestFilename = "pdlc.yaml"

// ManifestFilenames are the manifest names Load looks for, in order. Both YAML
// and JSON are supported; the extension selects the parser.
var ManifestFilenames = []string{"pdlc.yaml", "pdlc.yml", "pdlc.json"}

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
	// Profile is the PDLC profile: which domains are in scope
	// (minimal, standard, full, custom).
	Profile layout.Profile `json:"profile" yaml:"profile" jsonschema:"required"`

	// SpecProfiles are the pluggable methodologies the project uses. Each is
	// either a VisionSpec authoring profile (big-tech-product, big-tech-feature)
	// or a third-party builder methodology (aws-aidlc, github-speckit). A bare
	// string is shorthand for {name, provider: visionspec, role: authoring}.
	SpecProfiles []SpecProfile `json:"specProfiles,omitempty" yaml:"specProfiles,omitempty"`

	// Artifacts overrides the profile's level for individual domains,
	// keyed by domain ID (for example "guides" or "a11y").
	Artifacts map[string]layout.Level `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`

	// Locales declares the source and target locales.
	Locales Locales `json:"locales,omitempty" yaml:"locales,omitempty"`

	// Quality holds per-tool conformance targets.
	Quality Quality `json:"quality,omitempty" yaml:"quality,omitempty"`
}

// Provider identifies who resolves a spec profile.
type Provider string

const (
	// ProviderVisionSpec resolves a spec profile to VisionSpec templates and rubrics.
	ProviderVisionSpec Provider = "visionspec"

	// ProviderAIDLC resolves to the AWS AI-DLC builder methodology.
	ProviderAIDLC Provider = "aidlc"

	// ProviderSpecKit resolves to the GitHub Spec Kit builder methodology.
	ProviderSpecKit Provider = "speckit"
)

// Role is what a spec profile contributes to the lifecycle.
type Role string

const (
	// RoleAuthoring produces product-definition specs (templates + rubrics).
	RoleAuthoring Role = "authoring"

	// RoleBuilder consumes the baseline and produces implementation artifacts.
	RoleBuilder Role = "builder"
)

// SpecProfile is one pluggable methodology.
type SpecProfile struct {
	// Name is the methodology name, e.g. "big-tech-product" or "aws-aidlc".
	Name string `json:"name" yaml:"name"`

	// Provider resolves the methodology; defaults to visionspec.
	Provider Provider `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Role is what it contributes; defaults to authoring.
	Role Role `json:"role,omitempty" yaml:"role,omitempty"`
}

// ResolvedProvider returns the provider, defaulting to visionspec.
func (s SpecProfile) ResolvedProvider() Provider {
	if s.Provider == "" {
		return ProviderVisionSpec
	}
	return s.Provider
}

// ResolvedRole returns the role, defaulting to authoring.
func (s SpecProfile) ResolvedRole() Role {
	if s.Role == "" {
		return RoleAuthoring
	}
	return s.Role
}

// UnmarshalYAML accepts either a bare string (name shorthand) or a mapping.
func (s *SpecProfile) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		s.Name = value.Value
		return nil
	}
	type alias SpecProfile
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*s = SpecProfile(a)
	return nil
}

// UnmarshalJSON accepts either a bare string (name shorthand) or an object.
func (s *SpecProfile) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		s.Name = name
		return nil
	}
	type alias SpecProfile
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = SpecProfile(a)
	return nil
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

// New returns a project manifest with the given identity and profile.
func New(id, title string, profile layout.Profile) *Project {
	return &Project{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   Metadata{ID: id, Title: title},
		Spec:       Spec{Profile: profile},
	}
}

// Load reads the project manifest from a repository root, accepting either YAML
// (pdlc.yaml, pdlc.yml) or JSON (pdlc.json). It returns ErrNotFound (wrapped)
// when no manifest is present.
func Load(root string) (*Project, error) {
	path, err := FindManifest(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var proj Project
	if err := unmarshalByExt(path, data, &proj); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := proj.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &proj, nil
}

// FindManifest returns the path of the first manifest present under root, or a
// wrapped ErrNotFound.
func FindManifest(root string) (string, error) {
	for _, name := range ManifestFilenames {
		p := filepath.Join(root, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", p, err)
		}
	}
	return "", fmt.Errorf("%w in %s", ErrNotFound, root)
}

// Save writes the project manifest to a repository root. The format follows the
// filename: .json is written as JSON, everything else as YAML. When name is
// empty the default (pdlc.yaml) is used.
func Save(root, name string, proj *Project) error {
	if err := proj.Validate(); err != nil {
		return err
	}
	if name == "" {
		name = ManifestFilename
	}
	path := filepath.Join(root, name)

	data, err := marshalByExt(path, proj)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func unmarshalByExt(path string, data []byte, v any) error {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return json.Unmarshal(data, v)
	}
	return yaml.Unmarshal(data, v)
}

func marshalByExt(path string, v any) ([]byte, error) {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal project manifest: %w", err)
		}
		return append(data, '\n'), nil
	}
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal project manifest: %w", err)
	}
	return data, nil
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
	for i, sp := range p.Spec.SpecProfiles {
		if sp.Name == "" {
			problems = append(problems, fmt.Sprintf("specProfiles[%d]: name is required", i))
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
