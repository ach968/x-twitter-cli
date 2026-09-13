# 01: CLI foundation and empty Search pages

**What to build:** Deliver a reusable CLI application shell and a working `twt search` path that can execute a Search request and return a complete normalized empty search page. The shell centralizes command routing, help, argument validation, dependency wiring, streams, JSON rendering, failures, and exit statuses so later data commands can remain thin.

**Blocked by:** None (can start immediately).

**Status:** resolved

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [x] The executable delegates to one testable CLI application interface and contains no Search-specific policy.
- [x] `twt search` requires exactly one non-empty positional query and supports optional tab and cursor flags.
- [x] Tab input defaults to Top, accepts supported values case-insensitively, and reaches the request as the corresponding X product value.
- [x] A supplied cursor reaches the existing Search request as an opaque continuation value without modification.
- [x] A valid empty SearchTimeline source payload produces a successful page containing the original query, canonical lowercase tab, an empty result array, the active next cursor or null, and an empty warning array.
- [x] Root help and Search help are human-readable and successful; malformed commands, queries, flags, and tabs fail without invoking the request client.
- [x] Successful JSON is written to standard output, failures are written to standard error using the established stable error contract, and process statuses distinguish success from failure.
- [x] The production command loads existing state and contracts and invokes the existing request client once, without implicit login, refresh, browser launch, or retry.
- [x] CLI behavior is tested through the application interface with controlled arguments, streams, and dependencies rather than subprocesses or live X.
- [x] The application shell supports registering a later real command without placeholder commands or speculative operation models.
- [x] Deterministic repository checks remain green.
