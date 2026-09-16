# View command design interview

## Status

- Q1-Q13 resolved. Final shared understanding confirmed on 2026-09-16 with
  "no complaints".
- Decision frontier: empty. Implementation subsequently completed through the
  explicitly invoked implement workflow; see `spec.md` for verification.
- Goal: retrieve an X post using existing authentication and return the
  repository's canonical post model.
- Initial proposed syntax: `twt view --user <handle> --id <post-id>`.
- Accepted syntax: `twt view <url-or-id>`.
- Accepted scope: requested post, its parent chain, and replies beneath the
  requested post. Other reply branches from ancestors are excluded.
- Accepted retrieval policy: one X-provided page per invocation, with explicit
  continuation using `--cursor` and X's unchanged forward `next_cursor`.
- Accepted output: separate `post`, `ancestors`, and `replies`, using canonical
  posts throughout. An unavailable requested post produces a structured error.
- Accepted partial results: return available posts with warnings and explicit
  incomplete retrieval metadata if surrounding posts cannot be retrieved.
- Accepted links: recognized X/Twitter status links; ignore sharing parameters
  and use the post ID even if the handle is outdated. Reject malformed inputs.
- Accepted order: preserve X's reply order; order the available parent chain
  from oldest ancestor to immediate parent.
- Accepted continuation context: no extra request to repeat the requested post.
  Return `post: null` and `ancestors: []` if X omits them on continuation; include
  `post_id` on every page. Omission alone is not an unavailable-post error.
- Accepted compatibility: missing View contracts do not disable existing data
  commands; View requests explicit contract refresh.
- Accepted gap signal: `partial` describes known current-page gaps; normal
  pagination and continuation omission do not make a page partial.
- Recommendations below remain proposals until the user answers.

## Decision tree

1. Input form (resolved, Q1)
   - URL and ID validation; handle semantics; conflicting inputs.
2. Retrieval scope (conversation accepted, Q2; boundaries resolved, Q3)
   - Conversation boundaries; pagination and completeness; output envelope;
     embedded content; unavailable requested posts.
3. Retrieval feasibility (initial, continuation, and reply-target direct reads
   verified; broad source variants remain coverage work)
   - Upstream operation and focal-post identification.
   - Contract capture, validation, and recovery integration.
4. Compatibility and verification (depends on the above)
   - Existing model coverage; failures; live smoke and fixtures.
5. Shared-understanding confirmation (after the frontier is empty)

## Round 1

### Q1: Input form

Should callers pass a URL or post ID positionally, the proposed `--user` and
`--id` flags, or either form?

Recommendation: `twt view <url-or-id>`, allowing an agent to use a link directly.

Answer: "1. looks good" in response to the recommendation.

Decision: accept `twt view <url-or-id>`.

### Q2: Retrieval scope

Should the command return only the requested post, or also retrieve its
surrounding conversation?

Recommendation: only the requested canonical post, retaining embedded quote and
media fields supported by the shared model.

Answer: "i say we include all conversation."

Decision: include conversation retrieval. Q3 subsequently settled its boundaries;
Q8, Q10, and Q11 settled caller-controlled native pagination.

## Repository evidence

- `internal/app/models/post.go` defines the shared `Post` model and
  `DecodePostResult`; conversation posts can reuse that model.
- `internal/app/httpclient/client.go` centralizes direct authenticated execution.
- No single-post/detail operation is currently registered in the command or
  contract capture flows.
- Authenticated browser inspection of the supplied URL observed HTTP 200
  `TweetDetail` with `focalTweetId` selecting the requested post, and HTTP 200
  `TweetResultByRestId`. TweetDetail includes tweet results and a cursor-bearing
  timeline envelope. Follow-up direct replay verified initial, continuation,
  and reply-target reads with existing authentication and transport; see
  `evidence-audit-2026-09-16.md` for exact results and coverage limits.

## Round 2

Q1 was subsequently accepted alongside Q3.

### Q3: Conversation boundaries

If the supplied link points to a reply, should retrieval include its parent
chain and replies beneath that target, or start at the original root and include
other branches as well?

Recommendation: the requested post, its parent chain, and replies beneath the
requested post. Whether the user instead wants the entire root conversation is
an explicit decision, not an assumption.

Answer: "3) A".

Decision: requested post, its parent chain, and replies beneath the requested
post. Do not expand other branches from the conversation root.

## Round 3

### Q4: Output organization

Should the JSON separate the requested post, ancestors, and replies, or return
one flat list of posts?

Recommendation: separate `post`, `ancestors`, and `replies`, each reusing the
canonical post model. Q10-Q11 and Q13 subsequently defined the remaining fields.

Answer: "these look good" (Q4 and Q5).

Decision: accepted. Separate `post`, `ancestors`, and `replies`.

### Q5: Unavailable requested post

If the requested post is deleted or inaccessible to the authenticated identity,
should the command fail, or return surrounding posts when available?

Recommendation: fail with a structured error when the requested post cannot be
retrieved. Missing surrounding posts are a separate partial-result decision.

Answer: "these look good" (Q4 and Q5).

