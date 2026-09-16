# TweetDetail evidence audit (2026-09-16)

This is a redacted, authenticated structural observation for
`https://x.com/InternetH0F/status/2100261836654309833`. No cookies,
authorization values, response bodies, cursor values, or raw captures are
stored here.

## Observed request

Navigating the status URL produced HTTP 200 GraphQL requests for both
`TweetResultByRestId` and `TweetDetail`. The latter used `focalTweetId` equal
to the supplied status ID. The observed `TweetDetail` response was 54,249
bytes in this run.

## Observed envelope

The post/thread data is at:

`data.threaded_conversation_with_injections_v2.instructions`

Instruction types observed were `TimelineClearCache`, `TimelineAddEntries`,
and `TimelineTerminateTimeline`. The add instruction contained 13 entries:
one `TimelineTimelineItem`, eleven `TimelineTimelineModule` entries, and one
`TimelineTimelineCursor`. Each module contained an item whose normalized
source shape was available through `item.itemContent`.

The final entry was a `TimelineTimelineCursor` with cursor type
`ShowMoreThreads`. This is a native X continuation marker; its opaque value
was intentionally not recorded. No `Top` or `Bottom` cursor was observed.
One bounded native scroll on the rendered page exposed no visible “Show more”
control, so no second-page request was made.

## Post relationship counts

Walking tweet result objects and deduplicating by internal post ID found 12
posts: one focal post and eleven posts whose
`legacy.in_reply_to_status_id_str` points directly to the focal post. No
ancestor-chain edges were present in this particular response. This supports
separate focal and direct-reply collections for this sample, but does not
establish that every TweetDetail response has the same counts or nesting.

The existing canonical decoder remains structurally compatible: each tweet
result can be passed to `models.DecodePostResult`, which already handles note
text, media, quoted posts, Community Notes, and cycle/depth protection.

## Initial audit limits

The 200 browser response proves current authenticated browser behavior only.
It does not prove that a direct replay will work until a `TweetDetail`
operation contract is captured, validated, and exercised through the existing
direct transport.

Opening one observed direct-reply URL in the native browser rendered the
focal parent post above the reply, confirming parent context in the UI. This
was a read-only navigation; no interaction or engagement action was taken.

The current persisted contract file contains only `BookmarkSearchTimeline`,
`Bookmarks`, and `SearchTimeline`; it has no `TweetDetail` entry. Therefore a
direct replay was not performed in this bounded pass: doing so would require
capturing the browser request and constructing an ephemeral contract first.

## Follow-up: direct replay and continuation

A subsequent disposable probe captured the browser's TweetDetail request URL
and response in memory, constructed an ephemeral operation contract from its
method, host, path, variables, features, and field toggles, and reused the
repository's `httpclient.Client`, direct transport, stored authentication, and
fresh transaction ID generator. Saved authentication and contract files were
not changed. This supersedes the earlier untested-direct-replay limitation.

Three direct requests returned successful 2xx JSON responses with recognizable
TweetDetail envelopes and decoded posts:

1. Initial page: override `focalTweetId`; requested post present, eleven direct
   replies, and one `ShowMoreThreads` cursor.
2. Continuation: same focal ID, with `variables.cursor` set to the initial
   `ShowMoreThreads` value unchanged. Two direct replies were returned; the
   requested post was absent. Cursor types were `Top` and `Bottom`.
3. Reply target: set `focalTweetId` to one observed direct reply. Requested reply
   present, one available ancestor, one direct reply beneath the new target,
   and one `Bottom` cursor.

The follow-up's recursive structural scan found fourteen unique post objects
on the initial page, including embedded objects; these must not all be promoted
into the conversation collections. The eleven direct-reply relationships are
the relevant count, and quoted posts must remain embedded in their canonical
post model.

Verified implication: a valid continuation page can omit the requested post;
omission on continuation must not by itself be treated as post unavailability.
The captured initial cursor can be passed unchanged through `variables.cursor`.

Not yet established: exhausting all pages, every branch continuation shape,
deep ancestor chains, unavailable-post source variants, or end-of-pagination
behavior. These are coverage limits, not evidence of complete conversation
retrieval. The disposable probe was removed and its browser was closed.

## Implemented CLI verification

The completed command was checked with the supplied URL, its returned forward
cursor, and one of its returned replies as a new target. Safe output summaries:

- Initial: requested post present, eleven replies, next cursor, no known gaps.
- Continuation: requested post absent, two replies, next cursor, partial with
  `UNSUPPORTED_CONVERSATION_ITEM` for a surrounding item.
- Reply target: requested post present, one ancestor, next cursor, no known gaps.

Counts are observations, not test assertions or guarantees. The local opt-in
live suite also passed against a post discovered during capture. Contract refresh
and live checks used system Chromium; the initial managed-Chromium attempt
opened onboarding and was stopped. Applying existing stored authentication to
the application profile allowed the system-Chromium refresh to complete. No
credentials or ordinary source payloads were printed or retained in this audit.
