# Transaction ID probe

Throwaway live experiment: determine whether `x-client-transaction-id` values
can be generated outside Chromium and accepted by X for direct HTTP
`SearchTimeline` requests.

The probe reads the existing x-twt contract and authentication state. It never
prints cookie values, authorization values, or full transaction IDs.

Run from this directory:

```sh
bun install
bun run probe
```

Optional controls:

```sh
ATTEMPTS=10 QUERY=typescript bun run probe
```

A successful run requires every request to return HTTP 200 with a GraphQL-shaped
body and every generated ID fingerprint to be unique. This establishes current
repeatability, not long-term stability: X can change the private frontend
algorithm or its source data at any time.
