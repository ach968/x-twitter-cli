# SearchTimeline CLI and Normalization

Status: ready-for-agent

## Problem Statement

The project can execute X's private `SearchTimeline` operation and preserve its source payload, but callers still receive no stable, user-meaningful search interface. X returns five search tabs through a volatile timeline grammar containing instructions, entries, modules, cursors, prompts, posts, users, lists, nested quoted posts, media, and Community Notes. Exposing that source shape would force people and coding agents to understand private X implementation details that can drift independently of the product concepts they care about.

The executable is also only an evidence harness today. Adding Search as a one-off directly in the executable entrypoint would make every later operation repeat command routing, dependency construction, stream handling, JSON encoding, failure rendering, and exit-status policy. The project needs a complete but deliberately small CLI application foundation that keeps each data command thin and makes later read operations straightforward to add.

## Solution

Build a reusable CLI application shell and ship the first polished data command, `twt search`.

The CLI application shell owns command routing, help and usage, argument validation, dependency wiring, output streams, JSON serialization, failure rendering, and exit statuses. The executable entrypoint only supplies operating-system dependencies, invokes the application, and exits. The shell supports adding future commands without requiring placeholder commands or speculative models for operations that have not yet produced evidence.

The Search command accepts one query plus optional tab and cursor arguments. It uses the existing request client to obtain a `SearchTimeline` source payload, then passes that payload through one deep Search module. That module hides X's timeline grammar behind a single normalization interface and returns the stable search page described by the accepted JSON contract. Successful pages contain the query, canonical tab, ordered typed results, next cursor, and warnings. Results are flat post, user, or list variants; X's instructions, modules, prompts, presentation metadata, and directional cursor mechanics remain internal.

## User Stories

