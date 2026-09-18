# Adding an operation

The code-owned catalog in `internal/app/operation_catalog.go` owns shared
operation lifecycle rules. Catalog membership means an operation is supported
for contract loading and browser capture. Each entry explicitly declares:

- `ContractFile`: whether an existing contract file must contain it.
- `Refresh`: whether login or contract refresh must capture it.
- `Activation`: whether a present contract must pass a validation request.
- `TransactionID`: whether direct requests use the supplied transaction ID generator.

`Required` and `NotRequired` are explicit values; omitted rules fail catalog
tests. Callers receive copies, so they cannot change the catalog. Unknown
operation names are rejected by contract loading and direct execution, and
ignored during browser capture.

These rules do not belong in captured contract files. Those files continue to
hold X's current request details. In particular, older files can omit
TweetDetail, while a fresh login or refresh must capture it. HomeTimeline remains
supported but optional, with no activation request. Only SearchTimeline currently
requires transaction IDs. Shared startup lives in `cmd/twt/dependencies/runtime`; command wiring declares
the operations it needs and supplies its typed operation constructor.

## Implementation ownership

1. Add the operation name in `internal/app/types.go` and an explicit catalog entry.
2. Implement request selection and canonical output in the appropriate module
   under `internal/app/operations/`. One data command may use multiple operations;
   Bookmarks uses both Bookmarks and BookmarkSearchTimeline.
3. For a refresh requirement, connect the navigation or action in the management
   capture plan. Browser actions remain in `internal/app/browser/capture.go`.
   `ContractCaptureStep.CapturedOperations` describes the waits used by that
   same plan, including the actions executed by the browser.
4. For an activation requirement, add its request to `validationOverrides` in
   `internal/app/workflow/activation.go`. Keep semantic response checks in the
   operation module and invoke them on the already-fetched payload. Activation
   replaces active contracts only after every applicable check passes.
5. Use `runtime.New` in production command wiring, supplying the operation names
   needed by that command and a constructor for its typed requester. The runtime
   defers paths, contracts, authentication, transport, and any required generator
   until the first execution. It caches either the requester or its startup
   failure, including across concurrent calls.
6. Wire any new data command through its command parser, production dependency
   wiring, routing, help, and output documentation. The operation catalog does
   not register CLI commands or define their output models.

The browser capture helper also supports smaller diagnostic plans. Its final
minimum matches the contract-file requirements; management separately enforces
all refresh requirements before saving captured state.

## Verification

Run `make check`. The deterministic checks cover explicit policies, refresh
capture-plan coverage, and activation execution for every applicable catalog
entry. Marking an operation required without wiring its capture or validation
request fails these checks. These checks verify wiring; an operation still needs
behavioral tests for its requests and response decoding.

Use `make check-browser` when changing browser execution. View coverage remains
deterministic, with optional live probes under `.scratch/.sandbox/`; do not add
View cases to `test/live_x_test.go` or `test/browser_test.go`.
