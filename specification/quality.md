# Quality: Continuous Evaluation

PDLC treats quality evaluation as a cross-cutting activity that runs throughout the lifecycle, with results surfaced on the review site and enforced at stage gates. Evaluation combines LLM-as-a-Judge (qualitative) with deterministic conformance tools. Several of these tools (design-system-spec, api-style-spec, agent-a11y) are already implemented and in production use; PDLC specifies how their results compose.

**Enforcement is a report, not a rule engine.** Every check emits a [structured-evaluation](https://github.com/plexusone/structured-evaluation) report, and reports compose hierarchically: per-tool leaf reports roll up into a single project **readiness report** that gates, baselines, and the site dashboard all consume.

## Evaluation matrix

| Subject | Tool | Method | Input | Result location |
|---------|------|--------|-------|-----------------|
| VisionSpec specs (MRD/PRD/UXD/GTM/technical) | `visionspec eval` | LLM-as-a-Judge against per-spec rubrics | spec markdown | `eval/<spec>.eval.json` |
| AWS AI-DLC deliverables | AI-DLC evaluation profile | LLM-as-a-Judge | `aidlc-docs/` artifacts | `aidlc-docs/` + normalized copy in `quality/` |
| Prototype source | `dss validate` / `dss eval` ([design-system-spec](https://github.com/plexusone/design-system-spec)) | static scan of `.tsx/.css` for token, variant, and anti-pattern violations; plus LLM rubric producing structured-eval output | `prototype/src/` + the project's DSS spec (`design-system/` in-repo, or an external reference in `pdlc.yaml`) | `quality/design-system/` |
| Running prototype (**optional**) | `agent-a11y audit` ([agent-a11y](https://github.com/plexusone/agent-a11y)) | real-browser axe-core + WCAG 2.2 audit (crawl + journeys), optional LLM judge | served prototype build URL | `quality/a11y/` |
| OpenAPI contract | `api-style lint` / `analyze` ([api-style-spec](https://github.com/plexusone/api-style-spec)) | deterministic vacuum lint + LLM judge at bronze/silver/gold | `docs/api/openapi.yaml` + style profile (named, e.g. `omniagent-rest`, or in-repo spec) | `quality/api-style/` |
| Localization completeness | **l10n judge (to be built)** | key-set diff + staleness detection + LLM terminology/glossary consistency check | `locales/ui/*.json`, `locales/guides/`, `locales/glossary/` | `quality/l10n/` |
| Acceptance coverage & results | **acceptance runner (to be built)** | w3pilot-driven journey execution per locale, with step screenshots; coverage = every normative FR has a passing journey | `acceptance/journeys/` vs served prototype | `quality/acceptance/` |
| Cross-artifact consistency | **consistency judge (to be built)** | LLM-as-a-Judge over artifact pairs (see below) | specs, OpenAPI, prototype, guides, GTM, locales, journeys | `quality/consistency/` |
| Project readiness | **readiness roll-up (to be built)** | presence + quality aggregation over all leaf reports, per profile | all of the above + profile from `pdlc.yaml` | `quality/readiness.json` |

### Why the prototype a11y audit is optional

The prototype carries `advisory` authority: the builder may **rebuild** the production UI rather than evolve the prototype, in which case a11y work invested in the prototype implementation does not transfer. Auditing the prototype is therefore optional (`artifacts.a11y` in `pdlc.yaml`) — worth enabling when the prototype is expected to be promoted toward production, or when a11y findings would change the UX *design* rather than just the implementation. Accessibility **requirements** remain normative regardless; the production implementation is audited in the Builder Lifecycle, where agent-a11y applies as-is.

## Report hierarchy

```text
quality/
├── design-system/   # leaf: DSS conformance (structured-eval)
├── api-style/       # leaf: API style conformance (structured-eval + SARIF evidence)
├── a11y/            # leaf: WCAG audit (normalized to structured-eval via adapter)
├── l10n/            # leaf: localization coverage + terminology (structured-eval)
├── consistency/     # leaf: cross-artifact consistency judge (structured-eval)
└── readiness.json   # ROLL-UP: the project readiness report (structured-eval)
```

The **readiness report** (`quality/readiness.json`) is the top-level structured-evaluation report answering, per artifact domain the project's profile declares:

1. **Presence** — does the artifact exist? (OpenAPI present, user guide present, target-locale translations present, prototype build present, ...)
2. **Quality** — does its leaf report pass? (API definition exists *and* passes its style level; translations exist *and* meet coverage; ...)
3. **Exclusions** — domains not in the project's profile are reported as `excluded`, never as failures.

The readiness report is what stage gates check, what the baseline embeds at approval, and what the review-site dashboard renders as its top panel. Reviewers see one answer to "is this product definition complete and good?", with drill-down into each leaf report.

## The consistency judge (new component)

Per-artifact judges verify each artifact against its own spec; the consistency judge verifies the artifacts against **each other** — the drift between them is what most damages builder trust. It is LLM-as-a-Judge over artifact pairs, emitting structured-evaluation into `quality/consistency/`:

| Check | Artifacts compared |
|-------|-------------------|
| Every PRD workflow/FR is reachable via the API contract | PRD ↔ `docs/api/openapi.yaml` |
| Every UXD journey has a prototype scenario | UXD ↔ prototype scenarios |
| Guides describe screens/flows the prototype actually has | guides ↔ prototype |
| GTM artifacts promise nothing absent from the PRD | press/FAQ/six-pager ↔ PRD |
| Entity descriptions, Ent schema, and OpenAPI schemas agree | `docs/entities/` ↔ `prototype/schema/` ↔ OpenAPI |
| Terminology is uniform across specs, guides, and UI strings | glossary ↔ guides ↔ `locales/ui/` |
| Guide instructions match what acceptance tests actually do; docs promise no outcome no test verifies | guides ↔ `acceptance/journeys/` |

Findings carry the artifact pair, the specific conflict, and a severity; unresolved `error`-severity conflicts block baseline approval (a baseline with known contradictions is a defect shipped to the builder).

## The l10n judge (new component)

No existing tool in the ecosystem checks localization coverage. PDLC specifies a new judge with two layers:

1. **Deterministic coverage** — for each target locale, diff entry keys and `sourceHash` values against the source-locale catalog: report `missing`, `machine`, `translated`, `reviewed`, and `stale` counts for UI catalogs, and page-level presence/staleness for translated guides. Pure Go, no LLM.
2. **LLM terminology judge** — sample translated entries and guide sections against the glossary (including prohibited translations) and judge terminology consistency between guides and UI strings. This encodes the guides-first sequencing benefit: the guide translations establish terminology that the UI judge then enforces.

Additional deterministic checks worth including from the start: ICU/placeholder integrity (`{name}` present in both source and translation), `maxLength` violations, and locale-file well-formedness.

## Result normalization

The tools emit divergent formats:

| Tool | Native result format |
|------|---------------------|
| visionspec, dss, api-style | [structured-evaluation](https://github.com/plexusone/structured-evaluation) rubric results (+ tool-specific reports, SARIF) |
| agent-a11y | multi-agent-spec `AgentResult` + VPAT/OpenACR/JSON reports |
| l10n judge | structured-evaluation (build it this way from day one) |

**Rule:** the review-site quality dashboard consumes **structured-evaluation** as its single IR. Tool-specific formats are preserved in `quality/` as evidence; adapters normalize them for display. One adapter is required now (agent-a11y `AgentResult` → structured-evaluation); future tools must either emit structured-evaluation natively or ship an adapter. Detailed native reports (VPAT, SARIF, HTML) are linked from dashboard entries, not re-rendered.

## Gate semantics

Each stage gate references quality thresholds from `pdlc.yaml`:

- Gates consume the **readiness report** (and through it, normalized leaf results), so gate logic is uniform across tools and profiles.
- Regression gating uses each tool's baseline mechanism where available (`agent-a11y validate --baseline`, `dss visual` baselines); the baseline files live under `quality/<tool>/baselines/`.
- A failing evaluation blocks the stage gate but never blocks authoring — evaluations run continuously; gates are checked at stage exit and at baseline approval.
- The Product Baseline manifest embeds a summary of all evaluation results at approval time, so the builder receives the quality evidence alongside the requirements (see [handoff.md](handoff.md)).