Decision: accepted. Fail with a structured error if the requested post is
unavailable; Q6 and Q13 subsequently settled surrounding-post failure behavior.

## Round 4

### Q6: Partial conversation

If the requested post is retrieved but some surrounding posts cannot be
retrieved, should the command return the available posts with explicit warnings,
or fail the whole invocation?

Recommendation: return available posts with warnings and explicit incomplete
retrieval metadata. Exact metadata and interruption handling depend on the
retrieval policy still to be decided.

Answer: "good" (Q6 and Q7).

Decision: accepted. Return available posts with warnings and explicit incomplete
retrieval metadata.

### Q7: Accepted links and identity

Should input accept both X and legacy Twitter status links, including share
query parameters, and use the post ID as the identity rather than requiring the
handle in the link to match the current author?

Recommendation: accept recognized X/Twitter status-link forms, ignore share
parameters and use the post ID. Reject unrelated URLs and malformed IDs locally.
The exact accepted route and hostname list must be recorded in the final spec.

Answer: "good" (Q6 and Q7).

Decision: accepted. Recognize X/Twitter status links, ignore sharing parameters,
and identify the post by ID. Exact hostname/route grammar remains to be specified.

## Round 5

### Q8: Retrieval amount per invocation

For a large conversation, should one invocation automatically keep fetching
until available replies are exhausted, or return one batch with caller-controlled
continuation?

Recommendation: one batch with explicit continuation, keeping request volume and
output size under the caller's control. This is a product-policy proposal;
upstream continuation capabilities and the exact continuation format still need
live verification. Automatic traversal would require a separate limit and
interruption policy.

Answer: "i think we can use X's innate pagination for this".

Decision: use native X pagination, one upstream page per invocation with explicit
continuation. Do not automatically crawl the conversation. Exact cursor fields
and any separate branch-continuation behavior remain evidence-dependent.

## Round 6

### Q9: Conversation order

Should replies preserve X's returned order, or be locally sorted chronologically?

Recommendation: preserve X's reply order and arrange the available parent chain
from oldest ancestor to immediate parent. Do not locally re-rank replies.

Answer: "yes".

Decision: accepted. Preserve X's reply order and return the available parent
chain from oldest ancestor to immediate parent.

## Round 7

### Q10: Context on continuation pages

Live direct replay showed that a valid continuation page can omit the requested
post while returning more replies. Should output reflect that omission, or make
an extra request to repeat the requested post and ancestors on every page?

Recommendation: keep one native page per invocation. Return `post: null` and
`ancestors: []` when omitted on continuation, plus a stable `post_id` identifying
the requested post on every page. This omission is not an unavailable-post error.

Answer: "ye", after a simpler example explaining that later pages add replies
without fetching the requested post again.

Decision: accepted. One request per page, with nullable `post` on continuation,
available ancestors only, and stable `post_id`. Do not classify normal
continuation omission as post deletion or access denial.

### Q11: Forward cursor contract

Live replay consumed the initial `ShowMoreThreads` value through `cursor` and
returned `Top`/`Bottom` cursors on the next page. Should callers use the existing
CLI pattern `--cursor`, with a nullable `next_cursor` in output?

Recommendation: yes. Pass the opaque forward cursor unchanged; use the same
target for continuation. Do not expose backwards pagination. Nested branch
continuations still require separate evidence and must not be silently dropped.

Answer: "11 looks good".

Decision: accepted. Use `--cursor` and nullable `next_cursor`, passing X's forward
cursor unchanged. Backward pagination is out of scope. Q10 was subsequently
accepted after a simpler explanation.

## Round 8

### Q12: Existing installations

The saved operation contracts do not yet include TweetDetail. Should existing
Search and Bookmarks keep working until the user explicitly refreshes contracts,
with only View requesting that refresh?

Recommendation: yes. A missing View contract produces `CONTRACT_FAILED` with
`twt contract refresh`; it does not invalidate otherwise valid existing
contracts for Search and Bookmarks. Authentication and contract refresh capture
and validate View for future use.

Answer: "yes" (Q12 and Q13).

Decision: accepted. Existing commands remain usable with their valid contracts;
only View requires its missing operation contract to be refreshed.

### Q13: Partial-result metadata

Should `partial` describe known missing or unsupported surrounding content on
the current page, separately from `next_cursor` indicating another page?

Recommendation: yes. Return `partial: true` with warnings for known gaps. More
pages alone does not make a page partial. `partial: false` never promises that
every reply on X was retrieved, and normal continuation omission of the requested
post is not a gap.

Answer: "yes" (Q12 and Q13).

Decision: accepted. Use `partial` plus warnings for known current-page gaps,
independently from nullable `next_cursor`.

## Consolidation

All interview choices are recorded in `spec.md`. Detailed input grammar,
normalization safeguards, and verification coverage are included there for
final review. Unknown upstream variants are handled through the accepted partial
result or structured failure rules; they do not justify fabricated content or
claims that the entire conversation has been downloaded.

Final shared-understanding confirmation: "no complaints" on 2026-09-16.

The consolidated specification, including detailed parsing rules and verification
coverage, is accepted. The design interview is complete.
