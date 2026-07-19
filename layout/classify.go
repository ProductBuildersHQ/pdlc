package layout

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Confidence expresses how strongly an inventory entry matched an artifact kind.
type Confidence string

const (
	// ConfidenceHigh comes from an exact filename match or an already-canonical path.
	ConfidenceHigh Confidence = "high"

	// ConfidenceMedium comes from a directory-name or content match.
	ConfidenceMedium Confidence = "medium"

	// ConfidenceLow means the match is a guess and needs human confirmation.
	ConfidenceLow Confidence = "low"
)

// Entry is one classified path in a project repository.
type Entry struct {
	// Path is the repository-relative path found on disk.
	Path string `json:"path"`

	// ArtifactID is the matched artifact kind, empty when unclassified.
	ArtifactID string `json:"artifactId,omitempty"`

	// Canonical is where the artifact belongs, empty when unclassified.
	Canonical string `json:"canonical,omitempty"`

	// Confidence is the strength of the classification.
	Confidence Confidence `json:"confidence,omitempty"`

	// Conformant reports whether Path is already at its canonical location.
	Conformant bool `json:"conformant"`

	// Reason explains why the classification matched.
	Reason string `json:"reason,omitempty"`

	// Alternatives are other artifact kinds that also matched. A non-empty
	// list means the classification is ambiguous and needs human resolution.
	Alternatives []string `json:"alternatives,omitempty"`

	// Conflicts are other paths that classified to the same canonical
	// location. Only one file can occupy a canonical path, so a conflict
	// always requires a human to choose.
	Conflicts []string `json:"conflicts,omitempty"`
}

// NeedsMove reports whether the entry can be moved without human input: it is
// classified, misplaced, confidently matched, and uncontested. Weak or
// contested matches are reported as ambiguities instead, so a move plan never
// acts on a guess.
func (e Entry) NeedsMove() bool {
	return e.ArtifactID != "" && !e.Conformant && !e.Ambiguous()
}

// Ambiguous reports whether the entry needs human resolution before it can be
// moved, because it matched several artifact kinds, matched only weakly, or
// contests a canonical path with another file.
func (e Entry) Ambiguous() bool {
	return len(e.Alternatives) > 0 || len(e.Conflicts) > 0 || e.Confidence == ConfidenceLow
}

// Inventory is the classification of a project repository.
type Inventory struct {
	// Root is the absolute path of the scanned repository.
	Root string `json:"root"`

	// Entries are the classified paths, sorted by path.
	Entries []Entry `json:"entries"`
}

// Moves returns the entries that are classified but not at their canonical path.
func (inv Inventory) Moves() []Entry {
	var out []Entry
	for _, e := range inv.Entries {
		if e.NeedsMove() {
			out = append(out, e)
		}
	}
	return out
}

// Ambiguities returns entries a human must decide between: those matching more
// than one artifact kind, or contesting a canonical path with another file.
func (inv Inventory) Ambiguities() []Entry {
	var out []Entry
	for _, e := range inv.Entries {
		if e.ArtifactID != "" && (len(e.Alternatives) > 0 || len(e.Conflicts) > 0) {
			out = append(out, e)
		}
	}
	return out
}

// Possibles returns entries matched only by weak evidence, such as a content
// signature in a file whose name suggests nothing. They are worth a look during
// adoption but are never proposed as moves.
func (inv Inventory) Possibles() []Entry {
	var out []Entry
	for _, e := range inv.Entries {
		if e.ArtifactID != "" && e.Confidence == ConfidenceLow &&
			len(e.Alternatives) == 0 && len(e.Conflicts) == 0 {
			out = append(out, e)
		}
	}
	return out
}

// Found reports whether any entry was classified as the given artifact kind.
func (inv Inventory) Found(artifactID string) (Entry, bool) {
	for _, e := range inv.Entries {
		if e.ArtifactID == artifactID {
			return e, true
		}
	}
	return Entry{}, false
}

// maxContentScanBytes bounds how much of a file is read for content detection.
const maxContentScanBytes = 64 * 1024

// Classify walks root and classifies every file and directory against the
// manifest's artifact kinds. Immovable directories are skipped entirely, so
// their contents can never appear in a move plan.
func (m *Manifest) Classify(root string) (Inventory, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, fmt.Errorf("resolve root %q: %w", root, err)
	}

	inv := Inventory{Root: abs}
	seenDir := make(map[string]bool)

	walkErr := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %w", p, err)
		}

		rel, relErr := filepath.Rel(abs, p)
		if relErr != nil {
			return fmt.Errorf("relativize %q: %w", p, relErr)
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		if m.IsImmovable(rel) || m.IsIgnored(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		entry, matched := m.classifyPath(rel, p, d)
		if !matched {
			return nil
		}

		// Record a directory-level artifact once; do not also classify its children.
		if d.IsDir() {
			if seenDir[rel] {
				return nil
			}
			seenDir[rel] = true
			inv.Entries = append(inv.Entries, entry)
			return nil
		}

		// Skip files already covered by a classified ancestor directory.
		for dir := range seenDir {
			if strings.HasPrefix(rel, dir+"/") {
				return nil
			}
		}
		inv.Entries = append(inv.Entries, entry)
		return nil
	})
	if walkErr != nil {
		return Inventory{}, walkErr
	}

	markConflicts(inv.Entries)
	sort.Slice(inv.Entries, func(i, j int) bool { return inv.Entries[i].Path < inv.Entries[j].Path })
	return inv, nil
}

