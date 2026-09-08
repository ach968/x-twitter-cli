# 02: Top and Latest post results

**What to build:** Make Top and Latest searches return useful normalized post results through the complete Search command, covering ordinary post content, authorship, relationships, entities, and engagement while keeping optional rich content empty until later slices.

**Blocked by:** 01: CLI foundation and empty Search pages.

**Status:** ready-for-agent

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [ ] Direct post entries from Top and Latest pages become flat results with `type` equal to `post` and retain their upstream order.
- [ ] Each post exposes its string ID, canonical X URL, compact author reference, UTC RFC 3339 creation time, language, conversation ID, possible-sensitivity state, and stable null or empty values for unavailable fields.
- [ ] Author references contain the complete compact user-reference shape and use null for unavailable scalar values.
- [ ] Authoritative note-post text is preferred when present; legacy full text is the fallback; selection never depends on which string is longer.
- [ ] Returned text is neither truncated nor rewritten.
- [ ] Links expose expanded destinations and display URLs while the original text retains any shortened URL.
- [ ] Mentions, hashtags, and cashtags normalize into structured collections without character offsets or duplicate media entities.
- [ ] Reply posts expose nullable reply context containing the replied-to post ID, user ID, and username; non-replies return null without a redundant reply flag.
- [ ] Post metrics expose replies, reposts, quotes, likes, bookmarks, and views as non-negative integers or null, preserving observed zero.
- [ ] Media, quoted posts, and Community Notes use their stable empty or null representations pending their dedicated tickets.
- [ ] Exact golden output covers representative Top and Latest source fixtures, large string IDs, long-form text, missing values, and real zero counts.
- [ ] The behavior is demonstrable through `twt search` with a controlled requester and remains independent of live X.
- [ ] Deterministic repository checks remain green.
