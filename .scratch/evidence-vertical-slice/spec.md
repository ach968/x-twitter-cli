# Evidence Vertical Slice

Status: resolved

Follow-up evidence supersedes the original Search execution choice: an earlier probe generated 19/19 transaction IDs accepted by direct `SearchTimeline` requests across five fresh processes, and the current Go implementation subsequently passed the live smoke. `SearchTimeline` is now a direct operation with a frontend-derived transaction ID; the browser executor remains available as fallback infrastructure.

## Problem Statement

The project needs to read X without exposing X's private request formats to callers. Authentication state, operation contracts, volatile request metadata, and X response shapes can all drift independently. Building the polished CLI wrapper before observing a working end-to-end path would force public interfaces and output promises to be designed around guesses.

The immediate need is a runnable evidence slice that proves the risky path with a deliberately small operation set. It must show that the application can establish and reuse authentication state, execute direct reads while retaining a browser-backed fallback, distinguish actionable failure classes where evidence permits, and preserve sanitized source payloads for the next design round.

## Solution

Build a minimal vertical slice around two representative direct operations: `HomeTimeline` and `SearchTimeline`.

The slice uses Rod's pinned managed Chromium revision and an isolated application profile. A headed browser establishes authentication state when human interaction is required. Browser integration tests, contract capture, and any browser-backed fallback reuse the same profile headlessly.

A reusable browser-operation executor owns profile locking, headless Chromium launch, navigation, matching an operation's network response, response capture, and cleanup. It remains available for operations whose volatile metadata or transport shape cannot be reproduced reliably. `HomeTimeline` executes through direct HTTP using a captured contract that preserves either query parameters or a complete JSON body template. `SearchTimeline` also executes through direct HTTP and obtains a fresh transaction ID from the external Go generator initialized from X's current frontend. Results remain source payloads rather than normalized records.

Provide only the minimal development harness needed to run and verify this slice. Use the resulting captures and observed failures as evidence for later output, pagination, recovery, and public CLI design.

## User Stories

