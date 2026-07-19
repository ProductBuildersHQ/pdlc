# PDLC Lifecycle

The PDLC consists of two lifecycles joined by a formal handoff:

```text
Product Definition Lifecycle          Builder Lifecycle
(product person)                      (builder person)

Discovery                             Engineering
Definition                            Testing
Experience & Prototype                Deployment
API                                   Operations
Documentation
Localization
        │
        ▼
Approved Product Baseline  ──────────►  consumed by Builder
```

This specification governs the **Product Definition Lifecycle**. The Builder Lifecycle is delegated to established engineering methodologies (AWS AI-DLC, GitHub Spec Kit, OpenSpec, or team conventions); PDLC specifies only what the builder receives and how traceability is preserved.

## Roles

| Role | Responsibility |
|------|----------------|
| **Product person** (product manager, product owner) | Owns all Product Definition stages and approves the Product Baseline |
| **Builder person** (engineer, AI coding agent) | Consumes the approved Product Baseline; provides implementation feedback |
| **Reviewer** (design, QA, docs, localization, security, executive) | Reviews artifacts through the review site; participates in stage gates |
| **AI agent** | Drafts, evaluates (LLM-as-a-Judge), translates, and audits under human approval gates |

## Stages

Each stage lists its deliverables, the tool that produces them, and its exit gate. Quality evaluation (see [quality.md](quality.md)) and review-site assembly (see [review-site.md](review-site.md)) run continuously across all stages, not as a final stage.

### 1. Discovery

*Why build this, and is it worth building?*

| Deliverable | Location | Produced with |
|-------------|----------|---------------|
| Opportunity spec, market analysis | `docs/specs/source/` | VisionSpec (discovery profiles) |
| Vision | `docs/specs/source/` | VisionSpec |

**Exit gate:** opportunity approved by product person.

### 2. Definition

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

### 3. Experience & Prototype

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

### 4. API

*What is the product's programmable contract?*

| Deliverable | Location | Authority | Required? |
|-------------|----------|-----------|-----------|
| OpenAPI specification | `docs/api/openapi.yaml` | advisory draft until baseline; the builder may refine | **required** |
| Scalar-rendered API reference | review site (`/api/`) | generated | **required** |
| API style profile (in-repo or named reference) + structured-eval conformance results | `pdlc.yaml` → `quality/api-style/` | — | recommended; gate-checked when configured |
| API requirements narrative (workflow intent, versioning policy) | `docs/specs/source/` or `docs/specs/technical/` | normative | as needed |

The defined artifact (OpenAPI) is primary; narrative states only what the contract cannot.

**Exit gate:** OpenAPI present and valid; when a style profile is configured, `api-style` lint + LLM-judge conformance at the project's target level (e.g., silver).

### 5. Documentation

*How do users and administrators succeed?*

| Deliverable | Location |
|-------------|----------|
| User guide (source locale) | `docs/guides/user/` |
| Admin guide (source locale) | `docs/guides/admin/` |

Guides are written against the prototype — they describe the experience reviewers can click through, which keeps guides and prototype mutually validating.

**Exit gate:** guides cover primary journeys and admin workflows; reviewed by product person.

### 6. Localization

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

### 7. Baseline & Handoff

*Freeze what was approved; hand it to the builder.*

| Deliverable | Location |
|-------------|----------|
| Reconciled execution spec with traceability matrix | `docs/specs/spec.md` |
| Product Baseline manifest | `pdlc-docs/baselines/<id>/baseline.yaml` |
| Export packages for the builder's methodology | via VisionSpec export targets |

See [handoff.md](handoff.md).

**Exit gate:** readiness report (`quality/readiness.json`) green for the declared profile, with no unresolved error-severity consistency findings; baseline approved and revision-pinned; export generated for the selected builder workflow (AI-DLC, Spec Kit, OpenSpec, GitHub/Jira).

## Sequencing summary

The critical-path ordering, including the translation sequencing rule:

```text
Discovery → Definition → Prototype → API ─┐
                                          ├─→ Guides → Translate guides → Translate UI → Baseline
              (API may proceed in parallel with guides)
```

Stages may overlap where their inputs allow; gates may not be skipped. Which domains a project carries is declared by its **profile** in `pdlc.yaml` (see [project-layout.md](project-layout.md)); gates check the readiness report against the declared profile, so excluded domains never block and included domains always must pass.

## Feedback loops

Implementation feedback from the Builder Lifecycle (cost, feasibility, architectural constraints) flows back as input to the next baseline revision. Post-ship, VisionSpec's `current-truth.md` and drift detection reconcile the definition with shipped reality.
