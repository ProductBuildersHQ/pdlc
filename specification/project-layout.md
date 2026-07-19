# Canonical Project Layout

This is the PDLC filesystem contract for a product project repository. Like AWS AI-DLC's `aidlc-docs/` convention, these locations are a **public contract**: tools (VisionSpec CLI, VisionStudio, pdlc-workflows rules, the site generator) resolve artifacts by these paths, so directory names must not be renamed once adopted.

## Design rule: serve directly, project only when forced

The review site is a MkDocs site with `docs/` as its `docs_dir`, published to GitHub/GitLab Pages. **Everything that can live under `docs/` lives under `docs/`** so MkDocs serves it natively with zero copying. Build-time projection is reserved for content whose location is a contract we don't own — today that is `aidlc-docs/` (AWS AI-DLC's frozen convention) — and for computed pages (dashboards, coverage reports).

Since ProductBuildersHQ owns VisionSpec, VisionSpec's canonical spec output moves under `docs/specs/` (see "Required VisionSpec change" below). Specs are human-reviewed markdown; they belong in the served tree.

## The layout

```text
project/
├── pdlc.yaml                    # PDLC project manifest
├── visionspec.yaml              # VisionSpec manifest
├── mkdocs.yml                   # review-site configuration
│
├── docs/                        # MkDocs docs_dir — served directly on Pages
│   ├── index.md
│   │
│   ├── specs/                   # VisionSpec canonical output (relocated under docs/)
│   │   ├── requirements.yaml    #   requirement-ID registry (see requirements-acceptance.md)
│   │   ├── source/              #   mrd.md, prd.md, uxd.md, opportunity-spec.md
│   │   ├── gtm/                 #   press.md, faq.md, narrative-1p.md, narrative-6p.md
│   │   ├── technical/           #   trd.md, tpd.md, ird.md
│   │   ├── spec.md              #   reconciled execution spec
│   │   └── current-truth.md     #   post-ship reality
│   │
│   ├── concepts/                # descriptive requirements: domain concepts
│   ├── entities/                # entities and relationships
│   ├── processes/               # end-to-end process workflows (beyond user stories)
│   ├── personas/                # persona pages (+ personas.json IR)
│   │
│   ├── guides/
│   │   ├── user/                # user guide, source locale
│   │   └── admin/               # admin guide, source locale
│   │
│   ├── api/
│   │   ├── openapi.yaml         # the API contract (canonical; served + linted from here)
│   │   └── index.md             # Scalar-rendered API reference page
│   │
│   ├── prototype/
│   │   ├── index.md             # scenarios, journeys, how-to-run
│   │   ├── data-model.md        # ER diagrams (advisory validating evidence)
│   │   └── app/                 # BUILT React app (output of prototype build)
│   │
│   └── _generated/              # build-time projections ONLY — never hand-edited
│       ├── aidlc/               #   projected from aidlc-docs/ (native tree untouched)
│       ├── quality/             #   dashboards rendered from quality/ + eval/ results
│       ├── locales/             #   coverage dashboard computed from locales/
│       └── handoffs/            #   generated builder input packages
│
├── prototype/                   # prototype SOURCE — any static-site technology
│   ├── src/  public/  tests/    #   (e.g., React + TypeScript; the contract is the
│   │                            #    static build output, not the framework)
│   └── schema/                  # ER diagrams, Ent schema Go structs, seed data — advisory evidence
│
├── design-system/               # DSS design-system spec (JSON), when defined in-repo;
│                                #   alternatively pdlc.yaml references an external one
│
├── acceptance/                  # executable acceptance tests (NORMATIVE; transfer to builder)
│   ├── journeys/*.yaml          #   browser journeys with role-based locators, per-locale
│   └── config.yaml              #   target URLs, locale set, capture options
│
├── locales/
│   ├── manifest.yaml            # source locale, target locales, coverage targets
│   ├── glossary/                # terminology registry + prohibited translations
│   ├── ui/                      # UI string catalogs: <locale>.json (JSON IR)
│   └── guides/<locale>/{user,admin}/   # translated guides per locale
│
├── quality/                     # machine-written conformance evidence
│   ├── design-system/  a11y/  api-style/  l10n/  consistency/
│   ├── acceptance/              # journey run results + step screenshots per locale
│   └── readiness.json           # top-level roll-up report (structured-evaluation)
│
├── eval/                        # VisionSpec LLM-as-a-Judge results (<spec>.eval.json)
│
├── aidlc-docs/                  # AWS AI-DLC native workspace — NEVER reorganized
├── specs/  .specify/            # GitHub Spec Kit native workspace, when used
│
├── pdlc-docs/
│   ├── pdlc-state.md            # stage-level progress
│   ├── audit.md                 # append-only decision/approval log
│   └── baselines/PB-<id>/baseline.yaml
│
├── .visionspec/                 # VisionSpec WORKING STATE — never published (see below)
│
├── cmd/  internal/  pkg/  ...   # production source — team-defined, unmanaged
└── site/                        # rendered MkDocs output (CI artifact; gitignored)
```