// markConflicts records, on each entry, the other paths competing for the same
// canonical location. Only one file can occupy a canonical path, so every
// member of a contested group needs a human decision.
//
// Weak (content-only) matches are excluded: a source file that merely mentions
// a heading is not a genuine claimant, and treating it as one would bury real
// conflicts in noise.
func markConflicts(entries []Entry) {
	byCanonical := make(map[string][]int)
	for i, e := range entries {
		if e.ArtifactID == "" || e.Conformant || e.Confidence == ConfidenceLow {
			continue
		}
		byCanonical[e.Canonical] = append(byCanonical[e.Canonical], i)
	}

	for _, idxs := range byCanonical {
		if len(idxs) < 2 {
			continue
		}
		for _, i := range idxs {
			for _, j := range idxs {
				if i != j {
					entries[i].Conflicts = append(entries[i].Conflicts, entries[j].Path)
				}
			}
		}
	}
}

// classifyPath matches a single path against artifact kinds. rel is the
// repository-relative path used for matching; abs is used to read content.
func (m *Manifest) classifyPath(rel, abs string, d fs.DirEntry) (Entry, bool) {
	type match struct {
		artifact   Artifact
		confidence Confidence
		reason     string
	}
	var matches []match

	base := strings.ToLower(filepath.Base(rel))

	for _, a := range m.Artifacts {
		if a.Detect.Empty() {
			continue
		}
		if !underAllowed(rel, a.Detect) {
			continue
		}

		canonical := strings.Trim(a.Canonical, "/")
		if rel == canonical {
			matches = append(matches, match{a, ConfidenceHigh, "already at canonical path"})
			continue
		}

		if d.IsDir() {
			if matchesAny(base, a.Detect.Dirnames) {
				matches = append(matches, match{a, ConfidenceMedium, fmt.Sprintf("directory name %q", base)})
			}
			continue
		}

		if matchesAny(base, a.Detect.Filenames) {
			matches = append(matches, match{a, ConfidenceHigh, fmt.Sprintf("file name %q", base)})
			continue
		}
		// A content signature alone is weak evidence: prose that mentions
		// "Functional Requirements" is not necessarily a PRD. Such matches are
		// surfaced for human resolution rather than proposed as moves.
		if len(a.Detect.Content) > 0 && contentMatches(abs, a.Detect.Content) {
			matches = append(matches, match{a, ConfidenceLow, "content signature only"})
		}
	}

	if len(matches) == 0 {
		return Entry{}, false
	}

	best := matches[0]
	for _, mt := range matches[1:] {
		if rank(mt.confidence) > rank(best.confidence) {
			best = mt
		}
	}

	entry := Entry{
		Path:       rel,
		ArtifactID: best.artifact.ID,
		Canonical:  best.artifact.Canonical,
		Confidence: best.confidence,
		Conformant: rel == strings.Trim(best.artifact.Canonical, "/"),
		Reason:     best.reason,
	}
	for _, mt := range matches {
		if mt.artifact.ID != best.artifact.ID {
			entry.Alternatives = append(entry.Alternatives, mt.artifact.ID)
		}
	}
	return entry, true
}

func rank(c Confidence) int {
	switch c {
	case ConfidenceHigh:
		return 3
	case ConfidenceMedium:
		return 2
	case ConfidenceLow:
		return 1
	}
	return 0
}

// matchesAny reports whether base equals any candidate, case-insensitively.
// A candidate may use a "*" suffix or prefix as a simple wildcard.
func matchesAny(base string, candidates []string) bool {
	for _, c := range candidates {
		lc := strings.ToLower(c)
		switch {
		case strings.HasPrefix(lc, "*") && strings.HasSuffix(base, strings.TrimPrefix(lc, "*")):
			return true
		case strings.HasSuffix(lc, "*") && strings.HasPrefix(base, strings.TrimSuffix(lc, "*")):
			return true
		case base == lc:
			return true
		}
	}
	return false
}

// underAllowed applies the Under and NotUnder path constraints.
func underAllowed(rel string, d Detect) bool {
	segments := strings.Split(rel, "/")
	has := func(seg string) bool {
		for _, s := range segments {
			if strings.EqualFold(s, seg) {
				return true
			}
		}
		return false
	}

	for _, nu := range d.NotUnder {
		if has(nu) {
			return false
		}
	}
	if len(d.Under) == 0 {
		return true
	}
	for _, u := range d.Under {
		if has(u) {
			return true
		}
	}
	return false
}

// contentMatches reports whether the file at absPath contains any signature.
// Unreadable files simply do not match; a read failure is not fatal to a walk.
func contentMatches(absPath string, signatures []string) bool {
	f, err := os.Open(absPath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, maxContentScanBytes)
	n, err := f.Read(buf)
	if n == 0 || (err != nil && n == 0) {
		return false
	}
	head := string(buf[:n])
	for _, sig := range signatures {
		if strings.Contains(head, sig) {
			return true
		}
	}
	return false
}
