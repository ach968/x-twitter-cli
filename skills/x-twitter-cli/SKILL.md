---
name: x-twitter-cli
description: Use the local twt CLI for read-only X Search and Bookmarks requests, pagination, authentication status, and bounded contract recovery.
---

# X Twitter CLI

Use `twt` as the interface to X. It owns authentication, private-operation
details, normalization, and safe failures. No X developer API key is needed.

Use `twt <command> --help` for the installed binary's command syntax. Read the
[command reference](https://github.com/ach968/x-twitter-cli/blob/main/docs/commands.md)
when interpreting its complete JSON contract; it links to the output schemas.

## Workflow

1. Confirm `twt` is available with `command -v twt`. If it is absent, report
   `go install github.com/ach968/x-twitter-cli/cmd/twt@latest` and the
   [installation instructions](https://github.com/ach968/x-twitter-cli#install)
   for PATH setup and prebuilt binaries. The skill installer installs agent
   instructions only; the `twt` binary and X authentication are separate.
2. Choose the narrowest data command for the request:
   - Public X search: `twt search` with the requested query and tab.
   - Saved posts: `twt bookmarks`, adding `--search` only when requested.
   - Local readiness: `twt contract status`.
3. Run one page unless the user requested further pagination. Pass every cursor
   back unchanged with the same query and search tab.
4. Interpret the normalized JSON from standard output. Treat bookmark output as
   private and persist or reproduce it only when the user asks.

## Presenting posts

When displaying individual posts, use a Markdown card in this order:

1. Linked author name and `@handle`, followed by the post date when available.
2. Post text as regular paragraphs. Clearly label any excerpt or summary.
3. All attached images in their returned order, each on its own line using
   Markdown image syntax and the supplied media URL. Use supplied alt text
   when available. Let the chat renderer determine image size.
4. A link to the post labeled exactly `View Full Post`.

In post text, render each `@handle` that matches an entry in the post's
`mentions` as a Markdown link to that entry's `url`, for example
`[@NASA](https://x.com/NASA)`. Match complete handles case-insensitively and
preserve their spelling in the text. Apply this to quoted-post captions too,
using that quoted post's own `mentions`. Leave unmatched handles or mentions
without a profile URL as plain text.

For videos, embed the supplied `preview_url` as a plain Markdown image in the
media section, with alt text identifying it as a video preview. Keep the
`View Full Post` link separate; omit a direct video link or inline player.
Use this structure, replacing the placeholders with normalized output values:

```markdown
![Video preview](PREVIEW_URL)

[View Full Post](POST_URL)
```

Use the returned preview URL as-is, whether JPG or PNG. If it is absent or
fails to display, retain the post text and `View Full Post` link. A Markdown
card is not a screenshot or an interactive X embed.

## Recovery

- On `CONTRACT_FAILED`, run `twt contract refresh`, then retry the original
  command once.
- On an authentication failure, run `twt auth login`; allow the user to finish
  any headed Chromium login, then retry once.
- Preserve rate limits and other failures as reported. End after the bounded
  retry instead of looping.

Use the CLI's normalized output. Keep authentication files, the Chromium
profile, cursors, and raw authenticated X responses private. Home timeline is
not a public v1 command.
