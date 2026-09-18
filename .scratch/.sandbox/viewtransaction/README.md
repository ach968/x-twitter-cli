# View transaction ID comparison

Run from the repository root:

```sh
go run ./.scratch/.sandbox/viewtransaction
```

An optional argument replaces the example post ID. The probe reads existing
authentication and TweetDetail contracts without changing them. It compares
initial, continuation, and reply-target reads with and without generated
transaction IDs, then repeats without an ID and checks the captured target when
different. At most eight requests are made; unavailable scenarios are skipped.

Output contains only header-presence, HTTP, JSON, normalization, and partial-page
metadata. A successful comparison describes current X behavior, not a permanent
guarantee. The 2026-09-16 comparison passed all eight requests, including all five
requests without the header.