## Defined artifacts vs narratives

PDLC's rule is **define when possible; narrate when needed**: where a machine-readable form exists, it is the required artifact; prose supplements it. Per domain:

| Domain | Required defined artifacts | Recommended defined artifacts | Narrative role |
|--------|---------------------------|-------------------------------|----------------|
| **API** | `docs/api/openapi.yaml` + Scalar reference page | api-style profile (in-repo file or named reference in `pdlc.yaml`) + structured-eval results in `quality/api-style/` | API requirements prose states intent the contract can't (workflows, versioning policy rationale) |
| **Design** | — | DSS design-system spec (`design-system/` or external reference) + structured-eval results in `quality/design-system/` | Design principles and content-design guidance |
| **Prototype** | Static build at `docs/prototype/app/`; strings externalized to `locales/ui/` | ER/Ent schema evidence in `prototype/schema/` | Scenario and journey pages in `docs/prototype/` |
| **Localization** | `locales/manifest.yaml` + locale JSON IR catalogs | glossary YAML; l10n coverage results in `quality/l10n/` | I18N requirements prose (scripts, RTL, fallback policy) |
| **Personas** | `personas.json` IR | — | Persona pages rendered from the IR |
| **Accessibility** | — | agent-a11y results + baselines in `quality/a11y/` (optional: prototype audit may not transfer if the builder rebuilds the UI) | Accessibility requirements prose (normative for the product) |
| **Requirements** | `docs/specs/requirements.yaml` registry (immutable IDs, tombstoned lifecycle, structured acceptance criteria) | — | Requirement statements are authored inline in spec prose with `**FR-n**` markers |
| **Acceptance** | `acceptance/journeys/*.yaml` (executable, role-based locators, `verifies:` links to AC IDs) + results in `quality/acceptance/` | — | — |
| **Specs** | VisionSpec front matter, registry-backed requirement IDs, rubric evals in `eval/` | — | The spec bodies themselves (MRD/PRD/UXD, six-pager, press, FAQ) are the narrative payload |
| **Baseline** | `baseline.yaml` manifest | — | — |

"Required" means the stage gate checks for the artifact; "recommended" means the gate uses it when present (e.g., API conformance is judged only when a style profile is configured, but the OpenAPI file itself is always required for the API stage).

## Profiles

Which domains a project carries is declared by its **profile** in `pdlc.yaml`. The `full` profile is the reference practice — React-class prototype with full I18N/L10N, Ent-validated data model, user/admin guides, OpenAPI + style conformance, all judges — proven on real projects and carried in full every iteration. Reduced profiles exist to bring this practice to a broader community of product people, not to dilute it: `standard` and `minimal` select fewer domains, and the readiness report evaluates presence/quality **only against the declared profile** (excluded domains report as `excluded`, never as failures). A community builder can start `minimal` (specs + OpenAPI + baseline) and grow toward `full` without restructuring the repository, because the layout contract is identical at every profile.

## `.visionspec/` is tool state, not specs

`.visionspec/` holds VisionSpec's working files, none of which are project specifications:

| File | Purpose | Commit? |
|------|---------|---------|
| `context-cache.json` | cached codebase-context gathering | no — gitignore |
| `execution-status.json` | downstream execution tracking | optional |
| `metrics.json` | metrics history | optional |
| `maturity/*.json` | maturity-model state (VisionStudio) | yes |
| mirrored AI-DLC docs | `SyncEngine` workspace for `.visionspec/` ↔ `aidlc-docs/` sync | yes (sync state) |

The review site must never publish `.visionspec/`. Canonical, reviewable specs live in `docs/specs/`.

## Prototype placement

The prototype may be built with **any static-site technology** — React + TypeScript is typical, but the PDLC contract is only that the prototype (a) builds to static assets publishable at `docs/prototype/app/` under a Pages subpath (relative base path), and (b) externalizes user-facing strings into the locale JSON IR.