1. As a local user, I want to run `twt search` with a query, so that I can search X without opening the website.
2. As a coding agent, I want Search to return one stable JSON document, so that I can consume it without scraping terminal text.
3. As a coding agent, I want the response to echo my query, so that a saved page remains self-describing.
4. As a caller, I want Search to default to the Top tab, so that the common search requires no tab option.
5. As a caller, I want to select Top, Latest, People, Media, or Lists, so that I can request each search experience X exposes.
6. As a caller, I want tab names accepted case-insensitively, so that capitalization does not cause unnecessary failures.
7. As a coding agent, I want the returned tab written in canonical lowercase, so that I do not need to normalize it myself.
8. As a caller, I want to pass an opaque cursor with the same query and tab, so that I can fetch another page without hidden CLI state.
9. As a coding agent, I want `next_cursor` always present, so that pagination completion is represented predictably by null.
10. As a caller, I want the next cursor preserved exactly from X, so that the CLI does not invent an incompatible pagination token.
11. As a caller, I want a complete X-supplied page, so that local truncation cannot skip unseen results before the next cursor.
12. As a coding agent, I want results retained in X's order, so that Top ranking and Latest chronology are not changed locally.
13. As a coding agent, I want every result identified by a flat `type` discriminator, so that post, user, and list variants are easy to branch on.
14. As a caller, I want recognized result types preserved even when they appear in an unexpected tab, so that useful data is not discarded because of an assumption about X.
15. As a coding agent, I want all identifiers represented as strings, so that large X identifiers survive JavaScript and JSON tooling unchanged.
16. As a coding agent, I want timestamps normalized to UTC RFC 3339, so that I do not need to parse X's legacy timestamp format.
17. As a coding agent, I want unavailable scalar values represented as null, so that unknown values are not confused with empty or false values.
18. As a coding agent, I want empty collections represented as empty arrays, so that collection fields have a stable type.
19. As a coding agent, I want a missing count represented as null and a measured zero as zero, so that I can distinguish absence from a real value.
20. As a caller, I want posts to include canonical X URLs, so that I can open or cite them without constructing URLs.
21. As a caller, I want posts to contain their authoritative full text, so that long-form posts are not silently reduced to legacy text.
22. As a caller, I want post text preserved rather than rewritten, so that normalization does not alter authored content.
23. As a coding agent, I want expanded link destinations alongside unchanged post text, so that I can follow links without resolving `t.co` myself.
24. As a coding agent, I want structured mentions, hashtags, and cashtags, so that I can reason about post entities without parsing text.
25. As a caller, I want posts to identify their authors with compact user references, so that essential account context is available without embedding full profiles.
26. As a caller, I want post creation time, language, conversation identity, and sensitivity state, so that important post context is preserved.
27. As a caller, I want replies to identify the replied-to post, user, and username, so that reply relationships remain usable without exposing conversation modules.
28. As a caller, I want post reply, repost, quote, like, bookmark, and view counts grouped consistently, so that engagement is easy to inspect.
29. As a coding agent, I want unavailable engagement counts to remain null, so that absence is not reported as zero activity.
30. As a caller, I want images, videos, and animated GIFs normalized to stable media types, so that I do not depend on X's internal names.
31. As a caller, I want media URLs, preview images, dimensions, alternative text, and duration when available, so that media can be inspected or downloaded meaningfully.
32. As a caller, I want video playback to use the highest-bitrate available MP4, so that one useful playback URL is available without interpreting variants.
33. As a caller, I want quoted posts normalized recursively, so that quoted context is available without another request.
34. As a caller, I want Community Notes attached to the post they contextualize, so that note text and supporting sources are not lost.
35. As a coding agent, I want Community Notes to expose stable content rather than presentation labels and actions, so that UI changes do not break my workflow.
36. As a caller, I want People results to include identity, biography, profile links, privacy, verification signals, and useful counts, so that the result is informative without mirroring the entire source object.
37. As a caller, I want Premium, organization, and government verification kept semantically distinct, so that a generic verified flag does not imply the wrong kind of trust.
38. As a caller, I want affiliation, ID-verification, automated-account, Parody/Commentary/Fan, and professional-account signals represented separately, so that distinct profile claims are not collapsed together.
39. As a coding agent, I want absent account signals represented as unknown rather than false, so that the CLI does not invent negative claims.
40. As a caller, I want People metrics to include followers, following, posts, and media posts, so that essential account scale is available.
41. As a caller, I want Lists results to include identity, description, owner, privacy, banner, members, and subscribers, so that lists are directly useful.
42. As a coding agent, I want authors, list owners, automated-account operators, and mentions to share one compact user-reference shape, so that embedded account identities are consistent.
43. As a caller, I want X presentation prompts and timeline containers omitted, so that only search results appear in the stable output.
44. As a caller, I want conversation modules unwrapped into ordinary posts in their returned order, so that the output matches the visible search timeline rather than X's delivery structure.
45. As a coding agent, I want a valid empty search to succeed with an empty result array, so that no matches is not mistaken for a failure.
46. As a coding agent, I want recognized results preserved when another entry is unknown, so that one upstream addition does not destroy the whole page.
47. As a coding agent, I want skipped unknown entries reported through stable warnings, so that partial data loss is visible.
48. As a coding agent, I want usable partial pages to exit successfully, so that warnings do not make valid results inaccessible.
49. As a coding agent, I want invalid envelopes or pages with nothing meaningfully decodable to fail, so that malformed upstream data is not reported as a successful empty search.
50. As a coding agent, I want failures to use a concise stable JSON error shape and nonzero exit status, so that recovery logic does not parse prose.
51. As a developer, I want raw source payloads excluded from ordinary Search output, so that private X formats do not become a public compatibility promise.
52. As a developer, I want the Search decoder independent of transport, so that captured fixtures exercise normalization without X access.
53. As a developer, I want one normalization interface to hide all SearchTimeline grammar, so that callers do not learn instruction, entry, module, or cursor variants.
54. As a developer, I want the request client to remain responsible only for obtaining source payloads, so that transport and interpretation change independently.
55. As a developer, I want the CLI application shell independently testable with arguments, streams, and controlled dependencies, so that command behavior does not require subprocesses or live X access.
56. As a developer, I want the executable entrypoint to contain no command policy, so that future operations reuse the same application foundation.
57. As a developer, I want future commands added through the application shell without creating placeholder commands today, so that extensibility does not become speculative framework code.
58. As a contributor, I want synthetic reviewed fixtures committed under tests while real captures remain private, so that deterministic tests contain no personal or authentication data.
59. As a contributor, I want exact golden JSON coverage, so that field names, nulls, arrays, ordering, timestamps, identifiers, and variants cannot drift silently.
60. As a maintainer, I want live Search checks to remain explicit and opt-in, so that ordinary CI never requires an X account.

## Implementation Decisions

