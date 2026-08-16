# PDLC Lifecycle

PDLC is a **six-stage** lifecycle, machine-readably cataloged in [`productbuildershq-frameworks`](https://github.com/ProductBuildersHQ/productbuildershq-frameworks/tree/main/frameworks/pdlc) and re-exported from this module via [`Stages()`](../../stages.go). Each of AWS AI-DLC's three phases (Inception, Construction, Operation) splits into two parallel lenses — **product** and **builder** — so every stage has exactly one accountable role, while the two stages within a phase proceed together rather than gating each other:

```text
┌──────────────────────┬──────────────────────┬──────────────────────────────┐
│      INCEPTION        │     CONSTRUCTION      │          OPERATION           │
├───────────┬───────────┼───────────┬───────────┼───────────────┬──────────────┤
│ Product   │ Builder   │Implementa-│Deployment │  Builder      │  Product     │
│Definition │Definition │tion       │           │ Operations    │ Operations   │
│ (product) │ (builder) │(builder)  │(builder)  │  (builder)    │  (product)   │
└───────────┴───────────┴───────────┴───────────┴───────────────┴──────────────┘
                                                    └─── run in parallel ───┘
```

```text
                        Product Baseline
                              │
                              ▼
Product Definition ──────► Builder Definition ──► Implementation ──► Deployment
       ▲                                                                  │
       │                                              ┌───────────────────┤
       │                                              ▼                   ▼
       └──────────────────────────────────  Product Operations   Builder Operations
              (feedback loop closes the lifecycle, not a terminal stage)
```

This specification governs the full six-stage model in outline, and the **Product Definition** stage in normative depth — its seven detailed sub-stages (Discovery through Baseline & Handoff) are unchanged from prior versions of this document. The four builder-side stages (Builder Definition, Implementation, Deployment, Builder Operations) name their deliverables and gates but delegate internal process to established engineering methodologies (AWS AI-DLC, GitHub Spec Kit, OpenSpec, or team conventions) — PDLC specifies what crosses each stage boundary and how traceability is preserved, not how the builder does the work inside a stage.

## Roles

| Role | Responsibility |
|------|----------------|
| **Product person** (product manager, product owner) | Owns Product Definition and Product Operations; approves the Product Baseline and reviews adoption/drift signal |
| **Builder person** (engineer, AI coding agent) | Owns Builder Definition, Implementation, Deployment, and Builder Operations; consumes the approved Product Baseline; provides implementation feedback |
| **Reviewer** (design, QA, docs, localization, security, executive) | Reviews artifacts through the review site; participates in stage gates |
| **AI agent** | Drafts, evaluates (LLM-as-a-Judge), translates, and audits under human approval gates |

## Stages

Each stage lists its deliverables, the tool that produces them, and its exit gate. Quality evaluation (see [quality.md](quality.md)) and review-site assembly (see [review-site.md](review-site.md)) run continuously across all stages, not as a final stage. The machine-readable form of every stage — including deliverables, gates, the dependency graph, and the AI-DLC crosswalk — lives in the [`productbuildershq-frameworks` PDLC entry](https://github.com/ProductBuildersHQ/productbuildershq-frameworks/tree/main/frameworks/pdlc); this document is its normative narrative, not a duplicate source of truth.

### Stage 1: Product Definition (`product-definition`)

*(product person; AI-DLC: Inception)*

Product Definition has seven detailed sub-stages, each below.

#### 1. Discovery

*Why build this, and is it worth building?*

Discovery splits three ways by structure and sensitivity — the mirror image of how the prototype sits downstream of the specs:

| Deliverable | Location | Authority |
|-------------|----------|-----------|
| Opportunity spec, vision | `docs/specs/source/` | normative — VisionSpec |
| Competitive matrix, SWOT, TAM/SAM/SOM | `docs/specs/market/` | informative — VisionSpec discovery spec types |
| Personas | `docs/personas/` (+ `personas.json`) | normative — synthesized customer spec |
| Research synthesis notes | `docs/discovery/` | informative — narrative, PII-free |
| Evidence references | `docs/discovery/sources.yaml` | evidence — points to external raw data |
| Raw customer/market evidence (interviews, surveys, analytics) | **outside the repo** — excluded, or a separate RBAC/ABAC store | evidence — never in VisionSpec, never published |

The synthesized specs are the *conclusions*; the raw evidence is the *source material* they are grounded in. VisionSpec consumes discovery evidence to ground its synthesis the same way the builder later consumes the baseline — evidence in, specs out. Raw PII evidence never enters the reviewable repository; it is referenced from `sources.yaml`, which lets requirements and personas cite their evidence without exposing it (see [project layout](project-layout.md#where-spec-types-are-defined-vs-where-instances-live)).

**Exit gate:** opportunity approved by product person; where analytical specs are in scope, their VisionSpec rubrics pass.

#### 2. Definition

*What are we building, for whom, and how do we describe it?*

| Deliverable | Location | Produced with |
|-------------|----------|---------------|
| MRD, PRD, UXD | `docs/specs/source/` | VisionSpec |
| Requirement registry: immutable IDs + structured acceptance criteria | `docs/specs/requirements.yaml` | VisionSpec `pkg/requirements` (see [requirements-acceptance.md](requirements-acceptance.md)) |
| Press release, FAQ, one-pager, six-pager | `docs/specs/gtm/` | VisionSpec (synthesized, human-approved) |
| Personas | `docs/personas/` + `personas.json` IR | PDLC template (Go library + JSON IR planned) |
| Descriptive requirements: concepts, entities, relationships, process workflows | `docs/concepts/`, `docs/entities/`, `docs/processes/` | Hand-authored MkDocs content |

Descriptive requirements capture domain understanding beyond user stories — the concepts, entity relationships, and end-to-end processes the product must honor. They are normative product definition, expressed as readable pages rather than requirement lists.

**Exit gate:** required specs present, VisionSpec rubric evaluations passing, human approval recorded.

#### 3. Experience & Prototype

*What does it feel like, and does the definition hold together end-to-end?*

| Deliverable | Location | Authority |
|-------------|----------|-----------|
| Runnable static prototype (any static-site technology, e.g. React + TypeScript) | `prototype/` | normative for experience behavior |
| Prototype built for i18n from day one, source locale externalized | `prototype/` + `locales/ui/` | normative |
| ER diagrams, Ent schema Go structs, seed/demo data | `prototype/schema/` | **advisory** — validating evidence only |
| Acceptance journeys (executable ACs, role-based locators, per-locale) | `acceptance/journeys/` | **normative** — transfer to the builder |

The prototype is an executable specification of the experience. The database schema and Ent structs exist to prove the entities and workflows are implementable and internally consistent — they are **not** formal requirements and must not constrain the builder's production design.

The prototype must externalize every user-facing string into the locale JSON IR from the start; only the source locale (e.g., `en-US`) is populated at this stage.

**Exit gate:** prototype covers primary journeys and states; every normative functional requirement has an acceptance journey passing against the prototype; design-system conformance (`dss validate`) passing at agreed thresholds. Accessibility audit (`agent-a11y`) only when enabled — the prototype is advisory and the builder may rebuild it, so prototype a11y work may not transfer; a11y *requirements* remain normative and the production implementation is audited in the Builder Lifecycle.

#### 4. API

*What is the product's programmable contract?*

| Deliverable | Location | Authority | Required? |
|-------------|----------|-----------|-----------|
| OpenAPI specification | `docs/api/openapi.yaml` | advisory draft until baseline; the builder may refine | **required** |
| Scalar-rendered API reference | review site (`/api/`) | generated | **required** |
| API style profile (in-repo or named reference) + structured-eval conformance results | `pdlc.yaml` → `quality/api-style/` | — | recommended; gate-checked when configured |
| API requirements narrative (workflow intent, versioning policy) | `docs/specs/source/` or `docs/specs/technical/` | normative | as needed |

The defined artifact (OpenAPI) is primary; narrative states only what the contract cannot.

**Exit gate:** OpenAPI present and valid; when a style profile is configured, `api-style` lint + LLM-judge conformance at the project's target level (e.g., silver).

#### 5. Documentation

*How do users and administrators succeed?*

| Deliverable | Location |
|-------------|----------|
| User guide (source locale) | `docs/guides/user/` |
| Admin guide (source locale) | `docs/guides/admin/` |

Guides are written against the prototype — they describe the experience reviewers can click through, which keeps guides and prototype mutually validating.

**Exit gate:** guides cover primary journeys and admin workflows; reviewed by product person.

#### 6. Localization

*Does it work globally?*

Ordering within this stage is normative and deliberate:

```text
1. Translate user/admin guides        (guides carry the context translators need)
2. Translate UI string catalogs       (informed by guide terminology)
3. Judge l10n completeness            (coverage report on the review site)
```

| Deliverable | Location |
|-------------|----------|
| Locale manifest (locale codes, source locale, targets) | `locales/manifest.yaml` |
| Glossary / terminology registry | `locales/glossary/` |
| Translated guides per locale | `locales/guides/<locale>/` |
| UI string catalogs per locale (JSON IR) | `locales/ui/<locale>.json` |
| L10n coverage report | `quality/l10n/` |

I18N *requirements* (supported scripts, formatting, RTL, fallback) are defined during Definition and are cross-cutting: the prototype, guides, and API must satisfy them from their own stages onward. This stage produces the localization *assets*.

**Exit gate:** required locales at target coverage; glossary consistency between guides and UI catalogs; l10n judge passing.

#### 7. Baseline & Handoff

*Freeze what was approved; hand it to the builder.*

| Deliverable | Location |
|-------------|----------|
| Reconciled execution spec with traceability matrix | `docs/specs/spec.md` |
| Product Baseline manifest | `pdlc-docs/baselines/<id>/baseline.yaml` |
| Export packages for the builder's methodology | via VisionSpec export targets |

See [handoff.md](handoff.md).

**Exit gate:** readiness report (`quality/readiness.json`) green for the declared profile, with no unresolved error-severity consistency findings; baseline approved and revision-pinned; export generated for the selected builder workflow (AI-DLC, Spec Kit, OpenSpec, GitHub/Jira).

### Stage 2: Builder Definition (`builder-definition`)

*(builder person; AI-DLC: Inception)*

Consumes the approved Product Baseline and produces the authoritative technical design. The deliverable pattern mirrors Product Definition's normative-spec-plus-advisory-evidence shape: the **API Contract** (finalized OpenAPI, refined from Product Definition's advisory draft) is normative, and a **Reference SDK Client** auto-generated from that contract is advisory — a reference implementation used for acceptance testing, proving the contract is implementable without itself constraining the builder's production design, the same way the Product Definition prototype is advisory evidence for the PRD/UXD.

| Deliverable | Authority |
|-------------|-----------|
| Technical requirements (TRD/IRD): architecture, data models, system interfaces | normative |
| API Contract: finalized OpenAPI, refined from the Product Definition draft | normative |
| Reference SDK Client: generated from the API Contract | **advisory** — reference implementation and acceptance-testing tool only |

**Exit gate:** Technical Design Review — design is feasible, API Contract finalized, risks identified and mitigation planned.

### Stage 3: Implementation (`implementation`)

*(builder person; AI-DLC: Construction)*

Code is written against the Builder Definition technical contract. Overlaid by Application Security Posture Management (ASPM) domains covering git posture, code security, secret/PII scanning, open-source security, and SBOM generation — the security overlay is defined and owned by consuming security-analysis tooling (e.g. [Threat Model Spec](https://github.com/grokify/threat-model-spec)), not by this specification.

**Exit gate:** Code Review / CI Gate — automated checks (build, lint, test, SAST/SCA) plus human code review.

### Stage 4: Deployment (`deployment`)

*(builder person; AI-DLC: Construction)*

Built artifacts are released to target environments. Overlaid by ASPM domains covering infrastructure-as-code scanning, CI/CD posture, container security, and artifact security.

**Exit gate:** Release Gate — deployment validation, rollback readiness, release approval.

### Stage 5: Builder Operations (`builder-operations`)

*(builder person; AI-DLC: Operation — runs in parallel with Product Operations)*

Infrastructure, security, and reliability operations for the deployed system: monitoring, incident response, cloud security posture. Overlaid by the ASPM cloud-context domain, alongside dynamic testing (DAST, penetration testing, red teaming) which sits beside ASPM rather than within it.

### Stage 6: Product Operations (`product-operations`)

*(product person; AI-DLC: Operation — runs in parallel with Builder Operations)*

Adoption, usage, and feedback operations for the shipped product: activation, retention, feature usage, PMF signal, and customer feedback synthesis. This is where post-ship reconciliation happens — VisionSpec's `current-truth.md` and drift detection reconcile the approved definition with shipped reality.

**Exit gate:** Baseline Revision Trigger — significant drift between the approved definition and observed reality triggers a new Product Definition baseline revision, closing the lifecycle back to Stage 1 rather than terminating it.

## Sequencing summary

Within Product Definition, the critical-path ordering including the translation sequencing rule:

```text
Discovery → Definition → Prototype → API ─┐
                                          ├─→ Guides → Translate guides → Translate UI → Baseline
              (API may proceed in parallel with guides)
```

Across the six stages: Product Definition → Builder Definition → Implementation → Deployment → {Builder Operations ∥ Product Operations} → back to Product Definition. Stages may overlap where their inputs allow; gates may not be skipped. Which domains a project carries is declared by its **profile** in `pdlc.yaml` (see [project-layout.md](project-layout.md)); gates check the readiness report against the declared profile, so excluded domains never block and included domains always must pass.

## Feedback loops

The lifecycle is a cycle, not a line: Product Operations closes the loop back into the next Product Definition baseline revision, and Builder Operations feeds implementation/deployment learnings (cost, feasibility, architectural constraints, incident findings) back as input to the next Builder Definition cycle. Post-ship, VisionSpec's `current-truth.md` and drift detection reconcile the definition with shipped reality — this reconciliation is Product Operations' primary activity, not a separate mechanism.

## Machine-Readable Catalog

The stage IDs, deliverables, gates, dependency graph, and AI-DLC crosswalk are defined once in [`productbuildershq-frameworks`](https://github.com/ProductBuildersHQ/productbuildershq-frameworks/tree/main/frameworks/pdlc) and re-exported from this module:

```go
import "github.com/ProductBuildersHQ/pdlc"

stages := pdlc.MustStages() // six stages, in order

tech, _ := pdlc.StageByID(pdlc.StageBuilderDefinition)
```

Downstream consumers import the stage IDs rather than re-declare them — [`specification-workflow-spec`](https://github.com/ProductBuildersHQ/specification-workflow-spec) tags each spec type with its PDLC stage, and [Threat Model Spec](https://github.com/grokify/threat-model-spec) maps its ASPM security-posture domains onto the three builder-side stages that carry them (Implementation, Deployment, Builder Operations).