Prototype **source** stays outside `docs_dir` (at `prototype/`); the **built** app is emitted to `docs/prototype/app/`. This keeps `node_modules/` and source files out of MkDocs's walk while reviewers get the app on the same host.

*Accepted variant:* if a project keeps the app inside `docs/prototype/`, it must use MkDocs `exclude_docs` to omit `src/`, `node_modules/`, and config files so only the built assets are published. The spec default is source-outside, build-inside.

UI string catalogs live canonically in `locales/ui/<locale>.json`; the prototype build consumes them from there (import or copy step) so translation tooling and the l10n judge work against one set of files.

## Required VisionSpec change

VisionSpec currently routes specs to repo-root directories (`source/`, `gtm/`, `technical/` via `SpecType.Dir()`, scaffolded by VisionStudio). To serve specs natively, VisionSpec needs a configurable specs root in `visionspec.yaml`:

```yaml
# visionspec.yaml
specsRoot: docs/specs      # default "." preserves existing projects
```

All path resolution (`SpecType.Dir()`, template scaffolding, reconcile, VisionStudio's spec map) resolves relative to `specsRoot`. PDLC projects set `docs/specs`; existing projects are unaffected. `eval/` results stay at root — they are JSON evidence rendered into dashboard pages, not served pages.

## Locale JSON IR (initial shape)

One file per locale, keyed identically across locales; the source locale is the completeness reference. Designed so the coverage judge can diff key sets and flag untranslated or stale entries:

```json
{
  "$schema": "https://productbuildershq.org/schema/l10n-catalog.schema.json",
  "locale": "de-DE",
  "sourceLocale": "en-US",
  "domain": "ui",
  "entries": {
    "settings.profile.title": {
      "value": "Profileinstellungen",
      "source": "Profile Settings",
      "sourceHash": "9f2c1a",
      "status": "translated",
      "context": "Heading on the profile settings page",
      "maxLength": 40
    }
  }
}
```

`status` ∈ `missing | machine | translated | reviewed`; `sourceHash` marks entries stale when the source string changes. The same IR (with `domain: "guides"`) indexes guide-translation status. A Go library implementing this IR is planned; per the Go-first schema policy, the JSON Schema is generated from Go types, not hand-written.

## Personas IR (initial shape)

Persona pages in `docs/personas/` are rendered from a structured IR so personas can be referenced by ID from requirements, journeys, and prototype scenarios:

```json
{
  "id": "persona-admin-ops",
  "name": "Operations Administrator",
  "role": "admin",
  "goals": ["..."],
  "frustrations": ["..."],
  "journeys": ["JRN-ADMIN-001"],
  "locales": ["en-US", "de-DE"]
}
```

A Go library + generated schema is planned.

## `pdlc.yaml` manifest (initial shape)

```yaml
apiVersion: pdlc.productbuildershq.org/v1
kind: ProductProject
metadata:
  id: my-product
  title: My Product
spec:
  profile: full                # full | standard | minimal | custom
  artifacts:                   # per-artifact overrides of the profile
    prototype: required
    guides: required
    localization: required
    api: required
    designSystem: recommended
    a11y: optional             # prototype audit; enable when the prototype will be
                               # promoted toward production rather than rebuilt
  locales:
    source: en-US
    targets: [de-DE, ja-JP]
  quality:
    designSystem:
      spec: ./design-system          # in-repo DSS spec, or an external reference
      threshold: pass
    apiStyle:
      profile: omniagent-rest        # named profile, or spec: ./api-style/spec.json
      level: silver
    a11y: { wcagVersion: "2.2", level: AA }
    l10n: { minimumCoverage: 0.95 }
  builder:
    methodology: aws-aidlc     # aws-aidlc | github-speckit | openspec | custom
```

## Rules

1. `docs/_generated/` and `site/` are machine-written; hand edits are forbidden and overwritten.
2. `quality/` and `eval/` are machine-written evidence; re-runs replace reports, baselines are retained for regression gating.
3. `pdlc-docs/audit.md` is append-only.
4. `aidlc-docs/`, `specs/`, `.specify/` are governed by their own tools; PDLC reads and projects them, never writes into them.
5. `.visionspec/` is never published; its caches are gitignored.
6. Production-code layout is team-defined; PDLC records traceability references into it but never mandates it.
