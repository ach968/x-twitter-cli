# Bookmarks design interview

This document is the durable decision tree for `twt bookmarks`. It records the
evidence, recommendations, answers, and accepted decisions that shape the
command. Recommendations remain proposals until Matt explicitly accepts or
revises them.

The v1 release decision on 2026-09-12 made HomeTimeline optional WIP. Search,
Bookmarks, and BookmarkSearchTimeline are the required capture, activation, and
live-verification set. That decision supersedes the older four-operation
requirements retained later in this interview as design history.

## Status

- Q1-Q40: resolved
- All Bookmarks and bookmark search: in scope
- Bookmark Folders: deferred because no representative Premium evidence is
  available
- Decision frontier: empty
- Final shared understanding confirmed by Matt on 2026-09-11

## Current evidence

An authenticated, read-only browser inspection established the following
without retaining bookmark text, authors, post IDs, cursors, or authentication
material:

- X serves the Bookmarks UI at `/i/history` and exposes Search Bookmarks without
  a Premium subscription.
- All Bookmarks currently uses a GET `Bookmarks` operation. Its initial request
  supplied `count: 20` and `includePromotedContent: true`; an observed later
  request supplied a cursor and `count: 40`.
- Bookmark search currently uses a separate GET `BookmarkSearchTimeline`
  operation with `rawQuery` and `count: 20`.
- Both operations return timeline instructions containing ordinary post entries
  and opaque cursor entries. Bookmark search can return a valid page containing
  cursors but no posts.
- The inspected post structures match the structures already normalized for
  Search. X did not expose a distinct bookmark ID or bookmark timestamp.
- Exact response paths, limits, and unresolved evidence are recorded in
  [evidence-audit-2026-09-09.md](evidence-audit-2026-09-09.md).

## Resolved decisions

### Q1: Bookmark Folder scope

**Question:** Should v1 read only All Bookmarks, or also support Premium Bookmark
Folders?

**Recommendation:** Support All Bookmarks only. Folder support requires current
operation and response evidence.

**Matt's answer:** Shipping folders is not practical because the application
identity has no Premium subscription or folders, so their response shapes
cannot be observed.

**Decision:** Accepted. Bookmark Folders are out of scope until representative
evidence is available.

### Q2: Retrieval unit

**Question:** Should one invocation return one X-supplied page or automatically
retrieve the complete bookmark history?

**Recommendation:** Return one page and expose an opaque next cursor. Do not
traverse every page automatically.

**Matt's answer:** Approved provisionally, subject to current response evidence.

**Decision:** Accepted after both live operations showed cursor-based page
boundaries.

### Q3: Private output confirmation

**Question:** Should `twt bookmarks` ask for confirmation before printing the
authenticated identity's private bookmarks?

**Recommendation:** No. Running the explicit data command is sufficient intent,
and a prompt would obstruct scripts and coding agents. Do not persist ordinary
results automatically.

**Matt's answer:** Yes to the recommendation.

**Decision:** Accepted. Print immediately without persisting the result.

### Q4: Evidence gate

**Question:** Should the output and transport be designed from assumptions, or
only after inspecting current Bookmarks responses?

**Recommendation:** Inspect current evidence first, including populated, later,
and empty behavior where naturally available.

**Matt's answer:** Evidence first.

**Decision:** Accepted. Current structural evidence was collected before the
output contract interview continued.

### Q5: Listing command shape

**Question:** Should listing use `twt bookmarks` with only an optional cursor?

**Recommendation:** Yes. Do not accept an account, folder, page number, or input
from standard input.

**Matt's answer:** Yes.

**Decision:** Accepted. Bookmark search later extends this command with its own
option.

### Q6: Empty page

**Question:** Should a structurally valid page with no bookmarked posts succeed?

**Recommendation:** Yes. Return an empty collection and a null next cursor.

**Matt's answer:** Yes.

**Decision:** Accepted.

**Later clarification:** Q33 supersedes the assumption that an empty collection
always implies a null continuation cursor.

### Q7: All Bookmarks ordering

**Question:** Should the command preserve X's returned bookmark order?

**Recommendation:** Yes. Do not sort by post creation time or deduplicate
locally.

**Matt's answer:** Preserve X's order.

**Decision:** Accepted.

### Q8: Bookmark-search command shape

**Question:** Should bookmark search use `twt bookmarks --search 'query'` or a
`twt bookmarks search 'query'` subcommand?

**Recommendation:** Use the value-taking `--search` option because search is a
filtered form of the Bookmarks data command from the caller's perspective.

**Matt's answer:** Use `--search`.

**Decision:** Accepted.

### Q9: Search-query handling

**Question:** Should `--search` require one non-empty string and pass it to X
unchanged?

**Recommendation:** Yes. Do not interpret or rewrite X's query language.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q10: Search pagination

