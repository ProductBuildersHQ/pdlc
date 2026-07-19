// Package schema embeds the JSON Schemas generated from the PDLC Go types.
//
// The schemas are generated artifacts: edit the Go types in the layout and
// project packages, then regenerate. Never hand-edit the .schema.json files.
package schema

import (
	"embed"
	"fmt"
)

//go:generate go run generate.go

//go:embed *.schema.json
var files embed.FS

// FS returns the embedded schema filesystem.
func FS() embed.FS { return files }

// Read returns the named schema, for example "layout-manifest.schema.json".
func Read(name string) ([]byte, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read schema %q: %w", name, err)
	}
	return data, nil
}
