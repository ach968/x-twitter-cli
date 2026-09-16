# Conversation page contract

`twt view <url-or-id> [--cursor value]` returns one conversation page using
existing authentication. Its machine-readable contract is
[view-page.schema.json](view-page.schema.json). All posts use the
[shared canonical post model](shared-post-contract.md).

| Field | Meaning |
| --- | --- |
| `post_id` | The requested post ID, always a string. |
| `post` | The requested canonical post, or null when omitted on continuation. |
| `ancestors` | Available parent chain, oldest ancestor to immediate parent. |
| `replies` | Replies beneath the requested post, preserving X's order. |
| `next_cursor` | X's unchanged forward cursor, or null. |
| `partial` | Whether known surrounding content on this page could not be returned. |
| `warnings` | Non-fatal limitations, as code/message objects. |

Every successful page includes all seven fields. Collections are arrays,
including when empty. Quoted posts stay in `quoted_post`; they do not become
conversation replies. Nested replies retain their canonical `reply_to` links.
Other reply branches from ancestors and unrelated recommendations are excluded.

## Pagination

Repeat the same requested post with the returned `next_cursor`. The command
executes one TweetDetail request and does not fetch context again or automatically
traverse pages. The caller retains earlier pages.

Later pages can have `post: null` and `ancestors: []`. This means X did not
include that context on this page; it does not mean the requested post was
deleted. Empty reply pages can still have a next cursor.

The command uses supported top-level forward cursors. It does not expose
backward cursors or automatically expand individual reply branches. Unsupported
branch controls produce a partial-page warning. Any returned reply can be used
as the requested post in another View call.

Neither `next_cursor: null` nor `partial: false` proves that every reply on X
has been retrieved. They describe only the received page and its supported
continuation controls.

## Partial pages and failures

Known parent gaps, unavailable surrounding posts, unsupported entries or branch
controls, and ambiguous reply relationships retain usable posts and set
`partial: true`. Normal pagination and omitted continuation context do not.
Warnings are emitted once per code on a page:

- `UNSUPPORTED_CONVERSATION_ITEM`
- `UNSUPPORTED_CONVERSATION_INSTRUCTION`
- `UNSUPPORTED_CONVERSATION_CURSOR`
- `INCOMPLETE_PARENT_CHAIN`
- `UNKNOWN_REPLY_RELATIONSHIP`

Explicit requested-post unavailability returns `POST_UNAVAILABLE`. An initial
response that cannot identify the requested post, or an unrecognized required
envelope, returns `RESPONSE_SHAPE_CHANGED`. Upstream errors are preserved as
structured failures; the client does not infer deletion from unknown data.
Errors go to stderr with a nonzero exit status; partial pages go to stdout and
exit zero.

If View's contract is missing or stale, the failure is `CONTRACT_FAILED` with
`twt contract refresh` as recovery. Older valid contracts remain usable by Search
and Bookmarks before that refresh.