**Question:** Should bookmark search use the same `--cursor` option, with the
caller repeating the same search query for another page?

**Recommendation:** Yes. Keep one pagination pattern for listing and search.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q11: Search-operation fallback

**Question:** If X's bookmark-search operation fails, should the command download
All Bookmarks and filter them locally?

**Recommendation:** No. Local fallback could require many requests, return
different matches, and hide an operation-contract failure.

**Matt's answer:** No fallback.

**Decision:** Accepted. Return the operation failure.

### Q12: Bookmarked-post shape

**Question:** Should bookmarked posts use the same normalized post contract as
post results from Search?

**Recommendation:** Yes. A second operation now demonstrates the same observed
post structure, so a shared contract is justified.

**Matt's answer:** Yes.

**Decision:** Accepted. Do not invent a smaller Bookmarks-specific post shape.

### Q13: Page collection name

**Question:** Should a bookmark page call its ordered collection `bookmarks` or
`results`?

**Recommendation:** Use `bookmarks`; each member still has `type: "post"`.

**Matt's answer:** Use `bookmarks`.

**Decision:** Accepted.

### Q14: Page-size option

**Question:** Should v1 expose `--count`?

**Evidence:** X requested 20 entries initially and 40 on an observed later All
Bookmarks request, suggesting an internal UI policy rather than a stable caller
contract.

**Recommendation:** Do not expose `--count`. Return one complete page supplied
by X and reconsider caller control only with stronger evidence.

**Matt's answer:** Approved the approach.

**Decision:** Accepted.

### Q15: Bookmark-search ordering

**Question:** Should bookmark search preserve X's returned order?

**Recommendation:** Yes. Do not apply local sorting or deduplication.

**Matt's answer:** X's default bookmark order is already recency.

**Decision:** Accepted. Preserve upstream order and do not recalculate it from
post creation timestamps.

### Q16: Self-describing bookmark page

**Question:** Should every bookmark page contain `query`, using null for All
Bookmarks and the unchanged query for bookmark search?

**Recommendation:** Yes. Persisted output then identifies how it was produced.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q17: Direct post members

**Question:** Should each member of `bookmarks` be a post directly or a wrapper
around a post?

**Evidence:** The inspected response exposed no bookmark ID or bookmark
timestamp.

**Recommendation:** Return posts directly. Do not add a redundant wrapper or
`bookmarked: true`.

**Matt's answer:** Return posts directly.

**Decision:** Accepted.

### Q18: Partial success

**Question:** If X returns usable bookmarked posts plus an unsupported entry,
should the command keep the posts and add a warning?

**Recommendation:** Yes. A usable partial page succeeds; an invalid envelope or
a page with nothing meaningful fails.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q19: Continuation field

**Question:** Should every bookmark page contain `next_cursor`?

**Recommendation:** Yes. Return X's active continuation unchanged and use null
when no later page exists.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q20: Runtime schema version

**Question:** Should a bookmark page contain a `schema_version` field?

**Recommendation:** No. Add a runtime version only when an actual compatibility
or migration mechanism requires one.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q21: Raw output

**Question:** Should `twt bookmarks` expose a `--raw` source-payload option?

**Recommendation:** No. Keep volatile private payloads in explicit, user-only
evidence tooling.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q22: Warning contract

**Question:** What should happen when either Bookmarks operation encounters an
unsupported result-bearing entry?

**Recommendation:** Preserve usable posts and emit
`{"code":"UNKNOWN_BOOKMARK_ENTRY","message":"Skipped an unsupported bookmark entry"}`.
Do not include source content.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q23: Shared normalization seam

**Question:** Where should the shared post contract and normalization behavior
live?

**Recommendation:** Extract the complete shared post-normalization cluster. Keep
operation-specific page envelopes, timeline instructions, cursors, and warnings
with their operations.

**Matt's answer:** Create `internal/app/models` for all shared normalization
modules, and place Search and Bookmarks under `internal/app/operations/search`
and `internal/app/operations/bookmarks`.

**Decision:** Accepted. Shared models and normalization live under
`internal/app/models`; operation-specific page contracts and source-envelope
decoders live under the corresponding `internal/app/operations` subpackage.
Q26 resolves request-execution ownership.

### Q24: Shared contract documentation

**Question:** How should Search and Bookmarks avoid defining different versions
of the same normalized post contract?

**Recommendation:** Extract dedicated shared post documentation and JSON Schema,
then reference them from both page contracts.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q25: Shared models module depth

**Question:** Should `internal/app/models` contain only shared data structures,
or should it also own the shared normalization behavior that constructs them?

**Recommendation:** Let `internal/app/models` own both the shared normalized
types and the source-to-model normalization behavior. Begin with one cohesive
package and split it only when a demonstrated boundary warrants another
package.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. `internal/app/models` is a complete shared
normalization module rather than a structs-only package. Search- and
Bookmarks-specific envelopes, cursors, warnings, and operation policy remain
under their corresponding operation packages.

