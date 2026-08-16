package pdlc_test

import (
	"testing"

	"github.com/ProductBuildersHQ/pdlc"
)

func TestStages(t *testing.T) {
	stages, err := pdlc.Stages()
	if err != nil {
		t.Fatalf("Stages() error: %v", err)
	}
	if len(stages) != 6 {
		t.Fatalf("len(Stages()) = %d, want 6", len(stages))
	}

	expected := []string{
		pdlc.StageProductDefinition,
		pdlc.StageBuilderDefinition,
		pdlc.StageImplementation,
		pdlc.StageDeployment,
		pdlc.StageBuilderOperations,
		pdlc.StageProductOperations,
	}
	for i, s := range stages {
		if s.ID != expected[i] {
			t.Errorf("Stages()[%d].ID = %q, want %q", i, s.ID, expected[i])
		}
	}
}

func TestMustStages(t *testing.T) {
	// Should not panic — the embedded catalog is valid.
	stages := pdlc.MustStages()
	if len(stages) != 6 {
		t.Errorf("len(MustStages()) = %d, want 6", len(stages))
	}
}

func TestStageByID(t *testing.T) {
	s, ok := pdlc.StageByID(pdlc.StageBuilderDefinition)
	if !ok {
		t.Fatal("StageByID(StageBuilderDefinition) not found")
	}
	if s.Name != "Builder Definition" {
		t.Errorf("Name = %q, want %q", s.Name, "Builder Definition")
	}
	if s.Role != "builder" {
		t.Errorf("Role = %q, want %q", s.Role, "builder")
	}

	if _, ok := pdlc.StageByID("not-a-real-stage"); ok {
		t.Error("StageByID(\"not-a-real-stage\") should not be found")
	}
}

func TestStageIDConstantsMatchCatalog(t *testing.T) {
	// Every exported stage-ID constant must resolve to a real catalog stage,
	// so downstream consumers (specification-workflow-spec, Threat Model
	// Spec) can trust the constants without a runtime lookup failing.
	ids := []string{
		pdlc.StageProductDefinition,
		pdlc.StageBuilderDefinition,
		pdlc.StageImplementation,
		pdlc.StageDeployment,
		pdlc.StageBuilderOperations,
		pdlc.StageProductOperations,
	}
	for _, id := range ids {
		if _, ok := pdlc.StageByID(id); !ok {
			t.Errorf("stage ID constant %q does not resolve via StageByID", id)
		}
	}
}
