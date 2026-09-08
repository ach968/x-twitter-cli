# 08: Pagination, partial success, and contract acceptance

**What to build:** Finish the Search command as a resilient agent-facing feature across every supported tab, including later-page continuation, partial decoding, exact contract conformance, documentation, and release-level compatibility verification.

**Blocked by:** 06: Quoted posts, Community Notes, and conversation modules; 07: Remaining account and list signals.

**Status:** ready-for-agent

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, machine-readable schema, and every blocking ticket's completed notes. Treat them as authoritative.

- [ ] Initial add instructions and later replace instructions select the active Bottom cursor and expose it unchanged as `next_cursor`.
- [ ] Top cursors, cursor direction, entry placement, and replacement mechanics remain outside public output.
- [ ] A valid page with no recognized results succeeds with an empty result array and its active continuation value.
- [ ] Known post, user, and list results are decoded wherever they appear, regardless of the selected tab, while retaining X's relative order.
- [ ] Recognized presentation-only entries and cursors are ignored without warnings.
- [ ] Unsupported result-bearing entries are skipped with a stable code and concise warning while surrounding recognized results remain usable.
- [ ] Usable pages containing warnings exit successfully; an invalid SearchTimeline envelope or a page with nothing meaningfully decodable fails through the stable error contract.
- [ ] No result sorting, deduplication, local limiting, hidden pagination state, raw output, automatic refresh, or automatic retry is introduced.
- [ ] CLI acceptance coverage exercises Top, Latest, People, Media, and Lists; default and case-insensitive tabs; cursor forwarding; success and failure streams; and zero or nonzero statuses.
- [ ] Exact golden pages conform to the accepted machine-readable schema and cover all required result variants, nested objects, nulls, empty arrays, timestamps, identifiers, ordering, warnings, and continuation states.
- [ ] The complete deterministic suite, formatting checks, static analysis, race tests, schema validation, and existing repository checks pass.
- [ ] User-facing documentation describes the Search command, all tab and cursor options, JSON output, pagination flow, warning behavior, evidence boundaries, and representative agent usage.
- [ ] An explicit opt-in live smoke confirms that current SearchTimeline source payloads remain compatible with the finished decoder without printing credentials or committing live content.
- [ ] The finished command remains a thin adapter over the reusable CLI application shell, existing request client, and deep Search module.