1. As a local user, I want installing the Go CLI to avoid downloading Chromium, so that binary installation has no hidden large download.
2. As a local user, I want an explicit setup or first-authentication action to detect whether the required Chromium revision is present, so that browser requirements are clear before login begins.
3. As a local user, I want to approve the Chromium download before it occurs, so that setup does not modify my system unexpectedly.
4. As a local user, I want refusal of the Chromium download to end cleanly with an actionable installation instruction, so that I can install it later.
5. As a local user with an older Rod-managed Chromium revision, I want setup to describe the action as an update and show the installed and required revisions, so that I understand why another download is needed.
6. As a local user, I want authentication to open a headed Chromium window, so that I can complete X login and interactive challenges myself.
7. As a local user, I want the application to keep X authentication in its own persistent profile, so that it does not read from or modify my everyday browser profile.
8. As a local user, I want the application profile stored outside the repository with user-only access, so that authentication state is not committed or exposed to other local users.
9. As a local user, I want a later browser launch to reuse the same application profile, so that I do not have to log in for every contract refresh.
10. As a local user, I want browser workflows serialized with a filesystem lock, so that concurrent processes cannot corrupt the application profile or active contract properties.
11. As a developer, I want the initial authentication workflow to capture operation contracts after login succeeds, so that the first direct read has current contract properties.
12. As a developer, I want an explicit contract-refresh workflow, so that stale contracts can be replaced without coupling refresh to an ordinary data command.
13. As a local user, I want contract refresh to run headlessly when authentication remains valid, so that routine recovery does not require unnecessary interaction.
14. As a local user, I want contract refresh to open a headed browser when X requires login or an interactive challenge, so that recovery can continue safely.
15. As a developer, I want capture to record each operation's complete private request description, so that refresh updates more than a persisted-query identifier.
16. As a developer, I want each captured descriptor to include the operation name, host, path, method, encoding, required variables, feature flags, and field toggles, so that the direct request module can reproduce the observed request.
17. As a developer, I want captured contract properties to exclude cookies, authorization headers, CSRF tokens, and other secrets, so that the generated file is non-secret.
18. As a developer, I want authentication state and operation contracts stored separately, so that expiration or recovery of one is not confused with the other.
19. As a developer, I want contract properties represented as versioned JSON, so that nested request configuration is preserved and future format compatibility can be checked.
20. As a developer, I want the request module to receive a contract source through its interface, so that it does not hardcode live X contract values or a single filesystem location.
21. As a developer, I want missing, unreadable, unsupported-version, and structurally invalid contract properties rejected before any X request is sent, so that local configuration failures are deterministic.
22. As a developer, I want a newly captured file treated as a candidate, so that unverified data never becomes active merely because capture completed.
23. As a developer, I want candidate validation to confirm operation identity, an X-owned host, method, request shape, and a successful representative response for every required operation, so that the active set is internally complete.
24. As a local user, I want a failed candidate validation to leave the active contract properties untouched, so that refresh cannot destroy the current configuration.
25. As a local user, I want a valid candidate activated atomically, so that readers never observe a partially written contract set.
26. As a local user, I want candidate and active files to avoid retaining a rollback history, so that stale contract copies do not accumulate without a useful recovery purpose.
27. As a developer, I want direct execution of `HomeTimeline`, including JSON POST contracts, so that it does not launch Chromium unnecessarily.
28. As a developer, I want direct execution of `SearchTimeline` with a freshly generated transaction ID, so that the operation does not launch Chromium unnecessarily.
29. As a local user, I want each ordinary read to load persisted state, execute once, return its result, and exit, so that the slice does not require a background process.
30. As a local user, I want any operation that requires the browser fallback to launch Chromium headlessly and close it when the command finishes, so that it remains a non-interactive one-shot command.
31. As an agent caller, I want a contract failure to return an actionable recovery command, so that I can explicitly run contract refresh.
32. As an agent caller, I want the original data operation to avoid automatically refreshing or retrying itself, so that recovery remains visible and bounded.
33. As an agent caller, I want authentication failures, contract failures, rate limits, changed response shapes, and ambiguous upstream rejections kept distinct when the evidence supports that classification, so that I do not take the wrong recovery action.
34. As an agent caller, I want uncertain failures classified conservatively, so that an unexplained response such as an empty `403` is not mislabeled as a contract failure.
35. As an agent caller, I want every failed run to produce a nonzero exit status, so that failure cannot be mistaken for success.
36. As an agent caller, I want failed runs to return a small stable JSON error wrapper with a machine code and concise message, so that recovery logic does not depend on unstable X payload schemas.
37. As a developer, I want the sanitized upstream status, content type, and body preserved on failure, so that previously unseen X behavior remains diagnosable.
38. As a local user, I want secrets redacted and large upstream bodies bounded before they are emitted or saved, so that diagnostics do not leak credentials or grow without limit.
39. As a developer, I want successful X data preserved as source payloads, so that later normalization decisions are based on observed shapes rather than assumptions.
40. As a developer, I want representative successes and failures saved as sanitized evidence, so that they can become deterministic fixtures and inform the next design round.
41. As a contributor, I want deterministic tests to run without an X account, so that the default test suite is repeatable and safe for CI.
42. As a contributor, I want real Chromium exercised through Rod against a local test page, so that profile persistence and request capture are tested without contacting X.
43. As a maintainer, I want real-X checks to be explicitly opt-in and use my local application profile, so that credentials are never required in CI.
44. As a maintainer, I want an end-to-end live smoke run before the project proceeds to the polished wrapper, so that downstream design rests on a proven vertical slice.

## Implementation Decisions

