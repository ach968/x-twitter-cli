# 01: Shared post normalization expansion

**What to build:** Introduce the shared normalized post contract and
construction behavior beside the current Search implementation, while keeping
the complete `twt search` experience unchanged. This is the expand step that
makes the later Search migration and Bookmarks operation safe.

**Blocked by:** None (can start immediately).

**Status:** resolved

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, current Search contract, and Bookmarks evidence
audit. Treat them as authoritative.

- [x] One cohesive shared models module owns the normalized post, compact user
  reference, entities, media, reply context, metrics, quoted post, Community
  Note, and their shared source-to-model construction behavior.
- [x] Search-specific pages, result unions, users, lists, tabs, timeline
  instructions, cursors, and warnings remain outside the shared models module.
- [x] Dedicated human-readable documentation and JSON Schema define the shared
  post contract, and the Search page contract references that definition rather
  than duplicating it.
- [x] Existing Search callers remain supported through temporary compatibility
  delegation while the shared form is introduced.
- [x] `twt search` produces the same normalized JSON, failures, warnings,
  ordering, and exit statuses as before the expansion.
- [x] Shared-model behavior is verified through existing Search outcomes;
  narrower tests exist only where the Search seam cannot express the behavior.
- [x] No private captured source payload, authentication state, personal
  identifier, or unnecessary source bulk is added to the repository.
- [x] Deterministic repository checks remain green.

## Completion notes

- Added `internal/app/models`, the cohesive owner of shared post types and
  source-to-model construction behavior. Search retains compatibility aliases
  and delegates its post construction to this module.
- Added the shared human-readable contract and JSON Schema. The Search page
  schema now references the shared post schema.
- Added a Search-seam assertion that decoded posts have the shared `models.Post`
  type. Existing Search golden and behavior tests remain the regression proof
  for output compatibility.
- Verified with `go test ./internal/app/search`, `go test ./internal/app/...`,
  `gofmt`, and `git diff --check`.
