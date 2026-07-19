# PDLC Formal Model

`pdlc-lifecycle.pidl.json` is the formal, machine-readable model of the PDLC lifecycle, authored in the [PIDL](https://github.com/grokify/pidl) **process** profile. It is the executable/visualizable companion to the prose [specification](../docs/specification/) — the same lifecycle, expressed as typed steps, data flows, and gates.

## What it models

- **Steps** as PIDL entities, typed by how they run: `llm` (agent-drafted stages), `human` (approval gates), `deterministic` (the `pdlc check` readiness step), `external` (the builder).
- **Data flows** between steps, whose ports are the **canonical layout paths** from [`layout.yaml`](../layout.yaml) — e.g. `docs/specs/source/prd.md`, `acceptance/journeys`, `quality/readiness.json`. The model is tied to the real repository contract, not an abstraction of it.
- **Gates** as conditional flows (`definition_approval.approved == true`, `readiness.pass == true`) into human steps.
- **The handoff**: the deterministic readiness check feeds the human baseline gate, which hands a revision-pinned baseline to the builder; implementation feedback loops back to the next definition.

## Working with it

```bash
pidl validate model/pdlc-lifecycle.pidl.json
pidl generate -f mermaid model/pdlc-lifecycle.pidl.json     # sequence diagram
pidl generate -f d2-flow model/pdlc-lifecycle.pidl.json     # flow diagram
```

The `.pidl.json` is the source of truth; diagrams are generated, never committed.