- The deliverable is an evidence vertical slice, not the finished public CLI. A minimal development harness may expose setup, authentication, contract refresh, and the two data operations only to make the slice runnable.
- Implement the slice in Go using Rod and its pinned managed Chromium revision.
- Browser installation occurs only after explicit consent during setup or first authentication. Binary installation must not prompt or download a browser. Setup must distinguish a first installation from an update required because only older Rod-managed revisions are present, and update messaging must identify the installed and required revisions.
- V1 browser discovery is intentionally narrow: use only Rod-managed Chromium. Do not detect, launch, or automate Brave, Chrome, arbitrary system Chromium builds, or a user's default browser. `TWT_CHROMIUM_EXECUTABLE` is an explicit development/testing override.
- Launch the isolated Chromium profile with the tested `AutomationControlled` compatibility flag for interactive authentication and browser-backed reads. Do not add a fixed remote-debugging port or wildcard remote-origin permission. Keep automated browser tests headless.
- Use one application-owned persistent Chromium profile for one X identity. Store it in the platform-appropriate user configuration or state area, outside the repository, with user-only permissions.
- Serialize browser workflows and state replacement using filesystem locking. Browser-backed reads remain one-shot processes with no daemon, IPC socket, start or stop command, or idle-timeout lifecycle.
- Treat authentication state and operation contracts as independent inputs with separate persistence and failure handling.
- Derive the direct request client's cookie jar from authentication state held by the application profile. Copied cURL or cookie import is not the primary workflow for this slice.
- Capture requests passively from X's web application during authenticated browser navigation. Capture the complete descriptor required to reproduce each operation rather than extracting only a query ID.
- Support exactly two operations in this slice: `HomeTimeline` and `SearchTimeline`.
- Store contract properties as versioned JSON. The format must support operation identity, host, path, method, encoding, required static variables, features, and field toggles without containing authentication secrets.
- The request module requires a contract source through its interface. The surrounding harness resolves that source in the accepted order: explicit option, `TWT_CONTRACT_FILE`, then the user-local active file.
- Validate local readability, format version, and structure before network activity.
- Treat every generated contract set as a candidate. Validate the complete required operation set against operation identity, X-owned host rules, method, request shape, and expected successful responses before activation.
- Activate a valid candidate by atomic replacement. Failed validation leaves the active file untouched. Do not retain historical files or implement rollback.
- Authentication should capture and activate the initial contract set. Explicit contract refresh should reuse valid authentication headlessly and switch to a headed browser only when login or an interactive challenge is required.
- Data operations never refresh contracts implicitly or retry themselves after refresh. A browser-backed fallback launches Chromium only when it is the selected execution method, not as recovery from a failed direct request.
- The operation interface returns the X source payload without shared `Tweet`, `User`, timeline, or pagination normalization.
- Define a small stable failure interface now: nonzero process status plus a JSON wrapper containing a conservative machine code, concise message, optional exact recovery command, and optional sanitized upstream response.
- Preserve upstream HTTP status, content type, and bounded body on failure. Exclude request cookies and authorization headers, redact detected secrets, and classify only failure modes supported by strong evidence.
- Use a generic upstream-rejected failure for ambiguous responses. A `403` or other status alone is insufficient evidence of authentication or contract failure.
- Preserve sanitized representative source payloads and failures from live execution as explicit design evidence. Evidence containing authentication state or secrets must never be written.
- Completing this slice unlocks the next design round; it does not silently settle output normalization, pagination, or every future failure code.

## Testing Decisions

