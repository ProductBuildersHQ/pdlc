# The Review Site

Every PDLC project publishes **one static website** — built with MkDocs, hosted on GitHub Pages or GitLab Pages — that presents the complete product definition to human reviewers. A reviewer moves from vision to requirements to the six-pager, press release, and FAQ, into the clickable prototype, and through the user/admin guides without ever navigating the repository.

## Navigation contract

The site navigation reflects the reviewer's journey, not the filesystem:

```text
Overview
├── Product summary
├── Lifecycle status            (from pdlc-docs/pdlc-state.md)
└── Quality dashboard           (normalized evaluation results)

Vision & Opportunity
├── Vision
├── Opportunity / Market (MRD)
├── Six-Pager
├── Press Release
└── FAQ

Requirements
├── Product Requirements (PRD)
├── Experience Requirements (UXD)
├── Personas
├── Concepts
├── Entities & Relationships
├── Process Workflows
└── I18N Requirements

Prototype
├── Overview & scenarios
├── ▶ Launch prototype          (embedded static app at /prototype/app/)
├── Data model (advisory)       (ER diagrams, schema evidence)
└── Validation results          (a11y, design-system reports)

API
├── API Requirements
├── API Reference               (Scalar rendering of api/openapi.yaml)
└── Style conformance           (api-style results)

Guides
├── User Guide                  (per locale)
└── Admin Guide                 (per locale)

Localization
├── Locale coverage dashboard   (translations present / missing / stale)
├── Glossary
└── Per-locale status

Quality & Traceability
├── VisionSpec evaluations      (LLM-as-a-Judge results per spec)
├── AI-DLC evaluations
├── Accessibility (WCAG)
├── Design-system conformance
├── API style conformance
├── L10n completeness
└── Traceability matrix         (from spec.md reconciliation)

Baseline
├── Current baseline manifest
└── Baseline history
```

## Assembly model: native-first, project only what's outside `docs/`

Most of the site is **served natively**: specs (`docs/specs/`), descriptive requirements, personas, guides, the API contract, and the built prototype all live inside the MkDocs `docs_dir` per the [project layout](project-layout.md), so MkDocs publishes them with no copying.

Build-time projection into `docs/_generated/` is limited to:

| Projected content | Why it can't be served natively |
|-------------------|--------------------------------|
| `aidlc-docs/` → `docs/_generated/aidlc/` | AWS AI-DLC's directory is a frozen external contract; it cannot move under `docs/` |
| Quality dashboards ← `quality/` + `eval/` | Computed pages rendered from JSON evidence |
| Locale coverage ← `locales/` | Computed by diffing catalogs against the source locale |
| Handoff packages | Generated builder inputs |

```text
docs/ (native content)  ──────────────────────────────┐
aidlc-docs/  quality/  eval/  locales/  ─ projection ─┤
                                                      ▼
                                            docs/_generated/
                                                      │
                                                      ▼  mkdocs build
                                            site/  →  GitHub/GitLab Pages
```

Rules:

1. **Never hand-edit projections.** Every generated page carries front matter identifying its canonical source (`source_path`, `generated: true`) and a visible banner: *"Generated from `<path>` — edit the source, not this page."*
2. **The prototype is embedded, not linked away.** The prototype build emits its static app into `docs/prototype/app/` with a relative base path so it works under a Pages subpath; reviewers stay on one host. React source stays outside `docs_dir` (or is excluded via `exclude_docs`).
3. **OpenAPI renders via Scalar** as a self-contained page at `/api/`, reading `docs/api/openapi.yaml` in place.
4. **Evaluation results are normalized before rendering.** See [quality.md](quality.md) — the dashboard consumes one result IR, not five tool formats.
5. **Locale dashboard is computed, not authored.** Coverage pages are generated from the locale JSON IR by diffing key sets against the source locale — translated / machine / missing / stale per locale, for both UI catalogs and guides.
6. **Site nav is generated.** `pdlc site build` emits the MkDocs nav from the project's manifest and present artifacts; teams do not hand-maintain a giant `nav:` block. Missing optional artifacts drop out of the nav; missing required artifacts render as explicit "not yet authored" gaps so reviewers see incompleteness rather than silence.

The `aidlc-docs/` projection can be implemented either as a `pdlc site build` copy step before `mkdocs build`, or as a small MkDocs hook (`on_files`) that injects the pages at build time without any committed copy. Both satisfy the contract; the hook avoids stale committed projections.

## CI

```text
on push:
  1. build prototype            (npm ci && npm run build → docs/prototype/app/)
  2. run quality suite          (dss validate on prototype src, agent-a11y against
                                 served docs/prototype/app/, api-style on
                                 docs/api/openapi.yaml, l10n coverage on locales/)
  3. pdlc site build            (aidlc-docs + evidence → docs/_generated/)
  4. mkdocs build               (→ site/)
  5. publish                    (GitHub Pages / GitLab Pages)
```

Whether `docs/_generated/` and `docs/prototype/app/` are committed or CI-only is a project choice; when committed, CI must fail if they are stale.
