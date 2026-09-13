# Bookmarks CLI and Normalization

Status: resolved

## Problem Statement

The X read client cannot currently return the authenticated identity's saved
posts. People and coding agents must use X's web application to read bookmarks,
and they cannot use the client to search those bookmarks through X's own
bookmark-search operation.

X exposes All Bookmarks and bookmark search through separate volatile private
request formats. Those source payloads contain timeline instructions, entries,
cursors, feature configuration, and other details that must not become part of
the caller's language. Bookmark data is also private to the identity in the
application profile, so ordinary output, diagnostics, evidence, and tests must
not persist or reveal source content unexpectedly.

The existing Search command already normalizes the post shape needed by
Bookmarks, but Search currently combines command orchestration, direct request
execution, and normalization across shallow modules. Adding Bookmarks without
first creating a shared post-normalization module would duplicate behavior and
allow Search and Bookmarks to drift into different post contracts.

## Solution

Add a read-only `twt bookmarks` data command that returns one normalized
bookmark page for the identity in the application profile. The command lists
All Bookmarks by default. The caller can select X-provided bookmark search with
`--search`, and can request another page with an opaque `--cursor`.

Each successful bookmark page contains the unchanged search query or null, the
ordered bookmarked posts, the next cursor or null, and non-fatal warnings. A
bookmarked post uses the same shared normalized post contract as a post in a
search page. The command preserves X's order and never downloads All Bookmarks
to perform local search.

Restructure Search and Bookmarks as deep operation modules. Extract the shared
normalized models and the behavior that constructs them. Keep each operation's
source request, envelope, continuation, warning, and page policy local to that
operation. Keep only reusable authenticated request mechanics in the shared
HTTP client.

Extend operation-contract capture and validation to include both Bookmarks
operations. The browser capture module owns the X interactions needed to obtain
their current operation contracts. Extend the explicit live suite so a fresh
account can prove its authentication state and every required operation
contract without depending on account content.

## User Stories

1. As an X user, I want to list my saved posts from the command line, so that I
   can use my bookmarks outside the X web application.
2. As an X user, I want `twt bookmarks` to use the identity in my application
   profile, so that I do not have to select an account for each request.
3. As an X user, I want bookmark reading to remain read-only, so that listing or
   searching cannot change my saved posts.
4. As an X user, I want the command to print results immediately after I invoke
   it, so that interactive confirmation does not block scripts.
5. As a privacy-conscious user, I want ordinary bookmark results to remain
   transient, so that the client does not persist my private saved-post history.
6. As a caller, I want one bookmark page per invocation, so that I control how
   many requests the client makes.
7. As a caller, I want an opaque next cursor, so that I can request another page
   without learning X's continuation format.
8. As a caller, I want the next cursor returned unchanged, so that it remains
   valid for X's operation.
9. As a caller, I want an absent next cursor represented as null, so that the
   end of pagination is explicit.
10. As a caller, I want an empty bookmark collection to succeed, so that a fresh
    account is not treated as an error.
11. As a caller, I want an empty page to retain a next cursor when X supplies
    one, so that an empty collection does not incorrectly end pagination.
12. As a caller, I want X's bookmark order preserved, so that the client does
    not silently reorder my saved posts.
13. As a caller, I want the client to avoid local deduplication, so that the
    normalized page reflects X's returned sequence.
14. As an X user, I want to search my bookmarks with `--search`, so that X can
    find saved posts without the client downloading my full history.
15. As a caller, I want bookmark search to pass one non-empty query to X
    unchanged, so that X's search language retains its meaning.
16. As a caller, I want an empty `--search` value rejected, so that malformed
    requests fail before contacting X.
17. As a caller, I want the same cursor option for listing and search, so that
    pagination has one command-line pattern.
18. As a caller, I want to repeat the same search query with its next cursor, so
    that later search pages retain their original context.
19. As a caller, I want a bookmark-search failure returned directly, so that the
    client does not hide it behind a different local-search behavior.
20. As a caller, I want a bookmark page to contain its query, so that saved JSON
    is self-describing.
21. As a caller, I want All Bookmarks represented with a null query, so that it
    is distinct from bookmark search.
22. As a caller, I want bookmarked posts to use the Search post contract, so
    that one post has the same meaning across operations.
23. As a caller, I want each member of `bookmarks` to be a post directly, so
    that I do not have to unwrap an object with no independent identity.
