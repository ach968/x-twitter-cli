# x-twt-cli

This repository contains a Go evidence slice for a local, read-only X client. It currently proves contract loading and activation, secure authentication-state persistence, conservative response classification, and direct-HTTP `HomeTimeline` and `SearchTimeline` reads. The polished CLI is still out of scope.

## Layout

```text
cmd/twt/               executable entrypoint
internal/app/          shared operation, authentication, and result types
internal/app/browser/  Chromium setup, profile lifecycle, capture, and browser transport
internal/app/httpclient/   request preparation, direct HTTP transport, and response classification
internal/app/contracts/ contract validation and loading
internal/app/state/    application paths and persisted authentication/contract state
internal/app/workflow/ cross-module workflows such as candidate activation
test/                  build-tagged browser and live integration tests
docs/                  design history and agent guidance
.scratch/              local specifications and issue-tracker records
```

The root `app` package defines the stable vocabulary and transport boundary. Focused subpackages own policy and infrastructure; build-tagged integration tests live under `test/`. Both operations use the direct HTTP transport. The `SearchTimeline` operation requests a fresh `x-client-transaction-id` from the external [`x-client-transaction-id-go`](https://github.com/ach968/x-client-transaction-id-go) module and adds it before handing the complete request to the generic HTTP transport.

Build the `twt` executable:

```bash
go build ./cmd/twt
```

Run the deterministic unit suite:

```bash
go test ./...
```

Run the local browser integration suite with an already-installed Chromium executable:

```bash
TWT_CHROMIUM_EXECUTABLE=/path/to/chromium go test -tags=browser ./...
```

Production browser workflows use the Rod-managed Chromium revision under the user's cache. Browser setup is explicit: `EnsureChromiumAvailable` asks for confirmation before downloading that revision. `TWT_CHROMIUM_EXECUTABLE` is an advanced development override and is not used for browser discovery.

Real-X tests remain explicit and opt-in. They use authentication captured from the application profile under the XDG state directory and must never run in ordinary CI. The live smoke verifies contract capture and direct execution of both operations; it does not probe X's behavior after deliberately corrupting requests.

## Short commands

The [Makefile](Makefile) wraps the common development workflows:

```bash
make test                 # deterministic unit tests
make check                # formatting, vet, and race tests
make test-browser         # local Chromium integration tests
make check-browser        # all deterministic checks
make test-live            # full opt-in live-X suite
make build                # build ./twt
```

Browser targets default to `/usr/bin/chromium`. Override that when needed, for example `make test-live CHROMIUM=/path/to/chromium`. Run `make help` for the complete target list.
