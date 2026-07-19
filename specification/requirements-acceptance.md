# Requirement IDs & Acceptance Tests

Requirements are the unit of traceability across the entire PDLC — from spec prose through prototype, guides, translations, baseline, and the builder's implementation. This specification makes requirement IDs a **defined artifact** (a registry, not regex-recognized strings) and makes acceptance criteria **executable** (browser-driven journeys), so every row of the traceability matrix is checkable instead of asserted.

## The requirement registry

`docs/specs/requirements.yaml` is the canonical index of requirement IDs. Spec prose (PRD, UXD, ...) remains the authoring surface, with inline markers (`**FR-0012**: ...`); the registry makes the IDs stable.

```yaml
# docs/specs/requirements.yaml
version: 1
nextId: { FR: 13, NFR: 4, US: 22 }
requirements:
  - id: FR-0012
    kind: functional              # functional | non-functional | user-story
    title: Multi-organization membership
    statement: A user may belong to multiple organizations with a distinct role in each.
    spec: docs/specs/source/prd.md          # owning spec
    status: active                          # active | superseded | withdrawn
    acceptance:
      - id: FR-0012.AC-1
        given: a user belonging to two organizations
        when: they switch the active organization
        then: role-appropriate navigation for that organization is shown
        test: acceptance/journeys/org-switching.yaml#switch-org
```

### Stability rules (lint-enforced, deterministic)

1. IDs are allocated by tooling (`visionspec req new FR "..."`), monotonic and zero-padded; the number carries **no meaning** — semantics live in metadata, so IDs never need renumbering.
2. IDs are immutable once allocated. Dropped requirements are tombstoned (`status: withdrawn`, optionally `supersededBy`), never deleted or reused — old baselines stay interpretable forever.
3. Bidirectional integrity: every inline ID in spec prose exists in the registry; every `active` entry appears in its owning spec; every journey `verifies` reference resolves; every guide `covers` reference resolves.
4. Coverage (profile-dependent): every `active` functional requirement has at least one acceptance criterion; every acceptance criterion references a test.

The registry types, allocation, and lint live in VisionSpec (`pkg/requirements`), with the JSON Schema generated from the Go types.

## Acceptance tests: executable acceptance criteria

Each acceptance criterion maps to a step in a **journey** — a declarative browser scenario executed by a [w3pilot](https://github.com/plexusone/w3pilot)-driven runner against the prototype (and later, the production implementation).

```yaml
# acceptance/journeys/org-switching.yaml
id: org-switching
verifies: [FR-0012.AC-1, US-0007]
locales: [en-US, de-DE, ja-JP]
steps:
  - navigate: /dashboard
  - click: { role: button, name: "Organization" }
  - click: { role: menuitem, name: "Acme GmbH" }
  - assert: { visible: { role: heading, name: "Acme GmbH" } }
  - screenshot: org-switched
```

### Design rules

1. **Role/name locators, never CSS selectors.** Locators bind to accessible semantics, so journeys survive a UI rebuild — and they structurally pressure the prototype toward accessible markup. The locator strategy and accessibility goals reinforce each other.
2. **The acceptance suite is `normative` and transfers across the handoff.** Unlike prototype a11y work (which may be discarded when the builder rebuilds), journeys are the acceptance criteria in executable form — the most "defined" a requirement can get. The baseline pins `acceptance/` as normative; the builder's production implementation must pass the **same** journeys pointed at a different URL. Prototype-passing is advisory evidence; the suite itself is requirement.
3. **Per-locale execution.** Journeys run under every target locale, verifying translations in context (text expansion, untranslated strings in flows) — checks no catalog diff can perform.
4. **Screenshots are review evidence.** The runner captures a screenshot per declared step per locale. The review site renders one page per requirement: statement → acceptance criteria → journey steps → latest screenshots → pass/fail. Requirement review becomes watching evidence, not imagining prose.

### One journey format, three consumers

agent-a11y already parses, compiles, and executes journeys over w3pilot. That engine is extracted into a shared package so a single journey file drives:

| Consumer | Use |
|----------|-----|
| Acceptance runner | Functional assertions → `quality/acceptance/` (structured-evaluation) |
| agent-a11y | Axe/WCAG audits along the journey paths (its journeys mode) |
| Evidence capturer | Screenshots per step per locale for review-site requirement pages |

## Acceptance tests vs documentation

Two verification layers connect the suite to the guides:

1. **Deterministic (lint):** guide pages declare `covers: [FR-0012]` in front matter. Lint fails guides referencing dead IDs and flags normative requirements no guide documents (at profiles requiring guides).
2. **LLM (consistency judge):** compare guide instructions against journey steps for the same requirement — do the guides tell users to do what the tests actually do? Do the docs promise outcomes no test verifies? Findings land in `quality/consistency/`; error severity blocks baselines.

The triangle is self-reinforcing: guides are written against the prototype, journeys verify the prototype, and the judge verifies guides against journeys — all three describe one product.

## The traceability chain

```text
FR-0012 (registry, immutable)
  → acceptance criterion (structured, in registry)
    → journey (executable, role-based locators, per-locale)
      → run vs prototype   → screenshots + structured-eval   (definition evidence)
      → run vs production  → verification                    (builder phase)
        → guide page (covers: FR-0012) ← consistency-judged against the journey
          → readiness roll-up → Product Baseline
```

## Repository locations

| Artifact | Location |
|----------|----------|
| Requirement registry | `docs/specs/requirements.yaml` |
| Journeys | `acceptance/journeys/*.yaml` |
| Runner config | `acceptance/config.yaml` (target URLs, locale set, capture options) |
| Results | `quality/acceptance/` (structured-evaluation + screenshots) |
| Requirement evidence pages | `docs/_generated/requirements/` (site projection) |