24. As a caller, I want post identifiers represented as strings, so that large
    X identifiers retain their exact values.
25. As a caller, I want normalized post text, authors, links, media, replies,
    quoted posts, metrics, and Community Notes to follow the shared post
    contract, so that existing Search consumers can reuse their understanding.
26. As a caller, I want every successful bookmark page to contain `query`,
    `bookmarks`, `next_cursor`, and `warnings`, so that fields are never
    conditionally absent.
27. As a caller, I want no-warning success represented by an empty warning
    array, so that I do not need absent-field handling.
28. As a caller, I want supported bookmarked posts preserved when an unknown
    result-bearing entry appears, so that one unfamiliar entry does not discard
    the useful page.
29. As a caller, I want an unknown bookmark entry reported with a stable warning
    code and safe message, so that partial output is observable without exposing
    source content.
30. As a caller, I want malformed source envelopes to fail with
    `RESPONSE_SHAPE_CHANGED`, so that source drift is distinct from an empty
    page.
31. As a caller, I want classified X failures to retain stable codes, so that I
    can distinguish authentication, contract, rate-limit, and other failures.
32. As a caller, I want an operation-contract failure to suggest
    `twt contract refresh`, so that recovery is explicit.
33. As a caller, I want internal request failures represented by a safe generic
    failure, so that private implementation details do not enter command output.
34. As a caller, I want error documents written to standard error with a
    non-zero exit status, so that scripts can distinguish failure from a valid
    empty page.
35. As a caller, I want successful bookmark pages written as normalized JSON to
    standard output with a zero exit status, so that scripts can consume them.
36. As a caller, I want the command to avoid a runtime schema-version field, so
    that output contains no version without a real compatibility mechanism.
37. As a caller, I want no raw-output option, so that volatile private source
    payloads do not become a public command contract.
38. As a caller, I want page size to follow the captured operation contract, so
    that the command does not invent pagination policy.
39. As a caller, I want no `--count` option, so that the first version exposes
    only pagination behavior supported by current evidence.
40. As a maintainer, I want Search and Bookmarks to share normalized post types
    and construction behavior, so that fixes apply to both operations.
41. As a maintainer, I want Search-specific and Bookmarks-specific envelopes to
    remain separate, so that source changes stay local to the affected
    operation.
42. As a maintainer, I want command modules to parse arguments, invoke an
    operation, and render its result, so that they do not interpret source
    payloads.
43. As a maintainer, I want each operation to own its source interface and HTTP
    adapter, so that its source-request knowledge remains local.
44. As a maintainer, I want the shared HTTP client to own only authenticated
    request mechanics, so that operation policy does not accumulate there.
45. As a maintainer, I want one Bookmarks operation interface for listing and
    search, so that callers do not duplicate pagination behavior.
46. As a maintainer, I want one internal Bookmarks source interface, so that
    production and controlled test adapters use the same seam.
47. As a maintainer, I want one safe typed failure contract across operations,
    the HTTP client, and the CLI, so that error JSON does not drift.
48. As a maintainer, I want all required operation contracts validated before
    activation, so that a partial refresh cannot break a data command later.
49. As a maintainer, I want browser capture to own X-specific interaction
    recipes, so that contract refresh callers do not learn volatile controls or
    selectors.
50. As a maintainer, I want one Bookmarks-page visit to capture All Bookmarks
    and then trigger bookmark search, so that contract refresh does not open the
    same page unnecessarily.
51. As a maintainer, I want deterministic tests to use minimized synthetic
    source evidence, so that private bookmark content is not committed.
52. As a maintainer, I want the live suite to execute every required operation,
    so that current authentication state and operation contracts can be checked
    together.
53. As a maintainer, I want live checks to accept valid empty responses, so that
    a fresh X account can run the suite.
54. As a maintainer, I want live diagnostics to exclude authenticated upstream
    response bodies, so that test failures do not reveal private content.
55. As a maintainer, I want shared-model extraction to precede Bookmarks, so
    that the new operation is built on its final normalization seam.

## Implementation Decisions

- The first version is a read-only data command for the identity in the
  application profile. It does not select another account and does not ask for
  confirmation before output.
- The listing form is `twt bookmarks` with an optional opaque `--cursor`.
  Bookmark search adds `--search` with exactly one non-empty string. Options can
  be composed so a later search page receives the same query and its cursor.