- Build two deep modules at two levels: the CLI application shell is the externally visible command seam, and the focused Search module is the internal normalization seam. The CLI shell owns cross-command policy; the Search module owns all SearchTimeline interpretation.
- The executable entrypoint only constructs operating-system dependencies, invokes the CLI application shell, and exits with its returned status. It contains no Search-specific parsing, transport, normalization, or rendering policy.
- The CLI application interface accepts a context, argument list, input/output/error streams, and its required dependencies, and returns a process status. Tests and future commands use the same interface as the executable.
- The CLI application shell owns command routing, help and usage, argument validation, dependency wiring, JSON serialization, stable failure rendering, and exit-status selection. The choice of command-parsing library remains an implementation detail hidden behind this interface.
- Implement only the Search data command in this spec. The shell must make another command straightforward to register, but must not contain placeholder commands, generic plugin machinery, or speculative models for future operations.
- The Search command requires exactly one non-empty positional query. It accepts optional `--tab` and `--cursor` flags. The tab defaults to Top, accepts case-insensitive input, and normalizes to `top`, `latest`, `people`, `media`, or `lists`.
- The Search command loads the existing authentication state and operation contract through the project's established state and contract modules, invokes the existing request client once, and performs no implicit login, contract refresh, browser launch, or retry.
- The existing request client remains responsible for preparing and executing `SearchTimeline` and returning either a source payload or its established operation failure. It does not import normalized Search types.
- The Search module exposes one primary normalization interface, `DecodePage`, which accepts a source payload, original query, and canonical tab and returns either a normalized search page or an error. Timeline grammar helpers remain private to the module.
- A successful search emits the normalized page itself as one JSON document on standard output. Help emits human-readable usage and exits successfully. Argument, state, transport, and normalization failures emit the stable machine-readable error document on standard error and return nonzero.
- The successful page always contains `query`, `tab`, `results`, `next_cursor`, and `warnings`. It has no schema-version field.
- `next_cursor` is the exact active upstream Bottom cursor after applying relevant add-or-replace timeline instructions. Top cursors and cursor placement remain internal. A missing continuation is null.
- Results are flat discriminated objects whose `type` is `post`, `user`, or `list`. Variant-specific fields sit beside `type`; fields belonging only to another variant are absent.
- Recognized results retain their relative upstream order. The selected tab is request context rather than a strict result-type constraint, so every recognized result kind is decoded in every tab.
- Do not locally deduplicate results. Do not expose a result limit. Return the complete recognized page supplied by X.
- Every available identifier is a JSON string. Direct result, media, and Community Note IDs are required to decode those objects meaningfully. Never serialize X identifiers as JSON numbers.
- Normalize timestamps to UTC RFC 3339. Preserve fields defined for a variant; use null for unavailable scalar or object values and empty arrays for empty collections. Never convert an unavailable count to zero.
- A post contains its ID, canonical URL, authoritative text, compact author reference, creation time, language, conversation ID, nullable reply context, metrics, possible-sensitivity state, links, mentions, hashtags, cashtags, media, nullable quoted post, and nullable Community Note.
- Prefer note-post text when X supplies it and fall back to legacy full text. Do not truncate or rewrite the text. Expanded link destinations are separate link entities and fall back to shortened URLs only when expansion is unavailable.
- Reply context is null for a non-reply and otherwise contains the replied-to post ID, user ID, and username. Do not add a redundant reply boolean.
- Post metrics contain replies, reposts, quotes, likes, bookmarks, and views. Values are non-negative integers or null.
- Normalize media types to image, video, and animated GIF. Media contains ID, type, URL, preview URL, width, height, alternative text, and duration in milliseconds, using null where unavailable. Select the highest-bitrate MP4 for video playback and omit the upstream variants array.
- A quoted post uses the same normalized post shape recursively. The public contract has no fixed quote depth, while the implementation must prevent malformed cycles and unreasonable resource consumption.
- A Community Note contains its ID, text, language, source links, and canonical note URL. Omit icons, labels, footers, calls to action, translation controls, and other presentation state.
- A compact user reference contains ID, name, username, canonical URL, avatar URL, semantic verification, and protected state. Authors, list owners, automated-account operators, and mentions reuse it; unavailable mention fields remain null.
- A full user result adds biography, banner URL, website URL, identity-verification state, affiliation, automation operator, Parody/Commentary/Fan state, professional information, and metrics. It does not mirror viewer-control or relationship-perspective fields.
- Semantic verification is `premium`, `organization`, `government`, or null, corresponding to the visible blue, gold, or grey primary checkmark. It must not be interpreted as identity verification.
- Identity verification is a separate nullable boolean. Affiliation is a separate nullable object containing the available name, URL, and badge URL. Do not invent an organization ID when X supplies only display metadata and a URL.
- Parody/Commentary/Fan state is `parody`, `commentary`, `fan`, or null. Professional information contains its normalized type and category names. Automation identifies its operator through a compact user reference.
- User metrics contain followers, following, posts, and media posts. Exclude the upstream favorite count because it describes posts liked by the user and is easy to misinterpret.
- A list result contains ID, name, description, canonical URL, compact owner reference, private state, banner URL, and metrics. List metrics contain members and subscribers.
- Unknown instructions or entries do not destroy recognized output. Skip an unsupported result-bearing entry and add a warning containing only a stable code and concise message. Recognized presentation-only entries and cursors do not generate warnings.
- A structurally valid page with no results is successful. A page with recognized results and warnings is successful. Fail only when the SearchTimeline envelope is invalid or nothing meaningful can be decoded.
- Keep raw source payloads in private evidence tooling. The Search command has no raw-output option, and warnings never contain raw X entries.
- Reserve useful but not yet positively observed states in the public model. Gold and grey verification, explicit identity verification, non-empty Parody/Commentary/Fan values, and private lists must be populated only from explicit representative source evidence; nearby fields are not substitutes.
- Treat X's timeline instructions, modules, prompts, display types, cursor direction, tracking data, experiment data, and client-event metadata as private implementation details.
- The accepted normalized contract and its machine-readable schema are authoritative for the initial successful output. The existing stable operation-failure contract remains authoritative for failures.

