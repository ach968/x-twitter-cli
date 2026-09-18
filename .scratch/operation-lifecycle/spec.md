# Central operation lifecycle rules

Status: resolved

## Accepted design

Centralize shared rules in one code-owned operation catalog. Keep browser actions,
validation requests, and canonical response decoding in their owning modules.
Captured contract files describe current X requests and cannot change application
lifecycle policy. Preserve command behavior and old-file compatibility.

Every operation explicitly declares contract-file, refresh, activation, and
transaction-ID requirements. Catalog membership supplies support for contract
loading and capture. New requirements must be checked against actual capture and
activation wiring by deterministic tests. Command startup consolidation is a
separate architecture candidate and is outside this change.

## Implementation

`internal/app/operation_catalog.go` is consumed by contract loading, browser
capture, management refresh, activation, and direct HTTP execution. The browser
uses the same action declarations for execution and capture-plan inspection.
Activation iterates the catalog and fails if a required validation request is
missing. View continues to decode the already-fetched response.

Contributor instructions: `docs/operation-lifecycle.md`.

## Validation

An intentionally unwired required operation was temporarily added: both capture
coverage and activation wiring tests failed with the expected errors. The probe
entry was then removed. `make check-browser` passed: formatting, vet, the
race-enabled deterministic suite, and existing local Chromium integration tests.
No View cases were added to the browser or live-X integration suites.
