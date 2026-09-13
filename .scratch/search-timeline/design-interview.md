# SearchTimeline design interview

This document contains the SearchTimeline branch of the original `x-twitter-cli`
design interview. It preserves the original Q19–Q64 numbering so references in
the specification and decision history remain stable. Recommendations were
proposals until Matt explicitly accepted or revised them.

## Status

- Q19-Q64: resolved
- Normalized JSON contract: accepted
- `twt search`: implemented
- Remaining evidence questions are recorded below and do not block the resolved
  initial command

## Resolved SearchTimeline decisions

### Q19: Search tabs in scope

**Question:** Which `SearchTimeline` result selections must the first search model cover?

**Recommendation:** Capture and model all five selections exposed by X search—Top, Latest, People, Media, and Lists—using the same query where possible. Treat a shared page model and mode-specific result variants as a hypothesis until the captures show which timeline structures and entities are actually shared.

**Matt's answer:** Cover all five, beginning with `SearchTimeline` only.

**Decision:** Accepted. The evidence set must cover Top, Latest, People, Media, and Lists before the public search model is fixed.

### Q20: Public search result structure

**Question:** Should all five search tabs use one public page structure with typed result variants, while X-specific timeline containers remain internal decoding details?

**Recommendation:** Yes. Return one search page containing the selected tab, typed results, and continuation state. Represent posts, users, and lists as distinct flat result variants, identified by a `type` field whose value is `post`, `user`, or `list`; place the variant's fields alongside `type` rather than inside another variant-named object. Normalize tweet-bearing modules into posts rather than exposing X's presentation containers, and keep prompts out of the public model.

**Matt's answer:** Approved the recommendation. Name the selected result category `Tab`, matching the choice users make in X search.

**Decision:** Accepted and later clarified by Q38. The public search page uses `Tab`, flat post/user/list result variants, and continuation state; X's flat items, modules, instructions, and cursor placement remain decoder concerns.

### Q21: Scope of search entity models

**Question:** Should the initial post, user, and list result models be general X-wide entities or projections owned by `SearchTimeline`?

**Recommendation:** Keep them specific to `SearchTimeline` until another operation demonstrates a genuinely shared contract. Do not make one operation's observed fields into repository-wide `Post`, `User`, or `List` promises prematurely.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The first result projections belong to `SearchTimeline`; shared cross-operation entities may be extracted later from evidence.

### Q22: Initial field surface

**Question:** Should SearchTimeline projections expose a compact set of user-meaningful fields or mirror the full upstream objects?

**Recommendation:** Keep the initial contract small. A search page exposes its tab, typed results, and next cursor. Post results expose identity, text, author, creation time, language, conversation identity, essential engagement counts, sensitivity, and media. User results expose identity, names, biography, profile URLs, verification/privacy state, and essential counts. List results expose identity, name, description, owner, privacy, banner, and membership counts. Media exposes its kind, useful URLs, dimensions, alternative text, and duration when applicable. Exclude presentation, tracking, experiment, and viewer-control fields.

**Matt's answer:** Approved the compact projection; keep it simple for now.

**Decision:** Accepted. Fields are added to the stable SearchTimeline output only when they serve a demonstrated caller need.

### Q23: Pagination cursor

**Question:** Should the public search page expose only a next cursor rather than X's directional timeline cursors?

**Recommendation:** Yes. Translate X's current Bottom cursor into an opaque `NextCursor`, return it unchanged with the same query and tab to request another page, and omit the Top cursor from the initial contract. An absent next cursor means no further page is available.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The public model exposes only `NextCursor`; Top/Bottom placement and replacement instructions remain internal decoder concerns.

### Q24: Agent-facing pagination flow

**Question:** How should an agent request another search page without hidden CLI state?

**Recommendation:** Keep pagination stateless. Return the active upstream Bottom cursor unchanged as `next_cursor`; the caller repeats the same query and tab with `--cursor`. Use `null` when no continuation exists. Do not generate shell commands or persist an implicit current search. Add cursor-file or stdin input later only if copying long cursors proves troublesome.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. `next_cursor` is the exact active Bottom cursor value selected from X's add-or-replace timeline instructions, not a CLI-generated token.

