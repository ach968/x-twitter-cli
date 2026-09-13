# 02: Top and Latest post results

**What to build:** Make Top and Latest searches return useful normalized post results through the complete Search command, covering ordinary post content, authorship, relationships, entities, and engagement while keeping optional rich content empty until later slices.

**Blocked by:** 01: CLI foundation and empty Search pages.

**Status:** resolved

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [x] Direct post entries from Top and Latest pages become flat results with `type` equal to `post` and retain their upstream order.
- [x] Each post exposes its string ID, canonical X URL, compact author reference, UTC RFC 3339 creation time, language, conversation ID, possible-sensitivity state, and stable null or empty values for unavailable fields.
- [x] Author references contain the complete compact user-reference shape and use null for unavailable scalar values.
- [x] Authoritative note-post text is preferred when present; legacy full text is the fallback; selection never depends on which string is longer.
- [x] Returned text is neither truncated nor rewritten.
- [x] Links expose expanded destinations and display URLs while the original text retains any shortened URL.
- [x] Mentions, hashtags, and cashtags normalize into structured collections without character offsets or duplicate media entities.
- [x] Reply posts expose nullable reply context containing the replied-to post ID, user ID, and username; non-replies return null without a redundant reply flag.
- [x] Post metrics expose replies, reposts, quotes, likes, bookmarks, and views as non-negative integers or null, preserving observed zero.
- [x] Media, quoted posts, and Community Notes use their stable empty or null representations pending their dedicated tickets.
- [x] Exact golden output covers representative Top and Latest source fixtures, large string IDs, long-form text, missing values, and real zero counts.
- [x] The behavior is demonstrable through `twt search` with a controlled requester and remains independent of live X.
- [x] Deterministic repository checks remain green.