### Q26: Request-execution ownership

**Question:** Should the CLI continue coordinating source requests and
normalization, or should each operation own the complete request-to-page use
case?

**Recommendation:** Give each operation a small `Execute` interface that owns
request selection, failure interpretation, envelope decoding, and
normalization. Keep each operation's source interface and production HTTP
adapter in its corresponding operation directory. Reduce
`internal/app/httpclient` to shared authenticated request mechanics such as
contract lookup, headers, transaction IDs, transport, and response capture.

**Matt's answer:** Approved the recommendation and refined the layout so the
operation-specific HTTP adapter functions live under their respective
operation directories rather than in the generic HTTP client package.

**Decision:** Accepted. `internal/app/operations/search` and
`internal/app/operations/bookmarks` each own their source interface, HTTP
adapter, request variables, endpoint selection, failure interpretation,
operation-specific decoding, and normalized page result. The generic
`internal/app/httpclient` module owns only reusable authenticated transport
mechanics. CLI command modules parse arguments, invoke an operation, and render
its result.

### Q27: Bookmarks operation interface

**Question:** Should the Bookmarks module expose separate All Bookmarks and
bookmark-search methods, or one operation interface?

**Recommendation:** Expose one `Execute(context.Context, Request)` interface.
The request contains an optional search query and optional cursor. An absent
search query selects All Bookmarks; a present, non-empty query selects bookmark
search. Keep endpoint selection and shared pagination behavior inside the
operation.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The Bookmarks module exposes one `Execute` interface
returning a normalized `Page`. The CLI rejects an empty `--search` value before
calling the operation. Separate All Bookmarks and bookmark-search methods are
not part of the operation's caller-facing interface.

### Q28: Operation failure behavior

**Question:** What should `Execute` return when request execution or response
normalization fails?

**Recommendation:** Return safe typed failures through the ordinary `error`
result. Preserve already classified X failure codes, use
`RESPONSE_SHAPE_CHANGED` for an unrecognized envelope, and use a generic
`BOOKMARKS_FAILED` for request preparation, transport, or unexpected internal
failures. Never expose raw payloads, authentication material, request URLs, or
internal error text. Let the CLI render the typed failure without knowing the
operation's failure cases.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The operation translates its internal and source
failures into a safe typed error contract. The CLI renders that contract and
does not receive `OperationResult`, inspect source payloads, or classify
operation failures itself.

### Q29: Typed failure ownership

**Question:** Should each operation define its own caller-visible failure type,
or should operations share the repository's existing safe failure contract?

**Recommendation:** Evolve `app.OperationFailure` to implement Go's `error`
interface and use it across Search, Bookmarks, the generic HTTP client, and the
CLI. Let operation modules return `*app.OperationFailure`; let the CLI identify
it with `errors.As` and serialize the existing `code`, `message`, and optional
`recoveryCommand` fields. Map an unexpected non-typed error to a defensive
generic failure at the CLI seam.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The project retains one shared safe failure contract in
`internal/app`. Operation-specific packages do not introduce parallel public
error types. `OperationFailure` implements `error`, remains JSON-serializable,
and crosses the operation-to-CLI seam without exposing source details.

### Q30: Internal page sizes

**Question:** Since v1 has no `--count`, should the Bookmarks operation define
internal page-size constants?

**Initial recommendation:** Mirror the observed UI requests: 20 for the first
All Bookmarks page, 40 for a later All Bookmarks page, and 20 for bookmark
search.

**Matt's answer:** Asked why operation-owned page sizes are needed when the
command is querying the default Bookmarks endpoint.

**Revised recommendation:** Do not give the Bookmarks operation page-size
policy. The captured operation contract already contains X's default request
variables, including `count`, and `httpclient.prepareRequest` preserves those
variables unless the adapter supplies an override. The adapter should override
only semantic caller inputs such as `rawQuery` and `cursor`. The observed 20
and 40 values remain contract evidence rather than application constants.

**Matt's answer to the revised recommendation:** Yes.

**Decision:** Accepted. Page size is captured operation-contract metadata, not
Bookmarks operation policy. The adapter overrides only semantic caller inputs;
it does not define or switch `count` values.

### Q31: Contract activation

**Question:** Should a refreshed contract set be activated if Home and Search
validate but either Bookmarks operation has not been validated?

**Recommendation:** No. Require and validate both `Bookmarks` and
`BookmarkSearchTimeline` before activating a candidate contract set. Validate
bookmark search with a deliberately improbable query so success does not
depend on matching private content.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Both Bookmarks operation contracts join the required
contract set and its activation checks.

### Q32: Internal source interface

**Question:** Should the Bookmarks operation's source interface expose separate
All Bookmarks and bookmark-search methods, or one method?

