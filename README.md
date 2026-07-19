# PDLC — Product Development Lifecycle

**PDLC** is the ProductBuildersHQ specification for the product development lifecycle: the stages, deliverables, canonical repository layout, quality gates, and the formal handoff from the **product person** (product manager / product owner) to the **builder person** (engineer / AI coding agent).

PDLC answers: *what does a complete product definition look like, where does every artifact live in the repository, how is it reviewed as a single website, and what exactly gets handed to engineering?*

## Position in the ProductBuildersHQ ecosystem

| Repository | Role |
|------------|------|
| **pdlc** (this repo) | Normative lifecycle specification: stages, deliverables, project layout, review site, Product Baseline handoff |
| [pdlc-workflows](https://github.com/ProductBuildersHQ/pdlc-workflows) | Executable agent-rules reference implementation of this specification |
| [visionspec](https://github.com/ProductBuildersHQ/visionspec) | Specification-artifact engine: spec types, templates, rubrics, LLM evaluation, reconciliation, export targets |
| [visionstudio](https://github.com/ProductBuildersHQ/visionstudio) | Desktop IDE for authoring, reviewing, and observing PDLC projects |
| [productbuildershq-frameworks](https://github.com/ProductBuildersHQ/productbuildershq-frameworks) | Machine-readable framework definitions (including PDLC and AWS AI-DLC) with PIDL process models |

PDLC is the *specification*; `pdlc-workflows` is the *reference implementation* — the same relationship AWS uses between the AI-DLC methodology and [awslabs/aidlc-workflows](https://github.com/awslabs/aidlc-workflows).

## The specification

| Document | Contents |
|----------|----------|
| [Lifecycle](docs/specification/lifecycle.md) | Stages, sequencing (including the guides-before-UI translation order), entry/exit criteria, roles |
| [Project Layout](docs/specification/project-layout.md) | The canonical project-repository filesystem contract |
| [Review Site](docs/specification/review-site.md) | How every artifact is assembled into one static website for human review on GitHub/GitLab Pages |
| [Requirements & Acceptance](docs/specification/requirements-acceptance.md) | Requirement-ID registry with stability rules; executable acceptance tests (browser-driven journeys) tied to requirement IDs |
| [Quality](docs/specification/quality.md) | Continuous evaluation: LLM-as-a-Judge, design-system, API-style, accessibility, and localization conformance |
| [Handoff](docs/specification/handoff.md) | The Product Baseline — the versioned contract between product and builder |
| [Adoption](docs/specification/adoption.md) | Agent-driven brownfield migration to the canonical layout + readiness evaluation |
| [`layout.yaml`](layout.yaml) | The layout contract in machine-readable form: canonical paths, detection heuristics, per-profile requirements |
| [`model/`](model/) | The lifecycle as a formal [PIDL](https://github.com/grokify/pidl) process model — typed steps, data flows over canonical paths, and gates |

## Core principles

1. **The repository is the product definition.** Every deliverable — vision, requirements, narratives, prototype, API specs, guides, translations, evaluations — has one canonical location in the project repository.
2. **Define when possible; narrate when needed.** Where a machine-readable form exists — OpenAPI, design-system specs, API style profiles, locale catalogs, personas IR, structured-evaluation results, baseline manifests — that form is the required artifact and narratives supplement it. Prose carries intent, rationale, and domain description where structure doesn't fit.
3. **Native workspaces stay native.** VisionSpec directories, `aidlc-docs/`, and other tool-owned trees are never reorganized. The review site *projects* them; it never moves or edits them.
4. **One website for reviewers.** A human reviewer walks from vision → requirements → six-pager → press release → FAQ → prototype → user/admin guides without knowing the filesystem.
5. **Prototypes are validating evidence, not requirements.** ER diagrams, Ent schemas, and demo code prove the definition is coherent; they carry `advisory` authority, not `normative`.
6. **The Product Baseline is the handoff.** Engineering consumes an approved, git-revision-pinned baseline — not a moving target.
7. **Translate guides before UI.** Human-facing guides carry the context translators need; guide translation precedes UI-string translation.
