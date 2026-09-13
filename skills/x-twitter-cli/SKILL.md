---
name: x-twitter-cli
description: Use the local twt CLI for read-only X Search and Bookmarks requests, pagination, authentication status, and bounded contract recovery.
---

# X Twitter CLI

Use `twt` as the interface to X. It owns authentication, private-operation
details, normalization, and safe failures. No X developer API key is needed.

Read the [command reference](../../docs/commands.md) before constructing an
unfamiliar command or interpreting its complete JSON contract.

## Workflow

1. Confirm `twt` is available with `command -v twt`. If it is absent, report
   the installation command from the command reference.
2. Choose the narrowest data command for the request:
   - Public X search: `twt search` with the requested query and tab.
   - Saved posts: `twt bookmarks`, adding `--search` only when requested.
   - Local readiness: `twt contract status`.
3. Run one page unless the user requested further pagination. Pass every cursor
   back unchanged with the same query and search tab.
4. Interpret the normalized JSON from standard output. Treat bookmark output as
   private and persist or reproduce it only when the user asks.

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