- Bookmark search uses X's BookmarkSearchTimeline operation. It never downloads
  All Bookmarks for local filtering and has no local fallback when that
  operation fails.
- One invocation returns one source-supplied page. There is no page number,
  automatic traversal, standard-input request, or caller-selected count.
- Page size remains operation-contract metadata. The Bookmarks operation does
  not define or switch count constants. Its HTTP adapter overrides only
  semantic request inputs such as query and cursor.
- Every successful bookmark page contains four fields: `query`, `bookmarks`,
  `next_cursor`, and `warnings`. All Bookmarks uses a null query; bookmark
  search returns the unchanged query. Collections are always arrays, and
  unavailable scalar or object values are null.
- Members of `bookmarks` are normalized posts directly and retain the `post`
  discriminator. The output does not invent a bookmark identifier, bookmark
  timestamp, wrapper, or redundant bookmarked flag.
- Bookmarked posts use the shared normalized post contract established by
  Search. Dedicated human-readable documentation and JSON Schema define that
  shared contract, and both page contracts reference it rather than duplicate
  it.
- Shared normalized types and all shared source-to-model construction behavior
  form one cohesive models module. The initial design does not create
  speculative model subpackages.
- Search and Bookmarks are separate deep operation modules. Each owns its page
  contract, source envelope, continuation interpretation, warnings, request
  policy, source interface, and production HTTP adapter.
- Each operation exposes a small Execute interface that accepts request context
  and returns either its normalized page or an error. The Bookmarks request
  contains an optional search query and optional cursor. Source-operation
  selection is not part of the caller-facing interface.
- The Bookmarks module has one internal Fetch source interface. Its production
  adapter selects All Bookmarks or bookmark search from the optional query. A
  controlled adapter can supply source results through the same seam.
- The shared HTTP client owns contract lookup, captured default variables,
  application of authentication state, encoding, transaction IDs, transport,
  response capture, and source-level failure classification. It does not own
  Search or Bookmarks request policy.
- The CLI command modules own help, argument validation, exit status, and JSON
  rendering. They do not receive OperationResult, inspect source payloads, or
  classify operation-specific failures.
- The existing shared OperationFailure becomes a Go error while retaining its
  JSON code, message, and optional recovery command. Operations return safe
  typed failures. The CLI recognizes and serializes them; unexpected errors are
  mapped to a defensive generic command failure.
- Already classified X failure codes remain stable. Invalid Bookmarks source
  envelopes become `RESPONSE_SHAPE_CHANGED`. Request preparation, transport,
  and unexpected internal failures become `BOOKMARKS_FAILED`. No failure
  exposes authentication state, request locations, source payloads, or internal
  error text.
- An unsupported result-bearing entry does not discard recognized posts. The
  operation skips it and adds warning code `UNKNOWN_BOOKMARK_ENTRY` with message
  `Skipped an unsupported bookmark entry`. Presentation-only entries and
  continuation entries do not produce this warning.
- Post collection and continuation are decoded independently. A valid empty
  page succeeds and can retain an active next cursor. An invalid envelope or a
  page with nothing meaningfully decodable fails.
- Both Bookmarks operation names join SearchTimeline in the required
  operation-contract set. HomeTimeline remains an optional WIP contract and
  cannot block capture, validation, or activation for the v1 commands.
  Candidate contract properties are activated only after SearchTimeline,
  Bookmarks, and BookmarkSearchTimeline all execute successfully.
- Bookmark-search validation uses a deliberately improbable query, so contract
  activation does not depend on matching private content.
- The browser capture module owns operation-specific recipes. It opens the
  Bookmarks page, captures the Bookmarks operation contract, uses the Search
  Bookmarks control, submits the validation query, and captures the
  BookmarkSearchTimeline operation contract. Contract refresh callers do not
  know X controls or selectors.
- Ordinary command results are not persisted. Raw source payloads remain
  available only through explicit user-local evidence tooling, and the command
  has no raw-output option.
- Implementation proceeds in dependency order: extract shared models and move
  Search behind its operation interface; reduce the HTTP client to shared
  authenticated mechanics; then add the Bookmarks operation, contracts, CLI,
  documentation, and verification.

## Testing Decisions

