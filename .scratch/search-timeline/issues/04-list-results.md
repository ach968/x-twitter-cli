# 04: List results

**What to build:** Make Lists searches return normalized list results through the Search command, unwrapping X's list modules into stable identities, ownership, privacy, imagery, and membership metrics.

**Blocked by:** 01: CLI foundation and empty Search pages.

**Status:** resolved

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [x] List objects inside the observed vertical module become flat results with `type` equal to `list` and retain their module order.
- [x] Each list exposes string ID, name, description, canonical X list URL, compact owner reference, private state, banner URL, and metrics using the contract's null rules.
- [x] Owner data uses the same compact user-reference shape as post authors.
- [x] List metrics expose members and subscribers as non-negative integers or null, preserving observed zero.
- [x] Viewer-specific membership, following, muting, pinning, facepile, and presentation-context fields remain outside normalized output.
- [x] The reviewed public-list examples normalize exactly without inferring unobserved private-list behavior.
- [x] Exact golden output protects list ordering, ownership, URLs, banner handling, metrics, and nullability.
- [x] The behavior is demonstrable through `twt search --tab lists` with a controlled requester and remains independent of live X.
- [x] Deterministic repository checks remain green.
