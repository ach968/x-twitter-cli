# 06: Quoted posts, Community Notes, and conversation modules

**What to build:** Preserve rich post context through the Search command by recursively normalizing quoted posts, attaching Community Notes to the posts they describe, and translating X's conversation delivery modules into the ordinary post timeline callers see.

**Blocked by:** 05: Media results.

**Status:** ready-for-agent

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [ ] A quoted post becomes a nested normalized post with the same complete shape, including author, metrics, entities, media, reply context, further quotes, and Community Notes.
- [ ] Quote normalization has no published depth limit and preserves every quote returned by a valid source payload.
- [ ] The decoder safely rejects or bounds malformed cyclic or unreasonable nesting without exposing an arbitrary normal-case depth limit.
- [ ] A Community Note attached to a direct or quoted post exposes string ID, text, language, supporting links, and canonical note URL.
- [ ] Community Note labels, icons, footer presentation, calls to action, translation controls, and other interaction state remain outside normalized output.
- [ ] Source links inside Community Notes use the same expanded link shape as post links.
- [ ] Posts inside an observed `VerticalConversation` module become ordinary post results in module order.
- [ ] Conversation-module display metadata, incomplete post-ID metadata, and grouping do not appear in public output.
- [ ] Conversation identity and reply relationships remain on each individual normalized post.
- [ ] Exact golden output covers nested quotes, notes on direct posts, notes on quoted posts, media inside quotes, conversation-module unwrapping, and malformed recursion protection.
- [ ] The behavior is demonstrable through `twt search --tab top` with controlled source fixtures and remains independent of live X.
- [ ] Deterministic repository checks remain green.
