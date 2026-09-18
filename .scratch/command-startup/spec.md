# Shared data-command startup

Status: resolved

## Scope

Consolidate repeated startup across Search, Bookmarks, and View. Preserve typed
requests and output, startup failure behavior, lazy initialization, and cached
initialization success or failure. Help and invalid arguments must not trigger
state loading or network work. Generator initialization follows the operation
catalog and occurs only for commands whose declared operations require it.

## Implementation

`cmd/twt/dependencies/runtime` owns local state loading, contract availability,
authentication loading, generator selection, transport construction, and
concurrency-safe deferred initialization. Each command supplies its operation
names and typed requester constructor. Existing source and decoding modules are
unchanged. Former per-command lazy-initialization tests now exercise the shared
runtime; command-level contract failure regressions remain.

## Verification

Focused tests cover concurrent initialization, cached failures, state-loading
errors, requester reuse, and operation-specific generator initialization.

`make check` passed (formatting, vet, and race-enabled tests). The built CLI
returned help and argument errors for all three commands with unavailable local
state, without attempting startup.
