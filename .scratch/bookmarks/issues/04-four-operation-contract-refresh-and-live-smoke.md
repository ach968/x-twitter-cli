# 04: Four-operation contract refresh and live smoke

**What to build:** Make contract refresh and the explicit live suite prove that
the current authentication state can execute HomeTimeline, SearchTimeline,
Bookmarks, and BookmarkSearchTimeline through valid current operation
contracts, without depending on account content.

**Blocked by:** 03: Shared direct-operation transport contraction.

**Status:** ready-for-agent

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, Bookmarks evidence audit, and tickets 01-03's
completed notes. Treat them as authoritative.

- [x] Bookmarks and BookmarkSearchTimeline join HomeTimeline and SearchTimeline
  in the required operation-contract set.
- [x] Contract properties are rejected when any required operation is missing,
  duplicated, malformed, unsupported, or unsafe.
- [x] The browser capture module owns a recipe for each required operation and
  callers do not learn X controls or selectors.
- [x] One Bookmarks-page visit captures the Bookmarks operation contract, uses
  the Search Bookmarks control with the validation query, and captures the
  BookmarkSearchTimeline operation contract.
- [x] Candidate contract properties activate only after all four direct
  operations execute successfully.
- [x] Bookmark-search validation uses a deliberately improbable query and does
  not depend on matching private content.
- [x] The explicit live suite makes one representative direct request for each
  required operation and requires no transport error, a 2xx upstream response
  containing valid JSON, and no classified operation failure.
- [x] Live assertions do not depend on result counts, result kinds, next
  cursors, normalized content, Search-tab variants, or pagination.
- [x] A fresh account with valid empty operation responses can pass the live
  suite.
- [x] Live failure diagnostics never print an authenticated upstream response
  body; they expose only safe failure and response metadata.
- [x] Authenticated live verification remains explicit and opt-in rather than
  part of the deterministic suite.
- [x] Deterministic contract, browser-capture, activation, and request tests
  remain green.

## Completion notes

- Required contracts now include HomeTimeline, SearchTimeline, Bookmarks, and
  BookmarkSearchTimeline. Loading rejects malformed, incomplete, unexpected,
  and duplicate operation properties.
- Contract capture keeps the Bookmarks-page recipe and Search Bookmarks control
  interaction in `internal/app/browser`; management asks for one bookmarks-page
  visit after Home and Search. The validation query is deliberately improbable.
- Candidate activation executes all four direct operations before promotion.
  The explicit opt-in live suite mirrors that semantic smoke test and logs only
  safe failure metadata, never authenticated response bodies.
- Verified with deterministic `go test ./...`, browser-tag capture tests, and
  live-tag compilation/skipped tests. No authenticated live X request was run.
