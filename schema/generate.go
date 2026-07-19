//go:build ignore

// Command generate produces JSON Schema files from the Go types that define the
// PDLC contract. Go types are the source of truth; the schemas are generated
// artifacts and must never be hand-edited.
//
// Run with: go generate ./schema/...
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"

	"github.com/ProductBuildersHQ/pdlc/layout"
	"github.com/ProductBuildersHQ/pdlc/project"
)

const schemaBaseURL = "https://productbuildershq.org/schema/"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	targets := []struct {
		filename string
		value    any
	}{
		{"layout-manifest.schema.json", &layout.Manifest{}},
		{"product-project.schema.json", &project.Project{}},
	}

	for _, t := range targets {
		r := &jsonschema.Reflector{
			ExpandedStruct: true,
			DoNotReference: false,
		}
		s := r.Reflect(t.value)
		s.ID = jsonschema.ID(schemaBaseURL + t.filename)

		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", t.filename, err)
		}
		data = append(data, '\n')

		dest := filepath.Join("schema", t.filename)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		fmt.Println("generated", dest)
	}
	return nil
}
