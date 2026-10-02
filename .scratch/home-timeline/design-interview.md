# Home timeline design interview

Status: Design in progress; implementation has not started.

## Settled decisions

### Round 1 — 2026-09-19

- Support both For You and Following in this feature.
- `twt home` selects For You; `twt home --following` selects Following.
- Return normalized posts, rather than an experimental raw-response command.

## Open decisions

- Page envelope and pagination behavior.
- Feed context such as repost attribution and recommendation explanations.
- Treatment of promoted posts, conversation groups, and non-post entries.
- Contract capture, activation, and compatibility with existing installations.
- Acceptance checks for both feeds and continuation requests.

Current upstream request and response shapes must be verified before implementation;
historical HomeTimeline transport support is not proof of current behavior or
Following support.