## Testing Decisions

- A good test crosses either the CLI application interface or the Search normalization interface and asserts externally visible behavior. Tests must not assert private helper calls, internal traversal order, concrete command-parser internals, or private source-model types.
- The highest command seam is the CLI application runner. Invoke it with controlled arguments, in-memory streams, and controlled dependencies; assert status, standard output, standard error, and request inputs without launching a subprocess or contacting X.
- CLI tests cover root and Search help, missing or extra queries, empty queries, unknown commands, unknown flags, case-insensitive valid tabs, invalid tabs, default Top behavior, cursor forwarding, stable JSON output, failure JSON, and zero/nonzero statuses.
- CLI tests prove that successful output goes to standard output, failures go to standard error, help is successful, and malformed invocations do not execute the request client.
- CLI tests use a controlled Search requester that returns source payloads or established operation failures. Do not re-test HTTP construction or X response classification through the CLI seam; those behaviors already have focused request-client coverage.
- The primary normalization seam is `DecodePage`. Feed it reviewed synthetic source payloads and compare complete normalized pages with exact golden JSON.
- Golden tests protect the entire agent-facing contract: result discriminators, field names, canonical values, nulls, empty arrays, ordering, identifier strings, normalized timestamps, recursive objects, warnings, and next-cursor behavior.
- Fixture coverage includes Top, Latest, People, Media, Lists, a valid empty page, an initial cursor, a later-page cursor replacement, ordinary posts, note posts, quoted posts, nested quotes, replies, Community Notes, images, videos, animated GIFs, conversation-module unwrapping, prompts, known result kinds in unexpected tabs, and unknown-entry warnings.
- Tests verify authoritative note-post text selection and legacy fallback without using string length as the selection rule or silently truncating content.
- Tests verify that post text retains shortened URLs while link entities expose expanded destinations, and that mentions, hashtags, and cashtags normalize without character offsets.
- Tests verify post, user, and list metrics independently, including observed zero, unavailable null, and rejection or safe handling of invalid negative values.
- Tests verify string preservation for identifiers larger than JavaScript's safe integer range and UTC RFC 3339 conversion from observed X timestamps.
- Tests verify compact user references for authors, owners, automation operators, and mentions, including null fields when a mention supplies only partial account data.
- Tests verify blue Premium state separately from legacy verification fields and prove that missing account signals remain null rather than false.
- Positive mappings for organization verification, government verification, identity verification, Parody/Commentary/Fan labels, and private lists require minimized synthetic fixtures derived from representative captures before those branches are treated as implemented.
- Tests verify business-affiliation and automated-account labels from the observed highlighted-label structures without confusing one for the other.
- Tests verify professional type and categories, null professional state, and exclusion of favorite count and viewer-specific relationship state.
- Tests verify image normalization, highest-bitrate MP4 selection, video preview selection, dimensions, duration, alternative text, and null handling for missing media metadata.
- Tests verify recursive quoted-post normalization and enforce an internal safety guard against malformed cycles or unreasonable nesting without publishing a normal public depth limit.
- Tests verify Community Notes on direct posts and quoted posts, including source links and exclusion of presentation-only note fields.
- Tests verify that `VerticalConversation` modules become ordinary posts in module order and that module metadata does not appear publicly.
- Tests verify add and replace cursor instructions, selection of the active Bottom cursor, and omission of Top cursors from the public model.
- Tests verify that prompts and other recognized presentation-only entries disappear without warnings, while unsupported result-bearing entries produce stable warnings and preserve surrounding results.
- Tests verify successful empty pages, successful partial pages with warnings, invalid-envelope failures, and failure when no meaningful portion can be decoded.
- The machine-readable contract schema and representative composed examples must validate as part of deterministic verification or an equivalent contract test.
- Real captures remain private pending-review evidence. Checked-in fixtures are minimized, synthetic, reviewed for credentials and personal data, and owned by the tests that consume them.
- Existing request-client and transport tests continue to own method, path, variables, transaction-ID generation, authentication headers, status classification, sanitization, and source-payload preservation.
- Live X testing remains explicit, opt-in, outside ordinary CI, and limited to verifying that the current operation still returns a source payload compatible with the decoder. It must not print credentials or commit live content.
- Completion requires formatting, static analysis, deterministic unit and integration tests, race tests, schema validation, and the existing browser-independent repository checks. A live compatibility smoke is required before release, not for every local test run.