### Q25: Result ordering

**Question:** After removing non-result timeline entries, should the public page preserve X's result order?

**Recommendation:** Yes. Ranking and ordering belong to the selected search tab. Do not re-sort results by timestamp, popularity, or result kind.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Known results retain their relative upstream order.

### Q26: Unknown timeline entries

**Question:** How should decoding respond to an instruction or entry kind that the client does not recognize?

**Recommendation:** Preserve successfully decoded results, skip the unknown entry, and include a small structured warning. Fail only when the shared SearchTimeline envelope is invalid or no meaningful portion can be decoded. Recognized presentation-only entries such as prompts and cursors do not require unknown-entry warnings.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Unknown variants produce visible partial-success warnings rather than silent data loss or needless whole-page failure.

### Q27: Empty result pages

**Question:** Is a structurally valid SearchTimeline page with no recognized results successful?

**Recommendation:** Yes. Return an empty results array and the active next cursor, if any. Empty data is not an error; an invalid envelope remains an error.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q28: Self-describing search pages

**Question:** Should each page echo the original search query?

**Recommendation:** Yes. Pagination requires the same query and tab, so including the query keeps an agent-facing response self-contained.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. A public search page includes `query` and `tab`.

### Q29: Result JSON shape

**Question:** Should each result place its variant-specific fields directly beside a `type` discriminator rather than under another wrapper?

**Recommendation:** Yes. Use a flat discriminated shape such as `{"type":"post","id":"...","text":"..."}`. This is smaller and easier for agents to inspect while retaining typed variants.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. The discriminator name was later clarified as `type`.

### Q30: Stable optional output

**Question:** Should `warnings` and `next_cursor` always be present in a successful page?

**Recommendation:** Yes. Return an empty warnings array when there are none and a null next cursor when pagination is finished.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Successful pages use a predictable JSON shape rather than omitting these fields.

### Q31: Long-form post text

**Question:** When X supplies both legacy `full_text` and `note_tweet` text, what should the public `text` field mean, and should long-form text be bounded for agent consumption?

**Evidence:** Every captured note-post occurrence had different legacy and note text. Note text was generally longer and reached more than ten thousand characters in one capture, but one note value was shorter than its legacy value, so choosing whichever string is longer would not define the right semantics.

**Recommendation:** Define `text` as the authoritative post content. Prefer note text when present and fall back to legacy `full_text`; do not expose upstream note wrappers or silently truncate the model. If agent output later needs a text limit, make truncation explicit and mark affected results.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. A later reader-facing formatting requirement preserves
the authoritative content but collapses presentation whitespace, decodes HTML
character references, and renders straight double quotes typographically. Text
remains untruncated.

### Q32: Quoted posts

**Question:** Should a post include its quoted post as a nested normalized post rather than expose only the quoted post ID?

**Recommendation:** Yes. Use an optional `quoted_post` with the same compact post shape so agents receive the quoted context already present in the response.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q33: Community Notes

**Question:** Should a normalized post include the Community Note returned with it by SearchTimeline?

**Recommendation:** Yes. Add an optional `community_note` containing the note ID, text, language, source URLs, and Community Note URL. Apply the same post model to notes attached to nested quoted posts. Exclude `birdwatch_pivot` labels, icons, footer presentation, calls to action, and translation controls.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Community Notes are post context, not page warnings or timeline presentation metadata.

### Q34: Compact user references

**Question:** Should embedded accounts use a compact user reference rather than the full People-tab result model?

**Recommendation:** Yes. Define one `UserRef` containing `id`, `name`, `username`, canonical `url`, `avatar_url`, semantic `verification`, and `protected`. Reserve biography, counts, affiliation, ID verification, automation, Parody/Commentary/Fan, and professional information for full user results.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted and later expanded by Q59. This is the canonical embedded-user shape.

### Q35: Video URL selection

**Question:** Should the simple media model choose one playback URL from X's MP4 and HLS variants?

**Recommendation:** Yes. Select the highest-bitrate MP4 as `url` and expose the image thumbnail as `preview_url`. Do not expose the upstream variants array initially.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q36: Search-tab CLI behavior

