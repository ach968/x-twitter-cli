# View an X post and its conversation

Status: resolved

## Problem

An agent with an X post link cannot currently retrieve that post and its
conversation through the client's existing authentication and canonical post
model.

## Accepted behavior

- Invoke `twt view <url-or-id>` with one post link or post ID.
- Accept recognized X and legacy Twitter status links. Ignore sharing query
  parameters and use the post ID as identity even if the URL's author handle is
  outdated. Reject unrelated URLs and malformed IDs locally.
- Return the requested post, its parent chain, and replies beneath the requested
  post. Do not expand other branches from ancestor posts.
- Return one X-provided page per invocation. Continue only when the caller
  explicitly requests continuation via `--cursor`, using X's native forward
  pagination. Return the unchanged forward cursor as nullable `next_cursor`.
- Include `post_id` on every page. Do not fetch the requested post again on
  continuation. Return `post: null` and `ancestors: []` when the current page
  omits them; this omission alone is not an unavailable-post error.
- Separate JSON into `post`, `ancestors`, and `replies`; reuse the shared
  canonical post model for all posts, including its embedded quote/media fields.
- Preserve X's reply order. Order available ancestors from oldest ancestor to
  immediate parent.
- Return a structured error if the requested post is unavailable.
- If the requested post is available but surrounding posts cannot be retrieved,
  return available posts with warnings and explicit incomplete retrieval
  metadata.
- Use `partial: true` for known gaps on the current page. Normal pagination or
  omitted context on continuation does not set `partial` by itself.
- Existing Search and Bookmarks commands remain usable with their valid saved
  contracts. A missing View contract requires explicit `twt contract refresh`
  only for View.

## Command contract

```sh
twt view 'https://x.com/InternetH0F/status/2100261836654309833'
twt view 2100261836654309833
twt view 2100261836654309833 --cursor '<next_cursor>'
```

Exactly one positional input is required. The original `--user`/`--id` proposal
is superseded. No automatic traversal, backward pagination, page-size override,
local reply ranking, or extra context fetch is part of this command.

The following parsing details are included for final review:

- Accept a positive decimal post ID that fits an unsigned 64-bit integer;
  preserve its exact string form in output. Reject signs, zero, leading zeros,
  non-digits, and overflow.
- Accept HTTP or HTTPS status URLs on the exact hosts `x.com`, `www.x.com`,
  `mobile.x.com`, `twitter.com`, `www.twitter.com`, and `mobile.twitter.com`.
- Accept `/<handle>/status/<id>`, `/i/status/<id>`, and `/i/web/status/<id>`, with
  an optional trailing slash. Also accept a final `/photo/<positive-index>` or
  `/video/<positive-index>` suffix on these paths, identifying the same post.
- Ignore query parameters and fragments. Do not resolve redirects or contact
  the supplied hostname; parse the ID locally and use the captured X contract.
- Reject user information, explicit ports, unknown hosts, encoded path
  separators, unrelated routes, missing IDs, and extra path components.
- A supplied cursor must be non-empty and otherwise pass through unchanged.
  The caller must use it with the same requested post. Do not attempt to decode
  it locally or silently restart pagination if X rejects it.

## Output contract

Successful calls emit one JSON object to stdout, using existing command exit
and error conventions. Every field below is present:

| Field | Shape | Meaning |
| --- | --- | --- |
| `post_id` | string | Requested post ID, present on every page |
| `post` | canonical Post or null | Requested post if included on this page |
| `ancestors` | array of canonical Post | Available parent chain, oldest first |
| `replies` | array of canonical Post | Replies beneath the target in X order |
| `next_cursor` | string or null | Unchanged forward continuation supplied by X |
| `partial` | boolean | Known missing or unsupported surrounding page content |
| `warnings` | array of code/message objects | Non-fatal page limitations |

On initial reads, missing requested content is not a successful empty result:
classify explicit upstream unavailability as a structured unavailable-post error;
classify a response that cannot be understood as a response-shape failure. Do
not infer deletion or access denial merely from an unfamiliar envelope.

On continuation, absent requested context is valid. Return null/empty fields
without an extra request or warning solely for that omission. Explicit upstream
errors still fail the request. Empty reply pages may still carry a next cursor;
preserve it. Null `next_cursor` means no recognized forward cursor was supplied,
not proof that every reply on X has been retrieved. `partial: false` likewise
does not claim complete conversation retrieval.

Known unavailable or unsupported surrounding entries produce warnings and
`partial: true`, while retaining decodable in-scope posts. Whole-request failures
produce structured errors rather than invented partial output.

## Normalization and pagination

Use the existing canonical Post model and `models.DecodePostResult`. The View
operation owns traversal of the TweetDetail envelope, relationship classification,
cursor interpretation, and page warnings.

- Identify the requested post by exact ID, not entry position or author handle.
- Follow reply relationships to construct the available parent chain; do not
  fetch missing parents. Detect gaps or cycles and report known limitations.
- Traverse actual conversation items and modules, using reply relationships and
  observed module structure to retain replies beneath the requested post.
  Do not flatten all nested tweet objects indiscriminately: quoted posts remain
  inside `quoted_post`, and unrelated recommendations/promoted content are not
  conversation replies.