## Out of Scope

- Normalized output or polished commands for Home timeline, tweet detail, threads, profiles, user posts, bookmarks, or any operation other than Search
- Shared repository-wide Post, User, List, Media, or pagination models before another operation demonstrates a genuinely shared contract
- Placeholder commands or speculative command interfaces for future operations
- Posting, liking, replying, following, deleting, bookmarking, subscribing, or any other X mutation
- A raw Search output mode or any public exposure of private X timeline grammar
- Search result limiting, automatic traversal of every page, hidden pagination state, cursor files, or cursor input through standard input
- Client-side sorting or deduplication
- A public conversation result or preservation of `VerticalConversation` grouping metadata
- Repost models and unavailable or deleted-result models until representative payloads are captured
- Guessing positive gold, grey, identity-verification, Parody/Commentary/Fan, or private-list mappings without representative source evidence
- Automatic authentication, automatic contract refresh, automatic retries after refresh, or browser launch caused by a failed Search command
- Changes to authentication-state ownership, contract refresh, transaction-ID generation, direct transport, or browser fallback policy
- Multiple named X accounts, account switching, or cross-platform state and browser support
- A public output schema version before a real compatibility or migration mechanism requires one
- Shell completion, packaging, release channels, and binary compatibility policy beyond what is needed to keep the CLI application shell extensible

## Further Notes

- The five search tabs share one SearchTimeline envelope but not one source result shape. Tab is request context; result type is the normalized entity kind.
- The current captures directly support all five tab containers, posts, users, lists, blue Premium state, business affiliations, automated-account labels, professional data, metrics, links, media, replies, quotes, Community Notes, and cursor replacement.
- The source field named `verification.verified_type` cannot independently establish semantic organization verification. Captured users may contain the value `Business` while the accompanying verified flag is false.
- Explicit identity-verification data has not appeared in SearchTimeline evidence. The normalized field remains null unless a future response supplies an explicit signal.
- The accepted contract deliberately favors predictable fields and explicit nulls over omitting data opportunistically. Variant-specific fields remain absent only when they belong to another result type.
- The CLI application shell is infrastructure for future commands, but its value comes from centralizing real cross-command policy. It should not become a registry framework whose complexity merely moves command code elsewhere.
- The machine-readable contract is a development and test artifact; the successful runtime output intentionally contains no schema-version field.
