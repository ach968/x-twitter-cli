# Bookmarks operation evidence audit — 2026-09-09

This record summarizes an authenticated, read-only structural inspection of
X's Bookmarks UI. It excludes bookmark text, authors, post IDs, cursor values,
authentication material, and raw response payloads.

## UI route

- X redirected `/i/bookmarks` to `/i/history`.
- The Bookmarks tab was selected by default.
- A separate Search Bookmarks control was available without a Premium
  subscription.

## All Bookmarks

- The current GraphQL operation name was `Bookmarks` and the request used GET.
- The initial request supplied `count: 20` and `includePromotedContent: true`.
- A later request supplied an opaque `cursor`; the UI used `count: 40` for that
  request.
- The source envelope was
  `data.bookmark_timeline_v2.timeline.instructions`.
- The populated initial response contained ordinary post entries and cursor
  entries. A later response at the end of the observed history contained cursor
  entries without post entries.

## Bookmark search

- The current GraphQL operation name was `BookmarkSearchTimeline` and the
  request used GET.
- The request supplied `rawQuery` and `count: 20`.
- The source envelope was
  `data.search_by_raw_query.bookmarks_search_timeline.timeline.instructions`.
- A populated response contained ordinary post entries and cursor entries.
- A valid no-match response contained cursor entries without post entries.
- The populated post structure matched the observed post structure used by the
  existing Search normalization at the inspected field level. This supports
  considering shared normalization, but does not by itself settle that design.

## Evidence limits

- No raw response was retained by this structural inspection.
- No later bookmark-search page with a supplied cursor was available naturally.
- Deleted, withheld, unavailable, tombstone, quoted-post, Community Note, and
  other exceptional result shapes were not positively established here.
- Operation identifiers, feature flags, field toggles, and transaction-ID
  requirements remain volatile contract evidence and are not accepted design
  constants.
