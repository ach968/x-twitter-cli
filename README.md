# x-twt-cli

This repository contains a local, read-only X client. It proves contract loading and activation, secure authentication-state persistence, conservative response classification, direct-HTTP `HomeTimeline` and `SearchTimeline` reads, and the normalized `twt search` data command.

## Layout

```text
cmd/twt/               executable entrypoint
internal/app/          shared operation, authentication, and result types
internal/app/browser/  Chromium setup, profile lifecycle, capture, and browser transport
internal/app/httpclient/   request preparation, direct HTTP transport, and response classification
internal/app/contracts/ contract validation and loading
internal/app/state/    application paths and persisted authentication/contract state
internal/app/workflow/ cross-module workflows such as candidate activation
test/                  integration tests, evidence tooling, and reviewed testdata
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

## Search command

`twt search` returns one normalized JSON search page on standard output:

```bash
twt search 'golang'
twt search 'golang' --tab people
twt search 'golang' --tab=LATEST --cursor 'opaque-continuation-value'
```

The command accepts exactly one non-empty query. `--tab` is optional and
case-insensitive; its accepted values are `top` (the default), `latest`,
`people`, `media`, and `lists`. `--cursor` is optional and forwards an opaque
continuation unchanged to X. It does not accept a result limit, raw-output
mode, or client-side filtering.

Successful output is a single JSON document with `query`, canonical `tab`,
ordered `results`, nullable `next_cursor`, and `warnings`. Results are flat
`post`, `user`, or `list` objects. Refer to the accepted
[SearchTimeline contract](docs/search-timeline-contract.md) and its
[machine-readable schema](docs/search-timeline.schema.json) for fields and
nullability.

To fetch the next page, pass the previous page's non-null `next_cursor` back
unchanged:

```bash
twt search 'golang' --tab top --cursor 'cursor-from-previous-page'
```

Warnings describe skipped unsupported source entries. A usable page with
warnings still exits zero. Argument, local-state, transport, and incompatible
response failures write the stable JSON error document to standard error and
return nonzero. The command does not automatically log in, refresh contracts,
launch a browser, retry, or deduplicate results.

An agent should treat the output as the product interface and the source
payload as private evidence. The command has no raw-output switch. Capture
and review source payloads only through the explicit evidence workflow below;
never commit pending captures or authentication material.

Run the local browser integration suite with an already-installed Chromium executable:

```bash
TWT_CHROMIUM_EXECUTABLE=/path/to/chromium go test -tags=browser ./...
```

Production browser workflows use the Rod-managed Chromium revision under the user's cache. Browser setup is explicit: `EnsureChromiumAvailable` asks for confirmation before downloading that revision. `TWT_CHROMIUM_EXECUTABLE` is an advanced development override and is not used for browser discovery.

Real-X tests remain explicit and opt-in. They use authentication captured from the application profile under the XDG state directory and must never run in ordinary CI. The live smoke verifies contract capture and direct execution of both operations; it does not probe X's behavior after deliberately corrupting requests.

## SearchTimeline evidence

Capture one successful SearchTimeline source payload for the output-discovery corpus:

```bash
TWT_SEARCH_QUERY='golang' TWT_SEARCH_PRODUCT='Top' make capture-search-evidence
```

The command requires existing authentication state and active operation contracts. `TWT_SEARCH_PRODUCT` accepts `Top`, `Latest`, `People`, `Media`, or `Lists` and defaults to `Top`. It writes a timestamped JSON file under the user-local state directory (`$XDG_STATE_HOME/x-twt/evidence`, or `~/.local/state/x-twt/evidence`) with user-only permissions. Each file is marked `reviewStatus: pending`; it is local evidence, not an approved or commit-ready fixture. Review and minimize a sample before promoting it into `test/testdata/search-timeline/`.

To capture a subsequent page after manually identifying its bottom cursor, pass it explicitly:

```bash
TWT_SEARCH_QUERY='golang' TWT_SEARCH_PRODUCT='Top' TWT_SEARCH_SCENARIO='top-page-2' TWT_SEARCH_CURSOR='...' make capture-search-evidence
```

The recorder preserves the source payload shape while recursively redacting credential-shaped fields and strings. It never writes authentication state, request headers, or the cursor value.

## Short commands

The [Makefile](Makefile) wraps the common development workflows:

```bash
make test                 # deterministic unit tests
make check                # formatting, vet, and race tests
make test-browser         # local Chromium integration tests
make check-browser        # all deterministic checks
make test-live            # full opt-in live-X suite
TWT_SEARCH_QUERY='golang' make capture-search-evidence
make build                # build ./twt
```

Browser targets default to `/usr/bin/chromium`. Override that when needed, for example `make test-live CHROMIUM=/path/to/chromium`. Run `make help` for the complete target list.
