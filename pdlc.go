// Package pdlc provides the ProductBuildersHQ Product Development Lifecycle
// contract: the canonical layout manifest and the version of the specification
// this module implements.
//
// The layout contract is embedded, so tools depending on this module carry the
// contract with them and need no network or checkout of the specification repo.
package pdlc

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/ProductBuildersHQ/pdlc/layout"
)

// SpecVersion is the PDLC specification version this module implements.
const SpecVersion = "0.1.0"

//go:embed layout.yaml
var layoutYAML []byte

var (
	layoutOnce sync.Once
	layoutVal  *layout.Manifest
	layoutErr  error
)

// LayoutYAML returns the raw embedded layout manifest.
func LayoutYAML() []byte {
	out := make([]byte, len(layoutYAML))
	copy(out, layoutYAML)
	return out
}

// Layout returns the parsed and validated canonical layout manifest.
// The result is parsed once and cached.
func Layout() (*layout.Manifest, error) {
	layoutOnce.Do(func() {
		var m layout.Manifest
		if err := yaml.Unmarshal(layoutYAML, &m); err != nil {
			layoutErr = fmt.Errorf("parse embedded layout.yaml: %w", err)
			return
		}
		if err := m.Validate(); err != nil {
			layoutErr = fmt.Errorf("embedded layout.yaml: %w", err)
			return
		}
		layoutVal = &m
	})
	return layoutVal, layoutErr
}

// MustLayout returns the canonical layout manifest and panics if the embedded
// contract is invalid. An invalid embedded contract is a build-time defect, not
// a runtime condition, so callers in tools may use this directly.
func MustLayout() *layout.Manifest {
	m, err := Layout()
	if err != nil {
		panic(err)
	}
	return m
}