**Question:** What should happen when the caller omits `--tab` or uses different letter casing?

**Recommendation:** Default to Top, accept tab names case-insensitively, and emit canonical lowercase values: `top`, `latest`, `people`, `media`, and `lists`.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q37: Page-size controls

**Question:** Should the initial CLI expose a result-limit option?

**Recommendation:** No. Return the complete page supplied by X. Locally truncating a page while returning X's Bottom cursor could skip results that the caller never received. Add limiting only with a continuation design that cannot lose results.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. V1 has no client-side result limit.

### Q38: Conversation modules

**Question:** Should a Top-tab `VerticalConversation` module remain grouped in the public results, and what should that group be called?

**Evidence:** The captured module contained three post payloads sharing one conversation ID, while its metadata named six total post IDs. The returned posts included reply relationships but were only a selected excerpt of the larger conversation.

**Recommendation:** Do not create a public conversation result. Unwrap the module into ordinary post results in module order, retaining each post's conversation ID. Treat `VerticalConversation` and its deduplication metadata as upstream delivery mechanics unless a future caller demonstrates a need for the grouping.

**Matt's answer:** The actual Top search appears as an ordinary timeline of posts without meaningful conversation clustering, so omit the public conversation concept.

**Decision:** Accepted. Search result kinds are post, user, and list; there is no conversation variant.

### Q39: Identifier representation

**Question:** Should post, user, list, media, and Community Note IDs be JSON strings?

**Recommendation:** Yes. X identifiers can exceed JavaScript's safe integer range, so numeric JSON values could be silently changed by agent tooling.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. All public identifiers are strings.

### Q40: Timestamp representation

**Question:** Should public timestamps be normalized from X's legacy date strings?

**Recommendation:** Yes. Emit UTC RFC 3339 strings such as `2026-09-01T21:26:00Z`.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Public timestamps use UTC RFC 3339.

### Q41: Missing-field representation

**Question:** Should fields belonging to a result variant remain present when their values are unavailable?

**Recommendation:** Where practical, use null for unavailable scalar or object values and empty arrays for empty collections. Do not turn a missing count into zero because zero is a meaningful value. Fields belonging to other result variants remain absent.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Variant fields use null or empty collections for unavailable values; absence and numeric zero remain distinct.

### Q42: Canonical URLs

**Question:** Should normalized post, user, and list results include directly usable X URLs?

**Recommendation:** Yes. Agents should not need to reconstruct URLs from IDs and usernames.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q43: Media-kind names

**Question:** Should public media kinds normalize X's upstream names?

**Recommendation:** Yes. Expose `image`, `video`, and `animated_gif`; do not expose X's internal `photo` name.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q44: Partial-success warning shape

**Question:** What should a warning for a skipped unknown search entry contain?

**Recommendation:** Include only a stable code and concise message, such as `{"code":"UNKNOWN_SEARCH_ENTRY","message":"Skipped an unsupported search entry"}`. Do not include raw upstream content in ordinary output.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q45: Raw search output

**Question:** Should the initial `twt search` command expose a `--raw` payload option?

**Recommendation:** No. Keep private X payloads in separate evidence tooling so ordinary agent output remains stable, bounded, and sanitized.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. V1 search does not expose raw upstream payloads.

### Q46: Search command shape

**Question:** How should the command accept its query, tab, and continuation?

**Recommendation:** Require one positional query and keep `--tab` and `--cursor` optional: `twt search "golang" --tab top --cursor "..."`.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q47: Client-side deduplication

**Question:** Should the normalized decoder remove repeated posts by ID?

**Evidence:** The conversation-module capture contained 22 post payloads and 22 unique IDs; no duplicate result was observed.

**Recommendation:** Do not add unproven client-side deduplication. Preserve the recognized results X returned and collect evidence if duplicates appear later.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q48: Known result kinds in unexpected tabs

**Question:** Should a recognized post, user, or list be retained if it appears in a tab where it was not expected?

**Recommendation:** Yes. Decode every known result kind and preserve its position. A tab describes the request but is not a strict result-type guarantee.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q49: Unobserved result variants

**Question:** Should missing repost and unavailable/deleted-result evidence block implementation?

