package pdlc

import (
	"fmt"

	frameworks "github.com/ProductBuildersHQ/productbuildershq-frameworks"
)

// Stage IDs for the six PDLC stages. These are the canonical, stable
// identifiers downstream consumers import rather than re-declare — for
// example, specification-workflow-spec tags each spec type with one of
// these, and Threat Model Spec's ASPM security-posture domains map onto the
// three builder-side stages (Implementation, Deployment, Builder Operations).
const (
	StageProductDefinition = "product-definition"
	StageBuilderDefinition = "builder-definition"
	StageImplementation    = "implementation"
	StageDeployment        = "deployment"
	StageBuilderOperations = "builder-operations"
	StageProductOperations = "product-operations"
)

// Stage is a single PDLC lifecycle stage. The full stage catalog —
// deliverables, gates, the dependency graph, and the AI-DLC crosswalk — is
// defined once in the productbuildershq-frameworks PDLC entry; this package
// re-exports it so consumers depend on pdlc, the normative specification
// module, rather than on the catalog module directly.
type Stage = frameworks.PDLCPhase

// Stages returns the six canonical PDLC stages in order: Product Definition,
// Builder Definition, Implementation, Deployment, Builder Operations, and
// Product Operations (the last two run in parallel, not sequentially).
func Stages() ([]Stage, error) {
	f, err := frameworks.PDLC()
	if err != nil {
		return nil, fmt.Errorf("load PDLC framework: %w", err)
	}
	return f.Phases, nil
}

// MustStages returns the six canonical PDLC stages and panics if the
// embedded catalog is invalid — a build-time defect, not a runtime
// condition, so callers in tools may use this directly.
func MustStages() []Stage {
	s, err := Stages()
	if err != nil {
		panic(err)
	}
	return s
}

// StageByID returns the stage with the given ID and true, or a zero Stage
// and false if no stage has that ID.
func StageByID(id string) (Stage, bool) {
	for _, s := range MustStages() {
		if s.ID == id {
			return s, true
		}
	}
	return Stage{}, false
}
