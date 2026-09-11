# x-twt-cli design interview

This document is the durable foundation decision tree for `x-twt-cli`: a public Go CLI and thin Codex skill that provide a free, local, read-only interface to X. Feature-specific interviews live with their local specifications under `.scratch/`. Recommendations remain proposals until Matt explicitly accepts or revises them.

## Current evidence

The original interview assumed that normal data commands would navigate X and intercept browser responses. Testing the reference `twitter-web-client` changed that premise:

- Authentication state and X operation contracts expire independently. Refreshing one does not repair the other.
- A copied browser request contained valid authentication cookies under cURL's `-b` option, but the reference parser discarded them. This was an import defect, not evidence that browser sessions cannot be replayed.
- `HomeTimeline` changed both its persisted-query ID and its surrounding request contract. The current request includes a query ID, required variables such as `requestContext`, and a feature set; injecting only a new hash is insufficient.
- The browser is the source of authentication state and can also execute data operations headlessly through X's own frontend. Direct requests remain suitable only where replay has been proven reliable.
- Browser-backed data commands remain one-shot processes: launch headless Chromium with the application profile, capture the matching source response, close Chromium, and exit. They do not require a reusable background process.
- V1 does not discover or support arbitrary system browsers. It uses Rod's pinned managed Chromium revision, installed only after explicit consent during setup or the first authentication command. `TWT_CHROMIUM_EXECUTABLE` is an advanced development override rather than discovery behavior.
- A live headed-login test originally found that Google rejected the default automated browser. Launching the isolated application profile with Chromium's `AutomationControlled` Blink feature disabled allowed Google-to-X login to complete while preserving profile reuse and request capture. The Go implementation preserves this compatibility flag and covers it with a headless browser test; fixed debugging ports and wildcard remote origins are not used.
- A controlled removal test proved that `SearchTimeline` succeeds with a valid `x-client-transaction-id` and returns an empty `404` when only that header is removed. An earlier prototype generated 19 independently accepted IDs across five fresh processes. The independently versioned [`x-client-transaction-id-go`](https://github.com/ach968/x-client-transaction-id-go) module now derives its inputs from X's current frontend and passed the authenticated live Search smoke over direct HTTP.
- A later capture observed `HomeTimeline` change from a query-string `GET` to a JSON `POST`. The contract model now preserves the full JSON body template. The authenticated live smoke passes with both HomeTimeline and SearchTimeline over direct HTTP; Chromium remains necessary for authentication and contract capture.

## Status

- Q1-Q18: resolved foundation

## Resolved foundation

### Q1: Product and initial audience

**Question:** Should v1 be a public-quality Go CLI initially optimized for one local user, one X account, and Linux?

**Recommendation:** Yes. Preserve room for named accounts and other platforms without making them v1 requirements.

**Matt's answer:** Yes; publish it as a Go CLI.

**Decision:** Accepted.

### Q2: MVP operations and safety boundary

**Question:** Should v1 be strictly read-only and cover home timeline, search, tweet/thread retrieval, user profiles, user posts, and bookmarks?

**Recommendation:** Yes. Defer posting, liking, replying, following, deleting, and other mutations until they are separately designed and explicitly approved.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q3: Data-access architecture

**Question:** Should data operations be fulfilled by direct private requests, or through X's web application in the application profile?

**Recommendation:** Keep the operation interface independent of its execution method. Use direct HTTP where replay is proven reliable and materially cheaper. Generate SearchTimeline's volatile transaction ID from X's current frontend through the separately versioned Go module; retain the reusable browser executor as a fallback for operations whose metadata or transport shape cannot be reproduced reliably.

**Matt's answer:** Initially chose mimic-style direct requests, then revised the decision after the transaction-header removal test: execute affected operations through the browser headlessly and keep that machinery reusable for other operations.

**Decision:** Accepted revision. The current evidence slice uses direct HTTP for both operations, with runtime transaction ID generation for `SearchTimeline`.

### Q4: Authentication experience

**Question:** Should `auth login` open a headed browser for interactive X login and preserve local authentication state, rather than making copied cookies or cURL commands the primary onboarding path?

**Recommendation:** Yes. First ensure the Rod-managed Chromium revision is available, offering an explicit installation or update prompt when needed. Check the isolated application profile headlessly first; launch it headed with the tested `AutomationControlled` compatibility flag only when X redirects to login or an interactive challenge, so federated Google-to-X login can complete without adding a fixed debugging port or wildcard remote origin. After login succeeds, use the authenticated browser workflow to capture the initial operation contracts and generate the user's contract properties file. Keep cookie or cURL import as an advanced migration and recovery path only.

**Matt's answer:** Yes.

**Decision:** Accepted. Authentication state belongs to the application profile defined by Q8.

### Q5: Command and browser lifecycle

**Question:** Should data commands use a reusable background process with an idle timeout, or should each command load persisted state, perform its work, and exit?

**Recommendation:** Use one-shot commands. A browser-backed data command should acquire the application profile lock, launch Chromium headlessly, navigate only as needed to elicit the operation, return the matching source response, close Chromium, and exit. Interactive authentication may launch headed. Help, status, and installation do not launch Chromium.

**Matt's answer:** Use one-shot CLI commands and on-demand browser launches. Remove the previously accepted background-process and idle-timeout design because direct requests no longer require a warm browser.

**Decision:** Accepted revision. V1 has no background runtime, session start/stop commands, IPC socket, or idle-timeout lifecycle. Current data commands execute directly; browser-backed fallbacks launch Chromium headlessly on demand.

### Q6: Output ownership and Codex skill boundary

**Question:** Should the CLI own interpretation of X data while the Codex skill remains thin, even though the initial output shape is not yet normalized?

**Recommendation:** Keep the skill thin. Authentication, contract management, browser control, and actionable failure classification belong in the CLI. Until representative X payloads have been captured, return source payloads without promising a normalized field-level contract. The skill should select bounded commands and invoke explicit recovery commands rather than duplicating their implementation.

**Matt's answer:** Keep the skill thin, but do not normalize output until representative X payloads have been captured.

**Decision:** Accepted revision. The thin-skill boundary remains accepted, but normalized output is deferred until representative X payloads exist.

### Q7: Build sequence

**Question:** Should the request and contract modules be proven before building the polished CLI wrapper?

**Recommendation:** Yes, but through a vertical slice rather than by implementing every X operation first. Prove direct HTTP with `HomeTimeline`, then prove transaction-ID generation and direct HTTP with `SearchTimeline`; preserve their source payloads for design evidence; and distinguish authentication, contract, rate-limit, response-shape, navigation, and timeout failures. Then build the wrapper and add operations incrementally.

**Matt's answer:** Yes.

**Decision:** Accepted.

### Q8: Authentication-state ownership

**Question:** Should `x-twt` own a dedicated browser profile, export a smaller storage-state snapshot, or attempt to use the user's everyday browser profile directly?

**Recommendation:** Use one application-owned persistent Chromium profile for the default X identity in v1, stored outside the repository with user-only permissions. Derive the request client's cookie jar from that profile. Do not automate or copy directly from a running everyday profile: profile locking, browser-specific encryption, unrelated cookies, extensions, and concurrent writes make that an unsafe primary interface. Keep named profiles as a later extension.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q9: Operation-contract refresh

**Question:** Should a failed data command refresh contracts automatically, or should contract refresh occur only through an explicit command?

**Recommendation:** Never hardcode or bundle live X contract values in the application. The request module should receive them from a user-local contract properties file generated by `twt auth login`. Never refresh that file implicitly from a data command. A contract-related failure should return a typed, actionable result directing a human or agent to run `twt contract refresh`. That explicit command regenerates the file from complete captured descriptors—operation name, query ID, host, path, method, encoding, required static variables, features, and field toggles. It should refresh headlessly when authentication is valid and open a headed browser when the user must log in. The original data command is rerun explicitly after refresh. Also provide `twt contract status`.

**Matt's answer:** Do not hardcode contract values. Generate a contract properties file through the authentication workflow and inject it into the application. Remove automatic refresh: the agent skill should run the explicit contract-refresh command after a contract failure, refresh headlessly when possible, and launch a browser for login when required.

**Decision:** Accepted revision. Data commands never refresh contracts, retry themselves after refresh, or launch a browser because of a contract failure.

### Q10: Output discovery boundary

**Question:** Should pagination and a stable machine-readable envelope be designed before representative X payloads have been captured?

**Recommendation:** No. Preserve representative source payloads from the first vertical slice before choosing record shapes, pagination metadata, defaults, or compatibility promises. Initial output is explicitly unstable and exists to inform the later design question.

**Matt's answer:** Do not normalize anything until we have seen the payloads returned by X.

**Decision:** Accepted revision. Stable output and pagination remain intentionally unspecified until source payloads are available.

### Q11: CLI, executable, and skill release

**Question:** Should the CLI and Codex skill ship together, and which names should be canonical?

**Recommendation:** Release the Go CLI and skill from one repository and version line so compatibility is explicit. Use repository name `x-twt-cli`, executable `twt`, and skill name `twt-cli`. Provide an explicit skill-install command; binary installation must never modify Codex configuration, prompt interactively, or download Chromium.

**Matt's answer:** Approved the recommendation with executable `twt` and skill name `twt-cli`.

**Decision:** Accepted.

### Q12: Agent-led contract recovery

**Question:** Which layer coordinates recovery after a data command reports that its operation contract failed?

**Recommendation:** The CLI should emit a typed contract failure with the exact recovery command. The `twt-cli` skill should treat that result as a guardrail: run `twt contract refresh`, allow that command to request user login if necessary, and then rerun the original data command once. The skill must not fabricate contract values, reinterpret authentication failures as contract failures, or loop indefinitely.

**Matt's answer:** Put the recovery sequence in the agent skill rather than automatic CLI behavior.

**Decision:** Accepted.

### Q13: Browser selection and installation

**Question:** Should v1 discover and control compatible system browsers, or require a pinned managed Chromium revision?

**Recommendation:** Support only Rod-managed Chromium in v1. Binary installation never prompts for or downloads a browser. `twt setup` checks for the required Chromium revision. When no managed Chromium exists, it asks whether to install Chromium; when an older managed revision exists, it identifies the installed and required revisions and asks whether to update. `twt auth login` performs the same check when setup has not been completed. Acceptance installs the pinned compatible revision; refusal exits with the command the user can run later. Do not support Brave, Chrome, arbitrary system Chromium, or default-browser discovery in v1. Allow `TWT_CHROMIUM_EXECUTABLE` only as an explicit development/testing override.

**Matt's answer:** Require Rod-managed Chromium and offer to install it when setup discovers none, or update it when setup discovers only an older managed revision; avoid the complexity of default-browser detection and multiple browser implementations.

**Decision:** Accepted with one packaging constraint: the interactive prompt belongs to explicit setup or first authentication, never binary installation. After a successful revision update, offer a separate default-no cleanup prompt that warns older `twt` builds may still require the older managed revisions. Declining cleanup keeps the successful update and retains the older revisions.

### Q14: Contract-file validation and replacement

**Question:** How should the application prevent a bad capture from replacing the active contract properties file?

**Recommendation:** Store the generated contract properties file outside the repository and separately from authentication state. Treat each generated file as a candidate: validate every operation's identity, X-owned host, method, request shape, and expected successful response before activation, then replace the active file atomically. If validation fails, leave the active file untouched. Do not retain previous files or expose rollback behavior, and do not accept community-maintained or remotely supplied contract files in v1.

**Matt's answer:** Approved the recommendation, without retaining a last-known-good local contract for rollback.

**Decision:** Accepted.

### Q15: Normalization timing

**Question:** Should v1 define shared `Tweet`, `User`, and pagination records before observing representative X payloads?

**Recommendation:** No. Capture and inspect representative payloads from the vertical slice first. Do not introduce shared normalized records, field mappings, or stability guarantees until the actual response shapes and their variations are understood.

**Matt's answer:** Do not normalize at the moment; revisit the decision after seeing X's payloads.

**Decision:** Accepted. Normalization is a downstream design branch, not part of the initial contract and request work.

### Q16: Contract-file interface

**Question:** What format and lookup rules should let the generated contract properties file supply all live X contract values to the request module?

**Recommendation:** Use versioned JSON because operation descriptors contain nested variables, feature flags, and field toggles that do not fit a flat `.properties` format cleanly. The request module should require a contract source rather than know a filesystem path. The CLI supplies a generated default file from the user's platform config directory, while tests and advanced use can inject another path explicitly. Resolve paths in this order: command option, `TWT_CONTRACT_FILE`, default user-local path. Reject missing, unreadable, unsupported-version, or structurally invalid files before sending a request. Never place authentication state in this file.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q17: Agent-facing failures

**Question:** Which parts of a failed command must remain stable before source-payload schemas are organized?

**Recommendation:** A failed command must return a nonzero exit status and a small stable JSON wrapper containing a conservative machine code, concise message, and optional exact recovery command. Include the sanitized upstream response even when its meaning is unknown: preserve its HTTP status, content type, and body, while excluding request cookies and authorization headers, redacting secrets, and bounding large bodies. Classify only high-confidence failures; use a generic upstream-rejected code for ambiguous responses such as an empty 403. Successful and unsuccessful X payload schemas remain otherwise unspecified until representative responses have been collected.

Example:

```json
{"error":{"code":"UPSTREAM_REJECTED","message":"X rejected the request, but the cause could not be determined"},"upstream":{"status":403,"contentType":null,"body":""}}
```

**Matt's answer:** Keep the stable error wrapper and include the upstream response on failure; organize the actual response schemas later.

**Decision:** Accepted.

### Q18: Testing boundary

**Question:** Which test layers should run automatically, and which may depend on a real X account?

**Recommendation:** Require deterministic tests in CI for contract-file loading and validation, request construction, failure classification, atomic replacement, and the agent-facing error interface. Exercise Chromium through Rod against a local test page for profile persistence and request capture without contacting X. Keep real-X tests explicit and opt-in, using the local application profile; sanitize any payload promoted into a fixture. Require a live smoke run before publishing a release, but never place X credentials in CI or make ordinary contributors authenticate to run the default suite.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.


## Downstream specification work

- Exact contract-file fields, migrations, and compatibility checks
- Exact fixtures, test commands, and release-gate automation
- Diagnostic retention and redaction
- Release channels and binary compatibility policy

## Deferred beyond v1

- Multi-account naming and switching
- macOS and Windows state locations, locking, and Chromium installation
