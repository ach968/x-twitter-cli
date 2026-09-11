# 02: Search operation migration

**What to build:** Move the existing Search data command behind one deep
operation interface so callers receive the same search pages and safe failures
without coordinating source requests or decoding source payloads themselves.

**Blocked by:** 01: Shared post normalization expansion.

**Status:** ready-for-agent

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, current Search contract, and ticket 01's completed
notes. Treat them as authoritative.

- [x] The Search operation exposes one small Execute interface that accepts the
  query, Search tab, and optional next cursor and returns a normalized search
  page or error.
- [x] The Search operation owns its request policy, source interface, source
  envelope, result interpretation, continuation, warnings, and normalized page
  construction.
- [x] A production adapter and controlled test adapter satisfy the operation's
  source seam without exposing it through the CLI interface.
- [x] The Search CLI module owns only help, argument validation, operation
  invocation, JSON rendering, streams, and exit status.
- [x] The shared safe OperationFailure implements Go error behavior while
  retaining its existing JSON code, message, and optional recovery command.
- [x] Classified X failures, response-shape failures, and unexpected internal
  failures retain their established safe caller behavior.
- [x] Search tests observe normalization and failure behavior through the
  Execute interface; obsolete tests below that seam are removed or narrowed.
- [x] Top, Latest, People, Media, and Lists retain their complete accepted
  output, ordering, continuation, partial-success, and nullability behavior.
- [x] `twt search` remains demonstrably compatible for existing callers.
- [x] Deterministic repository checks remain green.

## Completion notes

Search now exposes `Execute(context.Context, Request) (Page, error)` from
`internal/app/operations/search`. Its direct source adapter owns SearchTimeline
selection and Search semantic variables; the CLI only validates input, invokes
the operation, and renders its normalized page or safe typed failure.

Focused operation, CLI, dependency, and full repository tests passed on
2026-09-11.
