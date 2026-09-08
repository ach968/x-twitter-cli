# SearchTimeline normalized JSON contract

Status: accepted design contract, consolidated into the ready-for-agent
SearchTimeline specification.

The exact required fields and nullability are machine-readable in
[`search-timeline.schema.json`](./search-timeline.schema.json).

This document defines the successful JSON output of `twt search`. It is a
stable caller contract; X's timeline instructions, modules, entry types,
feature flags, and raw payload fields are decoder details.

Example values are illustrative. A field's presence in an example does not by
itself claim that every value has appeared in the captured evidence.

## Page

```json
{
  "query": "golang",
  "tab": "top",
  "results": [],
  "next_cursor": null,
  "warnings": []
}
```

| Field | Type | Meaning |
| --- | --- | --- |
| `query` | string | The query supplied by the caller. |
| `tab` | `top`, `latest`, `people`, `media`, or `lists` | The canonical selected search tab. |
| `results` | result[] | Recognized results in X's order. |
| `next_cursor` | string or null | X's active Bottom cursor, unchanged; null means no next page. |
| `warnings` | warning[] | Non-fatal normalization warnings. |

The page has no `schema_version`. A page may contain any recognized result
kind even when that kind is unusual for the selected tab.

## Results

A result is a flat discriminated object. Its `type` is `post`, `user`, or
`list`. Fields belonging to the other variants are absent.

### Post result

```json
{
  "type": "post",
  "id": "2095005406548341158",
  "url": "https://x.com/example/status/2095005406548341158",
  "text": "Read this: https://t.co/example",
  "author": {
    "id": "123",
    "name": "Example",
    "username": "example",
    "url": "https://x.com/example",
    "avatar_url": "https://pbs.twimg.com/profile_images/example.jpg",
    "verification": "premium",
    "protected": false
  },
  "created_at": "2026-09-01T21:26:00Z",
  "language": "en",
  "conversation_id": "2095005406548341158",
  "reply_to": null,
  "metrics": {
    "replies": 12,
    "reposts": 4,
    "quotes": 2,
    "likes": 99,
    "bookmarks": null,
    "views": 1000
  },
  "possibly_sensitive": false,
  "links": [
    {
      "url": "https://example.com/article",
      "display_url": "example.com/article"
    }
  ],
  "mentions": [],
  "hashtags": [],
  "cashtags": [],
  "media": [],
  "quoted_post": null,
  "community_note": null
}
```

`text` is the authoritative note-post text when available, otherwise legacy
full text. It is not truncated. For agent-readable JSON, presentation whitespace
is collapsed to single spaces, HTML character references are decoded, and
straight double quotes are rendered as typographic quotes. Expanded destinations
are available through `links`. The same text cleanup applies to quoted posts,
Community Notes, names, biographies, list descriptions, and media alternative
text.

`reply_to` is null for a non-reply. For a reply it is:

```json
{
  "post_id": "2095000000000000000",
  "user_id": "456",
  "username": "another_user"
}
```

`quoted_post` is either null or another normalized post object. Quoted posts
may themselves contain quoted posts and Community Notes. The decoder guards
against malformed cycles and unreasonable upstream nesting rather than
publishing an arbitrary depth limit.

### User result

```json
{
  "type": "user",
  "id": "113419064",
  "name": "Go",
  "username": "golang",
  "url": "https://x.com/golang",
  "bio": "Go will make you love programming again. We promise.",
  "avatar_url": "https://pbs.twimg.com/profile_images/example.jpg",
  "banner_url": "https://pbs.twimg.com/profile_banners/example",
  "website_url": "https://go.dev/",
  "verification": "organization",
  "identity_verified": null,
  "protected": false,
  "affiliation": {
    "name": "Example Organization",
    "url": "https://x.com/example_org",
    "badge_url": "https://pbs.twimg.com/semantic_core_img/example.png"
  },
  "automated_by": null,
  "parody_commentary_fan": null,
  "professional": {
    "type": "business",
    "categories": ["Science & Technology"]
  },
  "metrics": {
    "followers": 211068,
    "following": 0,
    "posts": 2266,
    "media": 368
  }
}
```

`verification` is `premium`, `organization`, `government`, or null. It
describes the primary blue, gold, or grey checkmark respectively; it does not
claim that X confirmed the person's identity.

`identity_verified` is true or false only when SearchTimeline explicitly
supplies that fact. It is null when the response does not say. The captured
samples do not currently contain this signal.