- Good tests observe behavior through an agreed interface. They assert stable
  normalized pages, safe errors, command output, or prepared source requests.
  They do not call private decoder helpers, reproduce implementation logic in
  assertions, or inspect internal state.
- The primary deterministic seam is each operation's Execute interface. Search
  and Bookmarks tests provide controlled source results and assert normalized
  pages or safe typed failures through that interface.
- Shared model behavior is exercised through Search and Bookmarks operation
  outcomes. Add a narrower shared-model test only when behavior cannot be
  observed clearly through either operation interface.
- Narrow CLI tests cover help, accepted option forms, unknown options, missing
  values, empty search queries, cursor forwarding, JSON output, standard-output
  versus standard-error routing, and exit status.
- Narrow HTTP-adapter tests cover source-operation selection and semantic
  variable overrides. They verify that listing selects Bookmarks, search selects
  BookmarkSearchTimeline, query and cursor values pass unchanged, and page size
  remains supplied by contract properties.
- Contract tests cover the three-operation required set, optional valid
  HomeTimeline properties, candidate rejection when a required operation is
  missing or invalid, and activation only after all three required operations
  succeed.
- Browser capture tests cover operation-recipe selection and the combined
  Bookmarks-page flow without exposing browser selectors through the caller's
  interface.
- Synthetic source fixtures cover populated All Bookmarks, populated bookmark
  search, valid empty pages, empty pages with continuation, later pages, unknown
  entries with usable posts, invalid envelopes, nested normalized posts, and
  classified source failures.
- Synthetic fixtures contain minimized invented values. Private captured posts,
  authors, identifiers, queries, cursors, and authentication state are never
  committed.
- Existing Search command tests, Search normalization tests, HTTP-client tests,
  contract-activation tests, and browser-capture tests are prior art. Tests that
  assert behavior below the newly accepted operation interfaces are replaced
  when they no longer describe a useful seam.
- The explicit live suite makes one representative direct request for
  SearchTimeline, Bookmarks, and BookmarkSearchTimeline. It checks
  for no transport error, a 2xx upstream response containing valid JSON, and no
  classified operation failure.
- The live suite does not assert result count, result kind, continuation,
  normalized content, Search-tab variants, or pagination. A fresh account with
  valid empty responses passes.
- Live-test failure logs can include the operation name, transport error, safe
  failure code and message, HTTP status, and content type. They never print an
  authenticated upstream response body.
- Authenticated live verification remains explicit and opt-in. It is not part
  of the default deterministic suite.

## Out of Scope

- Adding, removing, clearing, or reorganizing bookmarks.
- Bookmark Folders. The current application identity has no Premium
  subscription or folders, so representative operation-contract and source
  evidence is unavailable.
- Multiple-account selection.
- Downloading the complete bookmark history in one invocation.
- Local bookmark search or fallback from X-provided bookmark search.
- Caller-selected page size, page numbers, or a `--count` option.
- Local sorting or deduplication.
- A raw-output option or automatic persistence of bookmark results.
- A bookmark wrapper, bookmark identifier, bookmark timestamp, or locally
  inferred bookmark metadata.
- A runtime schema-version field without a concrete compatibility mechanism.
- Caller-visible timeline instructions, source operation names, feature
  configuration, cursor direction, or other private request grammar.
- Bookmark content assertions, pagination checks, or all Search-tab variants in
  the authenticated live suite.
- Premium Folder design based on guessed source shapes.

## Further Notes

- An authenticated structural inspection established the current Bookmarks and
  BookmarkSearchTimeline operations, request-variable keys, source envelopes,
  ordinary post entries, empty-result behavior, and continuation entries. The
  companion evidence audit records those facts without private content.
- The structural evidence did not retain raw bookmark responses. Implementation
  must use minimized synthetic fixtures and explicit user-local evidence work
  when another source shape must be investigated.
- Deleted, withheld, unavailable, tombstone, repost, and other exceptional
  bookmark result shapes remain unsupported until representative evidence is
  available. Unknown result-bearing entries follow the accepted partial-success
  warning behavior.
- Operation identifiers, feature configuration, field toggles, default request
  variables, and transaction-ID requirements are volatile operation-contract
  evidence. They are captured and validated rather than hardcoded as stable
  product behavior.
- The accepted design interview remains the detailed decision history for this
  specification. The evidence audit remains the source record for the observed
  Bookmarks behavior.
