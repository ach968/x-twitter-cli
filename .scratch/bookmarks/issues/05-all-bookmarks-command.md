# 05: All Bookmarks command

**What to build:** Deliver a complete first-page `twt bookmarks` path that
returns the authenticated identity's All Bookmarks as one stable normalized
bookmark page, including valid empty accounts and ordinary bookmarked posts.

**Blocked by:** 04: Four-operation contract refresh and live smoke.

**Status:** resolved

**Completion notes (2026-09-11):** Implemented the read-only `twt bookmarks`
listing path with a controlled Execute seam, direct Bookmarks adapter, stable
normalized page output, shared-post schema reference, and synthetic fixture
coverage. Verification: focused Bookmarks tests and `go test ./...`.

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, Bookmarks evidence audit, shared post contract, and
tickets 01-04's completed notes. Treat them as authoritative.

- [x] The root CLI routes `twt bookmarks` and provides human-readable root and
  command help without changing existing commands.
- [x] The command reads the identity in the application profile without account
  selection, confirmation, implicit login, contract refresh, browser launch,
  retry, or write behavior.
- [x] One Bookmarks operation Execute interface accepts an All Bookmarks request
  and returns a normalized bookmark page or safe typed failure.
- [x] The operation owns one internal Fetch source seam, and its production
  adapter selects the Bookmarks source operation.
- [x] The production adapter relies on captured default request variables and
  does not introduce application-owned count constants.
- [x] A valid empty source page succeeds with `query` equal to null, an empty
  `bookmarks` array, the available next cursor or null, and an empty `warnings`
  array.
- [x] Ordinary bookmarked posts are returned directly in `bookmarks` using the
  complete shared normalized post contract and X's returned order.
- [x] The output does not invent a bookmark wrapper, identifier, timestamp, or
  redundant bookmarked flag.
- [x] Successful JSON is written to standard output with a zero exit status;
  safe failure JSON is written to standard error with a non-zero exit status.
- [x] Ordinary results are not persisted, and no raw-output option is exposed.
- [x] The initial Bookmarks page contract and machine-readable schema document
  the complete successful shape and reference the shared post contract.
- [x] Operation and CLI behavior is verified with minimized synthetic source
  evidence and controlled dependencies rather than live X.
- [x] Deterministic repository checks remain green.