- Keep collections role-separated. Do not repeat the requested post as a reply
  or promote ancestors into replies. Do not add global state to deduplicate or
  combine pages; callers retain earlier results.
- Preserve nested replies returned in the current page and their canonical
  `reply_to` relationships. A caller may also invoke View on any returned reply.
  Do not automatically open branch pages or traverse the whole reply graph.
- Map a supported top-level forward `ShowMoreThreads` or `Bottom` cursor to
  `next_cursor`. Ignore backward `Top` cursors. Preserve opaque values unchanged.
  Do not select an arbitrary branch cursor as the page's forward cursor.
- Source variants that expose unsupported branch continuations or ambiguous
  relationships must produce explicit partial-result warnings. They must not
  silently disappear or be presented as complete coverage. Capture bounded
  structural evidence before supporting additional cursor variants.

## Authentication, contracts, and recovery

Use the existing direct HTTP client and transaction ID generator. Execute one
TweetDetail data request per View invocation. This reuses authentication state;
it does not perform a fresh interactive login or start a browser to fulfill
ordinary View calls. Normal frontend initialization for transaction IDs remains
the shared client's concern.

Add TweetDetail to supported contracts and to authentication/explicit-refresh
capture and validation. Override only caller-owned semantic variables:
`focalTweetId` and a supplied `cursor`. Capture a first-page contract without
persisting a transient continuation cursor as a default. Preserve captured
feature flags and other request defaults.

Permit previously valid saved contract sets that omit TweetDetail. View must
check for its contract before execution and return `CONTRACT_FAILED` with
`recoveryCommand: "twt contract refresh"` when it is missing or rejected. Other
commands retain their existing operation requirements. Failed refresh validation
must preserve the prior active contract file.

Reuse existing authentication, rate-limit, upstream, and shape-failure behavior.
Do not automatically refresh contracts, retry via Search, or substitute a
browser fallback for a failed direct operation. Public CLI and installed-skill
guidance must describe the new command and continuation semantics; reconcile any
pre-existing local skill edits before changing those files.

## Verification required before completion

- Input parsing: bare IDs, supported URL forms, sharing parameters, obsolete
  handles, malformed IDs/hosts/paths, argument count, and empty cursors.
- Synthetic normalization fixtures: initial root, reply target with multiple
  ancestors, nested replies, embedded quotes/media/notes, unrelated items,
  missing ancestors, unavailable entries, malformed required envelopes, and
  unsupported branch continuations. Fixtures must use minimized synthetic data.
- Pagination: initial ShowMoreThreads, subsequent Bottom/Top, omitted focal post,
  empty pages with cursors, terminal pages, and correct forward-cursor selection.
- Command behavior: canonical JSON fields, structured initial unavailability,
  valid continuation omission, partial warnings, unchanged cursor propagation,
  and no extra context fetch or automatic page traversal.
- Contract compatibility: old sets keep Search/Bookmarks usable, missing View
  returns actionable recovery, new refresh captures and validates TweetDetail,
  failed activation preserves prior contracts, and page cursors are not saved.
- Run repository-required formatting and relevant tests, including `make check`.
  Extend explicit local live verification to exercise semantic View success
  with valid authentication. Do not assert specific account contents, reply
  counts, or identities. Use a live-accessible target rather than treating this
  example post as permanently available; never print raw authenticated payloads.

Deeper source variants remain implementation evidence work. The user-visible
fallback for unrecognized surrounding content is the accepted partial-page
behavior; an unrecognized required envelope is a structured failure.

## Evidence

See [the live audit](evidence-audit-2026-09-16.md) and
[the decision record](design-interview.md).

Authenticated browser navigation issued `TweetDetail`, with `focalTweetId`
selecting the requested post. Its observed envelope contains canonical-compatible
tweet result objects, conversation modules, and a `ShowMoreThreads` cursor.
An ephemeral contract subsequently succeeded through the existing direct client
for initial, cursor, and reply-target requests. The continuation response omitted
the requested post and returned `Top`/`Bottom` cursors. A reply-target response
contained one ancestor. See the audit for the sample's coverage limits.

## Confirmation

Q1-Q13 are resolved. The user confirmed the consolidated design with "no
complaints" on 2026-09-16 and then explicitly invoked the implement workflow.
Implementation is complete.

## Implementation verification (2026-09-16)

- TDD covered command input/output, the View operation, contract loading and
  activation, and Chromium capture. Tests use synthetic source data.
- `make check` passed formatting, vet, and the full deterministic race suite.
- Chromium capture regression tests passed with the race detector.
- The opt-in authenticated live suite passed initial View and available
  continuation reads using a target discovered during capture.
- The actual CLI passed initial, continuation, and reply-target checks for the
  supplied example. Ordinary continuation omission and a known unsupported
  surrounding item were represented correctly; no raw payloads were retained.
- Standards review: no documented violations; one helper-name suggestion fixed.
- Spec review: no actionable deviations. An additional unavailable-reply
  regression was reproduced and fixed, then focused race tests passed.

Only View-related changes are included; concurrent Bookmarks loader/probe edits
are outside this implementation.
