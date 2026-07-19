//go:build tools

// Package pdlc tracks build-time-only dependencies so `go mod tidy` retains
// them. These are used by code generators (see schema/generate.go) that are
// excluded from ordinary builds by their own build constraints.
package pdlc

import (
	_ "github.com/invopop/jsonschema"
)
