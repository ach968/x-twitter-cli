# Browser automation decision for Q13

**Question.** What is the smallest Go browser-automation stack that can provide an isolated persistent profile, headed authentication, headless contract capture, and an explicitly installed compatible Chromium revision?

**Answer.** Use [`github.com/go-rod/rod`](https://github.com/go-rod/rod) with its launcher-managed Chromium revision. Keep browser discovery intentionally narrow: production workflows use the managed revision, while `TWT_CHROMIUM_EXECUTABLE` provides an explicit development and test override.

This is the implemented Q13 decision. The CLI does not discover a default browser, attach to an existing browser, or automate the user's everyday profile.

## Requirements and mechanisms

| Requirement | Go mechanism | Boundary |
| --- | --- | --- |
| Compatible Chromium | Rod launcher and its pinned revision | The browser is installed only after explicit consent during setup or first authentication. |
| Persistent login | Application-owned user-data directory | Authentication survives later processes without reading or modifying an everyday browser profile. |
| Headed authentication | Rod launcher with headless mode disabled | The user can complete login and interactive challenges directly. |
| Headless reuse | Rod launcher with the same application profile | Contract refresh and browser-backed operations remain non-interactive one-shot processes. |
| Request and response capture | Rod network events | Capture is passive and limited to the operation being observed. |
| Concurrent-use protection | Filesystem lock around the application profile | Two processes cannot launch against or mutate the same profile simultaneously. |
| Development override | `TWT_CHROMIUM_EXECUTABLE` | The override is explicit and does not become production browser discovery. |

## Why the application launches Chromium itself

Opening an X URL in the system default browser is not enough. The application must select its own user-data directory, observe network traffic, wait for a matching operation response, and close the launched process deterministically. Attaching those responsibilities to an arbitrary existing browser would also expose unrelated cookies, extensions, and browsing state.

The application therefore owns both the browser process and its isolated profile. Authentication state lives in that profile; non-secret operation contracts live in a separate versioned JSON file.

## Managed-browser lifecycle

Installing the `twt` binary must not download or launch Chromium. The explicit setup path is responsible for browser availability:

1. Resolve the executable expected by the pinned Rod revision.
2. If it exists, continue without prompting.
3. If no managed revision exists, describe the action as an installation.
4. If only older managed revisions exist, describe the action as an update and report the installed and required revisions.
5. Ask for confirmation before Rod downloads Chromium.
6. If the user declines, exit cleanly and return `twt setup` as the action they can run later.

`twt auth login` performs the same availability check when setup has not already completed. Help, status, installation, and unrelated commands never start a browser.

## Profile and launch policy

- Store one application-owned profile outside the repository with user-only permissions.
- Serialize all uses of that profile with a filesystem lock.
- Launch headed only for authentication or an interactive challenge.
- Launch headlessly for automated browser tests, contract refresh when authentication remains valid, and any operation that selects the browser-backed transport.
- Disable Chromium's `AutomationControlled` Blink feature because the authenticated Google-to-X flow was verified with that compatibility setting.
- Do not expose a fixed debugging port or wildcard remote-origin permission.
- Close Chromium at the end of each workflow; there is no daemon, IPC socket, or idle-timeout lifecycle.

## Alternatives not selected for v1

### Default-browser discovery

Rejected. A desktop default can be Firefox, a custom handler, or a Chromium build that is incompatible with the automation stack. Supporting it would add desktop-entry resolution, platform-specific probing, and multiple browser behaviors without improving the evidence slice.

### Arbitrary system Chromium, Chrome, or Brave

Rejected for production use. A user-installed browser can change independently of the Go module and may not match the revision exercised by the test suite. The executable override is sufficient for deliberate local testing without making those installations part of the supported interface.

### Existing browser or everyday profile

Rejected. Existing processes introduce profile locks and lifecycle ambiguity, while everyday profiles contain unrelated credentials and extensions. The application profile is the only supported authentication-state owner.

## Resulting decision

Q13 is resolved as follows:

- use Rod and its managed Chromium revision;
- download Chromium only with explicit consent during setup or first authentication;
- use one isolated, persistent application profile for headed and headless workflows;
- keep authentication state separate from captured operation contracts;
- keep browser workflows serialized and one-shot; and
- reserve `TWT_CHROMIUM_EXECUTABLE` for development and testing.

This boundary is implemented by the browser setup and profile modules under `internal/app/browser`.
