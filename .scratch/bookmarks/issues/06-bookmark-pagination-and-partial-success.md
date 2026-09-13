# 06: Bookmark pagination and partial success

**What to build:** Make All Bookmarks resilient across later pages and source
drift by preserving X's continuation and order, returning usable partial pages,
and failing safely when no meaningful bookmark page can be decoded.

**Blocked by:** 05: All Bookmarks command.

**Status:** resolved

**Completion notes (2026-09-11):** Added opaque cursor forwarding, independent
empty-page continuation, preserved source order, and safe partial-success
warnings. Synthetic list/later/empty/continuation/partial/invalid fixtures are
covered through the operation seam. Verification: focused Bookmarks tests and
`go test ./...`.

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, Bookmarks evidence audit, shared post and bookmark
page contracts, and tickets 01-05's completed notes. Treat them as
authoritative.

- [x] `twt bookmarks --cursor` accepts one opaque value and passes it unchanged
  as the only caller-selected pagination input.
- [x] One invocation returns one source-supplied page and does not traverse the
  remaining bookmark history automatically.
- [x] The active next cursor is returned unchanged; absent continuation is null.
- [x] Bookmark collection and continuation are decoded independently, so a
  valid empty collection can retain a non-null next cursor.
- [x] Recognized bookmarked posts retain X's relative order without local
  sorting, deduplication, or limiting.
- [x] Presentation-only and continuation entries are ignored without warnings.
- [x] An unsupported result-bearing entry is skipped while surrounding usable
  posts remain, and the page includes `UNKNOWN_BOOKMARK_ENTRY` with the accepted
  safe message.
- [x] A page with usable posts and warnings succeeds with a zero exit status.
- [x] An invalid source envelope or a page with nothing meaningfully decodable
  fails with `RESPONSE_SHAPE_CHANGED` rather than masquerading as an empty page.
- [x] Every successful page contains `query`, `bookmarks`, `next_cursor`, and
  `warnings`, including empty arrays and nulls required by the contract.
- [x] No caller-selected count, page number, hidden retry, raw output, or local
  continuation interpretation is introduced.
- [x] Exact deterministic fixtures cover initial, later, empty, empty-with-
  continuation, partial, unknown-entry, and invalid-envelope pages.
- [x] Human-readable and machine-readable bookmark page contracts agree with
  all pagination and warning outcomes.
- [x] Deterministic repository checks remain green.