**Recommendation:** Use one `Fetch(context.Context, Request)` method. Its
production HTTP adapter interprets the optional search query and selects the
source endpoint; its mock adapter can return controlled source results through
the same seam.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The operation's internal source seam models one
Bookmarks request and does not duplicate pagination behavior across separate
methods.

### Q33: Empty-page continuation

**Question:** Q6 says an empty page has a null cursor, but later evidence found
bookmark-search pages containing cursor entries without posts. Should an empty
page retain an active Bottom cursor?

**Recommendation:** Yes. Decode the post collection and continuation
independently. An empty `bookmarks` collection succeeds and may have a non-null
`next_cursor` when X supplies an active continuation.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Q33 refines Q6: an empty collection does not imply the
end of pagination. Q19 governs `next_cursor` independently.

### Q34: Complete page shape

**Question:** Should `warnings` always be present, including on successful
pages without warnings?

**Recommendation:** Yes. Every successful page contains `query`, `bookmarks`,
`next_cursor`, and `warnings`; the latter is an empty array when there are no
warnings.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Successful Bookmarks pages have one stable top-level
shape with no conditionally absent fields.

### Q35: Verification seams

**Question:** What verification should accompany implementation?

**Recommendation:** Verify normalized behavior through the operation interface,
plus narrow CLI parsing and serialization and HTTP-adapter request mapping. Use
only minimized synthetic source fixtures in the repository. Keep authenticated
live verification explicit and opt-in.

**Matt's answer:** Approved, with the requirement that `test/live_x_test.go`
also gain Bookmarks tests and that the live Search tests be completed as part
of the operation restructuring.

**Decision:** Accepted. Default verification uses synthetic evidence and the
agreed interfaces without committing private bookmark captures. Q37 and Q38
later clarify that the opt-in live suite checks authentication state and the
current operation contracts, while deterministic tests own normalization
coverage.

### Q36: Delivery sequence

**Question:** Should the Search restructuring and Bookmarks implementation be
performed as one undifferentiated change?

**Recommendation:** No. First extract shared models and move Search behind its
operation interface; next reduce `httpclient` to shared mechanics; then add the
Bookmarks operation, adapters, contracts, CLI, and documentation.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Implementation proceeds in dependency order so the
shared foundation is independently reviewable and Bookmarks is not built
against a temporary layout.

### Q37: Live-suite purpose

**Question:** Should `test/live_x_test.go` make one representative direct
request for every required operation and require only semantic request success?

**Recommendation:** Yes. Exercise HomeTimeline, SearchTimeline, Bookmarks, and
BookmarkSearchTimeline once each. Success requires no transport error, a 2xx
upstream response containing valid JSON, and no classified X operation failure.
Do not duplicate the deterministic normalization suite here.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The live suite verifies that the current authentication
state and captured operation contracts can execute every required direct
operation. It does not exercise every Search tab or pagination variant.

### Q38: Account-independent live assertions

**Question:** Should live assertions avoid depending on account content?

**Recommendation:** Yes. Do not assert result counts, result kinds,
continuation values, or normalized content. Use an ordinary safe Search query
and a deliberately improbable bookmark-search query. A valid empty response
from a fresh account must pass.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Live success depends on operation execution, not on the
presence of posts or bookmarked posts.

### Q39: Live failure diagnostics

**Question:** When a live operation fails, should the test print the upstream
response body?

**Context:** The current `logLiveFailure` helper can print the first 600
characters of a sanitized upstream response. Sanitization removes known secret
values, but it does not remove private post or bookmark content.

**Recommendation:** Do not print authenticated upstream response bodies. Print
only the operation name, transport error, safe failure code and message, HTTP
status, and content type.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Live-test diagnostics never print an authenticated
upstream response body. Detailed source-payload inspection remains an explicit
evidence-capture task.

### Q40: Bookmark-search operation-contract capture

**Question:** Which module should know how to make X issue the
BookmarkSearchTimeline request during contract refresh?

**Context:** The current browser capture module can open a page and wait for an
operation. Opening the Bookmarks page causes the Bookmarks operation, but
BookmarkSearchTimeline starts only after a user uses the Search Bookmarks
control.

**Recommendation:** Keep that interaction inside the browser capture module.
The contract refresh asks for the BookmarkSearchTimeline operation contract;
it does not know X's controls or selectors.

**Matt's answer:** Approved the recommendation after confirming that Bookmarks
and BookmarkSearchTimeline both start from the Bookmarks page, with bookmark
search adding an interaction with the Search Bookmarks control.

**Decision:** Accepted. The browser capture module owns operation-specific
capture recipes. During one contract refresh it can open the Bookmarks page,
capture Bookmarks, use Search Bookmarks with the validation query, and capture
BookmarkSearchTimeline. Callers request operation contracts without learning
X's controls or selectors.
