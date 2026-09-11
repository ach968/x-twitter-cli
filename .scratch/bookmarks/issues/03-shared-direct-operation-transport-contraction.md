# 03: Shared direct-operation transport contraction

**What to build:** Complete the Search prefactor by reducing the shared HTTP
client to reusable authenticated request mechanics and moving Search request
knowledge into the Search operation's production adapter.

**Blocked by:** 02: Search operation migration.

**Status:** ready-for-agent

Before starting, read the parent Bookmarks specification, domain glossary,
accepted design interview, and tickets 01-02's completed notes. Treat them as
authoritative.

- [x] The shared HTTP client prepares and executes any captured operation
  contract using its default variables plus explicit semantic overrides.
- [x] Contract lookup, application of authentication state, request encoding,
  transaction IDs, transport, response capture, and source-level failure
  classification remain shared request mechanics.
- [x] Search operation names, variables, and request policy live in the Search
  production adapter rather than the shared HTTP client.
- [x] HomeTimeline contract validation can use the generic direct-operation
  execution path without requiring a Home-specific HTTP-client method.
- [x] Persisted-query rejection continues to produce `CONTRACT_FAILED` with the
  `twt contract refresh` recovery command.
- [x] Authentication rejection, rate limiting, malformed successful responses,
  and unclassified X rejection retain their stable safe failures.
- [x] All production and test callers migrate from the superseded
  operation-specific HTTP-client methods before those methods are removed.
- [x] Temporary Search compatibility delegation from ticket 01 is removed only
  after no caller depends on it.
- [x] Direct Search execution and the user-facing Search command remain green
  throughout the contraction.
- [x] Deterministic repository checks remain green.

## Completion notes

`httpclient.Client.Execute` is the generic captured-operation path. It retains
contract lookup, authentication, encoding, transaction IDs, transport, and
source failure classification; the removed operation-specific methods no
longer contain Search policy. Search's production adapter supplies only
`rawQuery`, `product`, and an optional opaque cursor.

Focused HTTP, operation, CLI, workflow, and full repository tests passed on
2026-09-11.
