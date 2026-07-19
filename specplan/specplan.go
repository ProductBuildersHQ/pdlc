// Package specplan resolves a spec profile (a VisionSpec authoring methodology
// such as "big-tech-product") into the concrete authoring plan for a project:
// for each required spec, where it belongs in the PDLC layout, and whether a
// template and rubric are available to author and judge it.
//
// This is the one package that imports VisionSpec. The core pdlc packages
// (layout, project, readiness) stay free of VisionSpec's heavier dependencies;
// the spec engine enters only here and through the mounted CLI.
package specplan

import (
	"fmt"
	"path"

	vsprofiles "github.com/ProductBuildersHQ/visionspec/pkg/profiles"
	vsrubrics "github.com/ProductBuildersHQ/visionspec/pkg/rubrics"
	vstemplates "github.com/ProductBuildersHQ/visionspec/pkg/templates"
	vstypes "github.com/ProductBuildersHQ/visionspec/pkg/types"
)

// DefaultSpecsRoot is where PDLC places VisionSpec output under a project.
const DefaultSpecsRoot = "docs/specs"

// Artifact is one required spec in an authoring plan.
type Artifact struct {
	// SpecType is the VisionSpec spec type, e.g. "prd".
	SpecType string `json:"specType" yaml:"specType"`

	// Path is the canonical repository-relative location for the spec.
	Path string `json:"path" yaml:"path"`

	// HasTemplate reports whether a template is available to author it.
	HasTemplate bool `json:"hasTemplate" yaml:"hasTemplate"`

	// HasRubric reports whether a rubric is available to judge it.
	HasRubric bool `json:"hasRubric" yaml:"hasRubric"`
}

// Plan is the authoring plan for one spec profile.
type Plan struct {
	// SpecProfile is the resolved profile name, e.g. "big-tech-product".
	SpecProfile string `json:"specProfile" yaml:"specProfile"`

	// SpecsRoot is the root the artifact paths are rooted at.
	SpecsRoot string `json:"specsRoot" yaml:"specsRoot"`

	// Artifacts are the required specs, in the profile's declared order.
	Artifacts []Artifact `json:"artifacts" yaml:"artifacts"`
}

// ListProfiles returns the available VisionSpec authoring profile names.
func ListProfiles() []string {
	return vsprofiles.DefaultProfileNames
}

// Resolve builds the authoring plan for a VisionSpec profile. specsRoot may be
// empty to use DefaultSpecsRoot.
func Resolve(profileName, specsRoot string) (*Plan, error) {
	if specsRoot == "" {
		specsRoot = DefaultSpecsRoot
	}

	profile, err := vsprofiles.DefaultLoader().Load(profileName)
	if err != nil {
		return nil, fmt.Errorf("load spec profile %q: %w", profileName, err)
	}

	// Chain the profile's own loaders over the embedded defaults, so a spec the
	// profile does not carry directly still resolves from the default set.
	templateLoader := vstemplates.NewChainLoader(profile.GetTemplateLoader(), vstemplates.DefaultLoader())
	rubricLoader := vsrubrics.NewChainLoader(profile.GetRubricLoader(), vsrubrics.DefaultLoader())

	plan := &Plan{SpecProfile: profileName, SpecsRoot: specsRoot}
	for _, name := range profile.RequiredSpecs() {
		specType := vstypes.SpecType(name)

		art := Artifact{
			SpecType: name,
			Path:     canonicalPath(specsRoot, specType),
		}
		if _, err := templateLoader.Load(specType); err == nil {
			art.HasTemplate = true
		}
		if _, err := rubricLoader.Load(specType); err == nil {
			art.HasRubric = true
		}
		plan.Artifacts = append(plan.Artifacts, art)
	}
	return plan, nil
}

// Template returns the template content for a spec type under a profile, chaining
// the profile's templates over the embedded defaults.
func Template(profileName, specType string) (string, error) {
	profile, err := vsprofiles.DefaultLoader().Load(profileName)
	if err != nil {
		return "", fmt.Errorf("load spec profile %q: %w", profileName, err)
	}
	loader := vstemplates.NewChainLoader(profile.GetTemplateLoader(), vstemplates.DefaultLoader())
	tmpl, err := loader.Load(vstypes.SpecType(specType))
	if err != nil {
		return "", fmt.Errorf("load template %q for profile %q: %w", specType, profileName, err)
	}
	return tmpl.Content, nil
}

// canonicalPath computes where a spec type belongs under specsRoot, using
// VisionSpec's own directory and filename routing so PDLC and VisionSpec agree.
// Spec types that route to the project root (output specs) are placed directly
// under specsRoot.
func canonicalPath(specsRoot string, specType vstypes.SpecType) string {
	dir := specType.Dir()
	if dir == "" {
		return path.Join(specsRoot, specType.Filename())
	}
	return path.Join(specsRoot, dir, specType.Filename())
}