**Recommendation:** No. Implement observed shapes now; until captured, unsupported variants follow Q26 and produce a partial-success warning.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q50: Partial-success exit status

**Question:** Should a page containing usable results plus warnings exit successfully?

**Recommendation:** Yes. Exit zero for usable partial results. Return nonzero only when the SearchTimeline envelope is invalid or nothing meaningful can be decoded.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q51: Search decoder module

**Question:** Where should SearchTimeline normalization live, and what interface should callers learn?

**Recommendation:** Put models and decoding in a focused `internal/app/search` module with one primary interface: `DecodePage(source any, query string, tab Tab) (Page, error)`. Keep the existing HTTP client responsible for obtaining source payloads so evidence tooling can still inspect them; the CLI passes successful payloads through the decoder.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Upstream timeline grammar remains local to the search module rather than leaking into the CLI or HTTP transport.

### Q52: Reviewed fixture coverage

**Question:** Which synthetic source-payload fixtures must cover the initial decoder?

**Recommendation:** Cover all five tabs, empty results, later-page cursor replacement, note posts, quoted posts, Community Notes, images, videos, conversation-module unwrapping, and unknown-entry warnings. Keep real captures private.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q53: Exact JSON contract tests

**Question:** Should tests compare complete normalized JSON output in addition to targeted behavior?

**Recommendation:** Yes. Use reviewed golden outputs to protect exact agent-facing field names, nulls, arrays, ordering, timestamps, and string IDs, alongside focused behavioral assertions.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q54: Account verification representation

**Question:** How much of X's account-verification state should a normalized user result expose?

**Evidence:** X distinguishes three primary checkmarks: blue for an eligible Premium account, gold for an official organization account, and grey for a government or multilateral organization or official. X also exposes separate identity signals—including affiliation badges, ID verification, automated-account labels, Parody/Commentary/Fan labels, and professional categories—which are not interchangeable with the primary checkmark. The captured People payload likewise contains separate `is_blue_verified` and `verification` fields.

**Recommendation:** Do not collapse these signals into one ambiguous `verified` boolean. Expose the visible primary checkmark as a nullable semantic value: `verification: "premium" | "organization" | "government" | null`. Preserve affiliation, ID-verification, automated-account, Parody/Commentary/Fan, and professional signals separately when SearchTimeline supplies enough evidence to do so.

**Matt's answer:** Approved the semantic verification enum and requested that the other useful account signals also be retained.

**Decision:** Accepted. Exact representations for the additional signals are addressed separately so absence from an upstream payload is not mistaken for a negative value.

### Q55: Expanded links

**Question:** Should normalized post text and entities expose expanded destination URLs rather than X's shortened `t.co` URLs where available?

**Recommendation:** Yes. Prefer the expanded destination so agents can use and reason about the link directly; retain a shortened URL only when no expanded value is available.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q56: Nested quote depth

**Question:** Should quoted posts be recursively normalized without a fixed public depth limit?

**Recommendation:** Yes. Decode the quoted-post chain represented in the response rather than imposing an arbitrary public cutoff. Guard internally against malformed cycles or unreasonable upstream nesting.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q57: Output schema version

**Question:** Should the first normalized SearchTimeline page include a `schema_version` field?

**Recommendation:** No. Keep the initial output small; add an explicit version only when an actual compatibility or migration mechanism requires it.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q58: Additional account signals

**Question:** How should affiliation, ID verification, automated-account, Parody/Commentary/Fan, and professional-account information appear on a user result?

**Evidence:** The captures include `affiliates_highlighted_label` values for business affiliation and automated-account labels, a `parody_commentary_fan_label` field, and professional type and category data. They do not currently contain a distinct ID-verification field. X treats these as separate profile signals rather than alternate meanings of one verification flag.

**Recommendation:** Give each signal its own nullable field rather than creating a heterogeneous labels array. Use a small structured affiliation containing its available name, URL, and badge URL; a nullable `identity_verified` boolean; a small `automated_by` user reference; a nullable `parody_commentary_fan` enum; and a professional object containing its type and category names. Use `null` when SearchTimeline does not supply a signal; do not infer `false` from absence. Do not invent an affiliation organization ID when the payload supplies only a URL and display metadata.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q59: Embedded user references

