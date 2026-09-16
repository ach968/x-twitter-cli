# Shared normalized post contract

The normalized post is the stable object returned by any read operation that
returns posts. It is defined machine-readably in
[`shared-post.schema.json`](./shared-post.schema.json). Search pages and
Bookmark and Conversation pages reference this contract; their timeline instructions, cursors,
warnings, and other page policy are operation-specific.

Every post contains `type: "post"`, a string `id`, `url`, `text`, `author`,
`created_at`, `language`, `conversation_id`, `reply_to`, `metrics`,
`possibly_sensitive`, `links`, `mentions`, `hashtags`, `cashtags`, `media`,
`quoted_post`, and `community_note`. Fields documented as unavailable are
represented as `null`; collections are always arrays.

Text is selected from authoritative note-post text when present, otherwise
legacy full text. It is normalized for reader-facing JSON: presentation
whitespace is collapsed, HTML character references are decoded, and straight
double quotes become typographic quotes. The same cleanup applies to author
names, media alternative text, and Community Note text.

`author` and each `mention` are compact user references with `id`, `name`,
`username`, `url`, `avatar_url`, `verification`, and `protected`. `reply_to`
contains `post_id`, `user_id`, and `username`. `metrics` contains `replies`,
`reposts`, `quotes`, `likes`, `bookmarks`, and `views`.

Links expose the expanded destination when available and a nullable display
URL. Media has `id`, `type` (`image`, `video`, or `animated_gif`), `url`,
`preview_url`, dimensions, alternative text, and duration. A quoted post is
another shared post; malformed cycles and unreasonable nesting are suppressed.
A Community Note contains its ID, normalized text, optional language, source
links, and canonical X URL.
