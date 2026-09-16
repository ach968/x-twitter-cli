# X Read Client

A local interface to X that exposes stable read operations to people and coding agents without making X's private request formats part of the caller's language.

## Language

**Data command**:
A CLI command that reads account or public X data. It requires valid authentication state; a direct operation additionally requires a current operation contract.
_Avoid_: Browser command, API command

**Authentication state**:
Locally persisted browser credentials established through interactive X login and loaded by later data commands. It changes independently from operation contracts.
_Avoid_: Auth headers, copied cURL, session

**Application profile**:
The isolated browser profile owned by `x-twitter-cli` for one X identity. It is distinct from the user's everyday browser profile.
_Avoid_: Default profile, main profile, user profile

**Operation**:
A stable, user-meaningful X read such as home timeline, search, or tweet detail, independent of the private request currently used to fulfill it.
_Avoid_: Endpoint, GraphQL query

**Search tab**:
One of the result selections offered by X search: Top, Latest, People, Media, or Lists. A search tab determines the kind or ordering of results requested from the search operation.
_Avoid_: Search mode, product

**Search page**:
The stable normalized output of one search request, containing the query, selected search tab, ordered results, continuation value, and non-fatal warnings.
_Avoid_: SearchTimeline response, timeline instructions, API payload

**Search result**:
A normalized post, user, or list returned on a search page. Its kind is independent of the selected search tab.
_Avoid_: Timeline entry, module, conversation result

**Requested post**:
The post identified by the caller's link or post ID. It is the focus of conversation retrieval and may itself be a reply.
_Avoid_: Root post, first result

**Parent chain**:
The sequence of posts reached by following the requested post's reply relationships toward the original post. Replies on other branches are not part of this chain.
_Avoid_: Entire thread, all replies

**Conversation page**:
The available requested post, parent chain, and replies beneath that post returned by one conversation read. Later pages can contain additional replies without repeating the requested post or parent chain.
_Avoid_: Entire conversation, complete thread

**Partial conversation page**:
A conversation page with known gaps in its surrounding content. The existence of another page, or the normal omission of the requested post on a later page, is not such a gap.
_Avoid_: Failed request, complete conversation

**Bookmarked post**:
A normalized post returned because the authenticated identity saved it. Its position reflects X's bookmark order, which is distinct from the post's creation time.
_Avoid_: Bookmark record, saved tweet

**Bookmark page**:
The stable normalized output of one All Bookmarks or bookmark-search request, containing ordered bookmarked posts, continuation state, and non-fatal warnings.
_Avoid_: Bookmarks response, bookmark timeline, API payload

**Bookmark search**:
An X-provided search limited to the authenticated identity's bookmarks. It is distinct from public search and from downloading All Bookmarks for local filtering.
_Avoid_: Search tab, local bookmark filter

**User reference**:
A compact account identity embedded in another search result, such as a post author, list owner, automated-account operator, or mention. It is distinct from a full user search result.
_Avoid_: User summary, embedded profile

**Next cursor**:
An opaque continuation value for requesting the next page of the same operation with the same selection, such as a search query and tab or a requested post. Its internal direction and encoding belong to X's timeline protocol.
_Avoid_: Bottom cursor, page number

**Community note**:
Reader-contributed context displayed with a post after X determines that the note is helpful. It includes the note text and supporting source links and may accompany either a direct result or a quoted post.
_Avoid_: Birdwatch pivot, annotation, warning

**Browser-backed operation**:
An operation fulfilled by navigating X's web application in the headless application profile and returning the matching network response. X's own frontend supplies volatile request metadata such as client transaction IDs.
_Avoid_: Browser command, DOM scrape, direct request

**Direct operation**:
An operation fulfilled without Chromium by reproducing a private X request from authentication state and an operation contract. It may add volatile request metadata derived from X's current frontend.
_Avoid_: Browser-backed operation, raw endpoint

**Transaction ID generator**:
A runtime module that initializes from X's current responsive-web application shell and produces a fresh `x-client-transaction-id` for an HTTP method and operation path. It contains no authentication state and is refreshed independently of operation contracts.
_Avoid_: Static header, persisted transaction ID, browser session

**Operation contract**:
The current private request description needed to execute a direct operation, including its transport shape and any persisted-query identifier or feature configuration. It changes independently from authentication state and is unnecessary for a browser-backed operation.
_Avoid_: Query hash, captured cURL, endpoint

**Contract refresh**:
An explicitly requested act of obtaining and validating a current operation contract from X's web application. It is separate from both the failed data command and authentication recovery.
_Avoid_: Re-authentication, hash rotation

**Contract failure**:
An actionable result indicating that a data command cannot use its current operation contract and requires an explicit contract refresh. It is distinct from an authentication failure, rate limit, or changed response shape.
_Avoid_: Authentication failure, automatic refresh, request failure

**Contract properties**:
The generated, non-secret description of current operation contracts supplied to the application independently of its code and authentication state.
_Avoid_: Hardcoded contracts, contract registry, hash list

**Contract properties file**:
The user-local representation of contract properties generated by authentication or explicit contract refresh and loaded by data commands.
_Avoid_: Packaged snapshot, authentication file, captured traffic

**Source payload**:
Data returned by X before `x-twitter-cli` defines any stable record mapping. It is evidence for later output design and carries no field-level compatibility promise.
_Avoid_: Normalized output, stable record

**Upstream response**:
The HTTP status and response body received from X, preserved in sanitized form when a command fails even if the failure cannot be classified precisely.
_Avoid_: Error classification, successful command