**Question:** Where should the compact `UserRef` defined by Q34 be reused?

**Recommendation:** Reuse the same `UserRef` for post authors, list owners, automated-account operators, and other embedded account references. This keeps nested results bounded and gives agents one stable account-reference shape.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. Q59 specifies reuse; Q34 owns the exact fields.

### Q60: Links inside post text

**Question:** Should normalization replace X's shortened `t.co` URLs inside returned post text?

**Recommendation:** No. Preserve the post text unchanged and expose usable expanded destinations separately in a `links` array containing `url` and `display_url` when available. This lets agents follow links without silently rewriting authored content.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted. This clarifies Q55: expanded destinations belong to structured link entities rather than text rewriting.

### Q61: Post engagement metrics

**Question:** How should normalized post engagement counts be organized and named?

**Recommendation:** Group them under a `metrics` object using X's current public terminology: `replies`, `reposts`, `quotes`, `likes`, `bookmarks`, and `views`. Apply Q41 within the object: an unavailable count is `null`, while an observed zero is `0`.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q62: User and list metrics

**Question:** Which observed counts should normalized user and list results expose?

**Evidence:** All 20 direct users in the captured People page supplied follower, following, post, and media-post counts. Both direct lists in the captured Lists page supplied member and subscriber counts. User `favorites_count` was also present, but represents posts liked by that user rather than likes received.

**Recommendation:** Use a `metrics` object consistently. User metrics contain `followers`, `following`, `posts`, and `media`; list metrics contain `members` and `subscribers`. Exclude `favorites_count` because it is nonessential and easy to misinterpret. Continue applying Q41 so an unavailable future value is `null`, not zero.

**Matt's answer:** Approved the recommendation based on the captured evidence.

**Decision:** Accepted.

### Q63: Structured post entities

**Question:** Which entities parsed from post text should the normalized post expose?

**Evidence:** The captures contain expanded URL entities, user mentions, hashtags, and cashtags represented upstream as `symbols`; media entities are also present but already belong to the normalized media collection.

**Recommendation:** Expose `links` as objects containing usable destination and display URLs, `mentions` as compact user references, and `hashtags` and `cashtags` as string arrays. Keep media in the separate `media` array. Omit upstream character offsets initially because callers have no demonstrated need for UTF-16 text positions.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

### Q64: Reply relationships

**Question:** How should a normalized post retain conversation and reply relationships without exposing X's conversation modules?

**Evidence:** Captured posts supply a conversation ID and, for replies, the replied-to post ID, user ID, and username.

**Recommendation:** Keep `conversation_id` on the post and expose a nullable `reply_to` object containing `post_id`, `user_id`, and `username`. Use `null` for a non-reply. Do not add an `is_reply` boolean because the presence of `reply_to` already conveys it.

**Matt's answer:** Approved the recommendation.

**Decision:** Accepted.

## Search evidence established

- All five tabs share the `SearchTimeline` timeline-instructions envelope.
- Top and Latest returned post items; Top can wrap some posts in `VerticalConversation` modules and can include presentation prompts, both of which remain internal decoder concerns.
- People returned direct user items.
- Media returned posts inside a vertical-grid module.
- Lists returned list objects inside a vertical module.
- A targeted SearchTimeline capture returned the linked Community Note as a post-level `birdwatch_pivot`, including note identity, text, language, source-link entities, a note destination, and presentation-only labels and actions. The same structure can occur on a nested quoted post.
- Initial-page cursor placement varies by tab, and an observed later Top page used `TimelineReplaceEntry`.
- Raw captures remain private pending-review evidence; checked-in fixtures are synthetic, reviewed representations of the response grammar.

## Remaining evidence questions

- Shapes for reposts and unavailable or deleted results, which have not appeared in the captures.
- Video playback variants and partially missing media metadata.
- Later-page behavior for People, Media, and Lists.
- Positive examples of gold organization and grey government verification, explicit ID verification, Parody/Commentary/Fan labels, and private lists.
- Additional instruction or entry variants that should be ignored, normalized, or rejected.