`parody_commentary_fan` is `parody`, `commentary`, `fan`, or null.
`professional.type` is the normalized upstream professional-account type;
`professional.categories` contains category names. Both `affiliation` and
`professional` are null when not supplied.

### List result

```json
{
  "type": "list",
  "id": "95292832",
  "name": "Gophers",
  "description": "Golang enthusiasts",
  "url": "https://x.com/i/lists/95292832",
  "owner": {
    "id": "200500313",
    "name": "Example Owner",
    "username": "example_owner",
    "url": "https://x.com/example_owner",
    "avatar_url": "https://pbs.twimg.com/profile_images/example.jpg",
    "verification": null,
    "protected": false
  },
  "private": false,
  "banner_url": "https://pbs.twimg.com/media/example.png",
  "metrics": {
    "members": 79,
    "subscribers": 89
  }
}
```

## Shared objects

### User reference

Authors, list owners, automated-account operators, and mentions use the same
shape:

```json
{
  "id": "123",
  "name": "Example",
  "username": "example",
  "url": "https://x.com/example",
  "avatar_url": "https://pbs.twimg.com/profile_images/example.jpg",
  "verification": "premium",
  "protected": false
}
```

Unavailable scalar fields are null. This is especially relevant to mentions,
whose source entity may not contain an avatar or privacy state.

### Link

```json
{
  "url": "https://example.com/article",
  "display_url": "example.com/article"
}
```

`url` prefers the expanded destination and falls back to the shortened URL
only when no expanded value exists. Character offsets are not exposed.

### Media

```json
{
  "id": "123",
  "type": "video",
  "url": "https://video.twimg.com/example.mp4",
  "preview_url": "https://pbs.twimg.com/ext_tw_video_thumb/example.jpg",
  "width": 1280,
  "height": 720,
  "alt_text": null,
  "duration_ms": 42000
}
```

`type` is `image`, `video`, or `animated_gif`. For video and animated GIF
media, `url` is the highest-bitrate MP4 and `preview_url` is the image
thumbnail. Images use their image URL as `url`. Unavailable dimensions,
alternative text, duration, or preview URL are null.

### Community Note

```json
{
  "id": "123",
  "text": "Readers added context to this post.",
  "language": "en",
  "sources": [
    {
      "url": "https://example.com/source",
      "display_url": "example.com/source"
    }
  ],
  "url": "https://x.com/i/communitynotes/n/123"
}
```

Community Note presentation labels, icons, footers, calls to action, and
translation controls are not part of the contract.

### Warning

```json
{
  "code": "UNKNOWN_SEARCH_ENTRY",
  "message": "Skipped an unsupported search entry"
}
```

Warnings contain only a stable code and concise message. They never include a
raw X entry.

## Presence and normalization rules

- Every available identifier is represented as a JSON string, never a JSON
  number. Direct result, media, and Community Note IDs are required for those
  objects to be meaningfully decoded.
- Every timestamp is a UTC RFC 3339 string.
- Fields defined for a result variant remain present where practical.
- An unavailable scalar or object is null; an empty collection is `[]`.
- A missing count is null and an observed zero is `0`.
- Canonical X URLs are included directly rather than left for callers to
  construct.
- Known results retain their relative upstream order and are not deduplicated.
- Presentation-only entries and cursors are not results.
- Unknown entries are skipped with warnings while recognized results remain
  usable.
- A structurally valid page with no results is successful.
- Usable partial output exits successfully; an invalid SearchTimeline envelope
  or a page with nothing meaningful decodable fails through the separate
  command error contract.

## Deliberately outside this contract

- Raw X payloads and a `--raw` option
- Top cursors, cursor direction, timeline instructions, and modules
- Conversation result grouping
- Viewer-control and relationship-perspective fields
- Presentation, experiment, tracking, and client-event metadata
- Client-side result limiting or deduplication
- Repost and unavailable/deleted-result models until representative evidence
  is captured

## Evidence boundary

The current captures directly support the page envelope, all three result
shapes, blue Premium state, business-affiliation and automated-account labels,
professional accounts, post/user/list metrics, links and text entities, media,
quotes, replies, Community Notes, and nullable cursor behavior.

The contract reserves the following useful states, but no current sample
contains a positive instance from which to implement their mappings:

- gold organization and grey government verification
- explicit ID-verification state
- a Parody, Commentary, or Fan value other than `None`
- a private list

Until representative evidence is captured, the decoder must return null for
an unavailable signal rather than infer it from a nearby field. In particular,
`verification.verified_type == "Business"` is not enough to infer organization
verification: captured users carry that value while `verification.verified` is
false.