- A good test crosses a defined module interface and asserts externally visible behavior. Tests should verify returned results, emitted failures, network requests, active state, and preserved evidence rather than private helper calls or internal implementation order.
- The primary deterministic seam is the operation interface. Direct-operation tests inject controlled upstream responses; browser-backed tests use real headless Chromium against a controlled local page.
- At the request-module seam, test both GraphQL operations, including method, path, encoding, variables, features, headers derived from authentication state, response preservation, and conservative failure classification.
- Test contract-property loading through its public interface for missing files, unreadable files, unsupported versions, malformed JSON, incomplete descriptors, unexpected operations, invalid hosts, invalid methods, and invalid request shapes.
- Test candidate validation and activation as externally observable state transitions: a complete valid candidate replaces the active file atomically, while any invalid or unsuccessful candidate leaves the active file byte-for-byte unchanged.
- Test sanitization through its output: secrets are absent, safe upstream fields remain present, and oversized bodies are bounded deterministically.
- Test the agent-facing failure interface for nonzero exit behavior, stable machine codes, concise messages, exact recovery commands where applicable, and inclusion of sanitized upstream responses.
- The browser-workflow seam uses real Chromium through Rod against a controlled local page. Verify that state persists between headless launches, page-generated request metadata reaches the operation, matching network responses are returned, and concurrent profile use is rejected by locking.
- Local browser tests must not depend on X, a personal account, or pre-existing browser state. X-host validation and observed successful-response rules should be tested deterministically with fixtures at the contract interface.
- The live acceptance seam is an explicit opt-in, fully headless run using the already-authenticated local application profile. It must prove direct execution of both `HomeTimeline` and `SearchTimeline`, including runtime transaction-ID generation for Search. Interactive authentication is verified manually rather than inside automated tests.
- Live checks must skip unless explicitly enabled. They must never run in ordinary CI, print credentials, require a committed cookie or cURL file, or mutate X account data.
- Sanitize every live result before promoting it to a fixture. Review fixtures for cookies, authorization values, CSRF tokens, account-identifying data, and unbounded content.
- Prior art exists in `twitter-web-client`: its request-contract tests assert constructed `HomeTimeline` and `SearchTimeline` requests; its cURL parser tests exposed the importance of preserving cookie headers; and its live-X tests separate deterministic response validation from opt-in authenticated checks. Reuse those behavioral lessons without carrying forward hardcoded live contracts or copied-cURL authentication as the new architecture.
- The slice is accepted only when the deterministic suite passes and one opt-in live smoke run produces sanitized representative successes and failures for the evidence set.

## Out of Scope

- The polished public CLI command hierarchy, help text, shell completion, installation documentation, and final binary release workflow
- The Codex skill that invokes the eventual CLI and coordinates manual contract recovery
- Stable normalized `Tweet`, `User`, timeline, search, or pagination records
- Pagination defaults, traversal behavior, truncation rules, and output compatibility guarantees
- Operations beyond `HomeTimeline` and `SearchTimeline`, including tweet detail, threads, profiles, user posts, and bookmarks
- Posting, liking, replying, following, deleting, bookmarking, or any other X mutation
- Automatic contract refresh, automatic retry after refresh, or implicit browser launch from a failed data operation
- Using the user's everyday browser profile, copying cookies from a running browser, or controlling an existing browser tab
- Default-browser discovery or support for Brave, Chrome, Firefox, system Chromium, or other browser implementations
- Multiple named X accounts or profile switching
- macOS and Windows state locations, locking behavior, and Chromium installation support
- Community-maintained, bundled, remotely supplied, or hardcoded live operation contracts
- Contract-file history, rollback, or retention of a last-known-good copy after successful replacement
- A complete taxonomy for every upstream X failure before representative failures have been observed
- Live X credentials or live-X execution in CI
- Treating copied cURL import as the primary authentication or contract workflow

## Further Notes

- Authentication state and operation contracts expire independently. A working login does not prove a current contract, and a current contract does not prove valid authentication.
- `HomeTimeline` has previously changed both its persisted-query ID and surrounding variables and feature configuration. Successful capture must therefore prove the whole operation contract.
- A controlled removal test proved that `SearchTimeline` returns `200` normally and an empty `404` when only `x-client-transaction-id` is removed. Follow-up probes proved that the ID can currently be generated reliably outside Chromium, and the live smoke now proves direct execution of both operations.
- The source payloads created by this slice carry no field-level stability promise. Their purpose is to reveal the real shapes and variations needed for the next design decisions.
- The slice is complete now that the Chromium setup and authentication path works with explicit consent, authentication state survives later use, both operations execute directly, deterministic tests pass, and live evidence validates the generated SearchTimeline transaction ID.
