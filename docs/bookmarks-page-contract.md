# Bookmarks normalized JSON contract

The successful output of `twt bookmarks` is a stable, operation-level page.
Its machine-readable definition is
[`bookmarks-page.schema.json`](./bookmarks-page.schema.json). Each member of
`bookmarks` uses the single [shared normalized post contract](./shared-post-contract.md)
defined by [`shared-post.schema.json`](./shared-post.schema.json).

X timeline instructions, modules, entry types, request variables, and cursor
grammar are source details, not caller-facing fields.

## Page

```json
{
  "query": null,
  "bookmarks": [],
  "next_cursor": null,
  "warnings": []
}
```

| Field | Type | Meaning |
| --- | --- | --- |
| `query` | string or null | `null` for All Bookmarks; otherwise the unchanged non-empty `--search` value. |
| `bookmarks` | post[] | Recognized bookmarked posts in X's returned order. |
| `next_cursor` | string or null | X's active Bottom cursor, unchanged; null means no later page. |
| `warnings` | warning[] | Non-fatal normalization warnings. Empty when none occurred. |

Every successful page includes all four fields. An empty `bookmarks` array is a
valid result and does not imply a null `next_cursor`; X can supply a
continuation without a recognized post. The command makes one source request
per invocation and does not sort, deduplicate, interpret, or follow the
cursor.

Bookmarks are posts directly, not wrappers. The page therefore has no bookmark
ID, bookmark timestamp, or redundant `bookmarked` flag. The post discriminator
remains `type: "post"` as defined by the shared contract.

## Warnings and source drift

When an unsupported result-bearing entry appears beside usable bookmarked
posts, the command returns those posts and includes:

```json
{
  "code": "UNKNOWN_BOOKMARK_ENTRY",
  "message": "Skipped an unsupported bookmark entry"
}
```

Presentation-only entries and continuation entries are ignored without a
warning. A malformed Bookmarks or BookmarkSearchTimeline envelope, or a page
with nothing meaningfully decodable, is not an empty page: it fails through the
separate safe command-error contract with `RESPONSE_SHAPE_CHANGED`.

## Deliberately outside this contract

- Raw X payloads and a `--raw` option
- Bookmark Folders, bookmark writes, and account selection
- Page sizes, page numbers, source request grammar, and automatic traversal
- Local search, local fallback, sorting, or deduplication
- A runtime schema-version field
