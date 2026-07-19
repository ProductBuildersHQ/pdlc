# Adoption: Agent-Driven Layout Migration & Evaluation

PDLC is adopted **brownfield-first**: an AI agent reads the layout contract, inventories an existing repository, moves artifacts to their canonical locations under human approval, and then evaluates the project — emitting the readiness report. The same procedure is idempotent: on an already-conformant repository it changes nothing and just re-emits the report.

## The machine-readable contract

Agents do not parse prose to learn the layout. The contract exists as data:

| File | Contents |
|------|----------|
| [`layout.yaml`](../layout.yaml) | Every artifact kind: canonical path, authority, per-profile requirement level, and detection heuristics (filenames, directory names, content signatures) |
| `pdlc.yaml` (in the project) | The project's declared profile and per-artifact overrides |

`layout.yaml` also declares the **immovable list** — tool-native directories (`aidlc-docs/`, `specs/`, `.specify/`, `.visionspec/`) that must never be reorganized regardless of where detection heuristics might otherwise place their contents.

The executable procedure agents follow lives in [pdlc-workflows](https://github.com/ProductBuildersHQ/pdlc-workflows) (`adoption/adopt-layout.md`, `evaluation/run-readiness.md`).

## The adoption procedure

```text
1. Inventory      walk the repo; classify files against layout.yaml detection
                  heuristics; skip immovable directories entirely
2. Plan           produce a move plan table: current path → canonical path,
                  classification confidence, and any ambiguities as questions
3. ⛔ GATE        present the plan; DO NOT move anything until the human approves
                  (ambiguous classifications are resolved by the human here)
4. Execute        `git mv` only (history preserved); update intra-repo links;
                  create pdlc.yaml (profile confirmed at the gate) and
                  pdlc-docs/ state files if absent
5. Verify         re-run inventory; confirm zero remaining moves
6. Evaluate       run the readiness evaluation (below); emit quality/readiness.json
7. Audit          every decision, approval, and move appended to pdlc-docs/audit.md
```

Rules:

- **Moves are `git mv`, never copy-delete** — file history is part of the product definition's provenance.
- **Ambiguity goes to the human, not to a guess.** A file matching multiple artifact kinds (or none confidently) is listed as a question at the gate.
- **The agent never edits content during adoption** — it relocates and re-links only. Content gaps are the evaluation's job to report, not adoption's job to fill.
- **Immovable directories are skipped at inventory time**, not filtered at execution time, so they can never appear in a plan.

## The readiness evaluation

The evaluation produces `quality/readiness.json` as a [structured-evaluation](https://github.com/plexusone/structured-evaluation) `Rubric` report:

- `reviewType: "pdlc-readiness"`, with `rubricId`/`rubricVersion` identifying the PDLC readiness rubric and `judge` metadata recording who/what evaluated (agent model or Go checker version) — the provenance that keeps results comparable across projects.
- **One category per artifact domain** the profile declares, scored on the **checklist scale**: `checklistResults.requiredPresent` / `requiredMissing` carry the presence facts (API definition exists, user guide exists, target-locale catalogs exist, ...).
- **Quality facts** come from leaf reports when the corresponding tool has run (`eval/*.eval.json`, `quality/design-system/`, `quality/api-style/`, `quality/a11y/`, `quality/l10n/`, `quality/acceptance/`, `quality/consistency/`). A domain whose tool hasn't run scores `partial` with a `TOOL_NOT_RUN` reason code — never silently `pass`.
- Profile-excluded domains are omitted from categories and listed under `extensions.excluded`.
- `pass` is the explicit gate: all required checklist items present and no blocking reason codes. `blocking` carries what failed.
- `extensions` carries computed detail: locale coverage percentages, acceptance coverage (FRs with passing journeys / total normative FRs), profile and overrides in effect.

An agent can execute this evaluation directly (presence checks are file operations; leaf reports are JSON to read), and the planned Go checker implements the identical rubric — same report either way, which is what makes adoption results comparable across projects and across executors.

## Idempotence

Running adopt + evaluate on a conformant repository must: move nothing, change nothing except a refreshed `quality/readiness.json`, and append one evaluation entry to the audit log. This property is what lets the procedure run repeatedly — locally, in CI, or by different agents — without drift.
