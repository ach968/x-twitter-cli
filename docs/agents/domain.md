# Domain docs

This repository uses a single domain context.

## Before exploring

Read:

- `CONTEXT.md` for the project's canonical domain vocabulary
- Relevant records under `docs/adr/` for architectural decisions affecting the work

If either location does not exist, proceed without flagging its absence. Domain documentation is created only when terminology or a qualifying architectural decision is resolved.

## Layout

```text
/
├── CONTEXT.md
├── cmd/
│   └── twt/
├── internal/
│   └── app/
├── docs/
│   └── adr/
└── .scratch/
    └── .sandbox/
```

## Use canonical vocabulary

Use terms as defined in `CONTEXT.md` when naming domain concepts in issues, specifications, tests, hypotheses, and implementation proposals.

If a needed concept is absent, reconsider whether existing vocabulary already covers it. If it represents a real domain distinction, resolve it through the domain-modeling workflow.

## Flag conflicting decisions

If proposed work conflicts with an existing ADR, identify the conflict explicitly instead of silently overriding the decision.
