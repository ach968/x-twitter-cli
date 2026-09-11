# 07: Bookmark search and contract completion

**What to build:** Complete the Bookmarks data command with X-provided bookmark
search, including stable query handling, later-page continuation, no local
fallback, and final conformance of code, tests, help, and documentation to the
accepted Bookmarks contract.

**Blocked by:** 06: Bookmark pagination and partial success.

**Status:** ready-for-agent

**Completion notes (2026-09-11):** Added `--search` command parsing, bookmark
search source selection, combined cursor forwarding, help, schema contract,
and populated/no-match search fixtures. Verification: focused CLI and adapter
tests, schema conformance, and `go test ./...`.

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, Bookmarks evidence audit, shared post and bookmark
page contracts, and every blocking ticket's completed notes. Treat them as
authoritative.

- [x] `twt bookmarks --search` requires exactly one non-empty query, rejects
  missing or empty values before requesting X, and passes valid input unchanged.
- [x] Bookmark search can combine its query with the same opaque cursor option
  used by All Bookmarks, and both values pass unchanged.
- [x] The Bookmarks Execute interface selects bookmark search when the optional
  query is present while keeping source-operation selection hidden from callers.
- [x] The production source adapter selects BookmarkSearchTimeline and relies on
  its captured default variables without application-owned page-size policy.
- [x] Search never downloads All Bookmarks for local filtering and never falls
  back to local search when the source operation fails.
- [x] Every successful bookmark-search page returns the unchanged query,
  normalized bookmarked posts in X's order, the active next cursor or null, and
  warnings including an empty array when none occurred.
- [x] A valid no-match source page succeeds with an empty bookmark collection
  and preserves any active continuation independently.
- [x] Populated, empty, later, partial, unknown-entry, invalid-envelope, and
  classified-failure bookmark-search fixtures behave according to the same
  stable page and failure contracts as All Bookmarks.
- [x] CLI acceptance coverage includes listing, search, both cursor forms,
  malformed flags and values, output streams, and success or failure statuses.
- [x] Human-readable help explains listing, bookmark search, pagination, query
  repetition, private output, warnings, and recovery without exposing source
  request grammar.
- [x] The shared post documentation and schema, Search page contract, and
  Bookmarks page contract reference one consistent normalized post definition.
- [x] No private bookmark capture, authentication state, personal query,
  identifier, cursor, or authenticated upstream response body enters committed
  fixtures, diagnostics, or documentation.
- [x] Formatting, static analysis, deterministic tests, race tests, schema
  validation, and all established repository checks pass.
- [x] The finished command remains read-only and excludes Bookmark Folders,
  writes, multiple-account selection, raw output, local sorting, local
  deduplication, automatic history traversal, and runtime schema versioning.
