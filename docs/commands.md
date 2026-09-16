# Command reference

The executable is named `twt`. It writes human-readable help and maintenance
status, while data commands write one normalized JSON document to standard
output. Run `twt --help` or `twt <command> --help` for the installed binary's
concise help.

## Version

```text
twt version
twt --version
```

Prints the installed version. Tagged release binaries report their release tag;
local development builds report `dev`.

## Search

```text
twt search <query> [--tab top|latest|people|media|lists] [--cursor value]
```

Examples:

```bash
twt search 'golang'
twt search '$NVDA'
twt search 'golang' --tab people
twt search 'golang' --tab=latest --cursor 'opaque-continuation-value'
```

Search requires exactly one non-empty query. `--tab` is case-insensitive and
accepts `top` (the default), `latest`, `people`, `media`, or `lists`.
`--cursor` forwards an opaque continuation value to X unchanged. Fetch the
next page by repeating the same query and tab with the returned non-null
`next_cursor`.

Shell expansion happens before `twt` receives a query. In zsh, bash, and
similar shells, use single quotes (`'$NVDA'`) or escape the dollar sign
(`"\$NVDA"`) to pass a literal cashtag.

Successful output contains `query`, canonical `tab`, ordered `results`,
nullable `next_cursor`, and `warnings`. Results are normalized `post`, `user`,
or `list` objects. The complete field contract is documented in the
[Search contract](search-timeline-contract.md) and
[Search JSON Schema](search-timeline.schema.json).

## Bookmarks

```text
twt bookmarks [--search query] [--cursor value]
```

List one page of the authenticated account's bookmarks:

```bash
twt bookmarks
twt bookmarks --cursor 'opaque-continuation-value'
```

Search bookmarks using X's bookmark-search operation:

```bash
twt bookmarks --search 'golang'
twt bookmarks --search 'golang' --cursor 'opaque-continuation-value'
```

`--search` accepts one non-empty query. The query and cursor are passed through
unchanged. The command does not download all bookmarks and filter them locally,
automatically traverse every page, sort, or deduplicate results. Fetch the next
page by repeating the same query, if any, with the returned non-null
`next_cursor`.

Successful output contains the unchanged `query` or `null`, ordered normalized
`bookmarks`, nullable `next_cursor`, and `warnings`. The complete field contract
is documented in the [Bookmarks contract](bookmarks-page-contract.md) and
[Bookmarks JSON Schema](bookmarks-page.schema.json).

Bookmark output is private account data. Redirect or persist it only when that
is intentional.

## View

```text
twt view <url-or-id> [--cursor value]
```

Read a post, its available parent chain, and one page of replies beneath it:

```bash
twt view 'https://x.com/InternetH0F/status/2100261836654309833'
twt view 2100261836654309833 --cursor 'opaque-continuation-value'
```

Pass one positive decimal ID or an HTTP/HTTPS status link on `x.com` or
`twitter.com`, including their `www` and `mobile` hosts. Status paths may use
`/<handle>/status/<id>`, `/i/status/<id>`, or `/i/web/status/<id>`, with an optional
trailing slash or `/photo/<index>` or `/video/<index>` suffix. Sharing query
parameters and fragments are ignored; the post ID identifies the post even if
the handle is outdated. IDs must fit an unsigned 64-bit integer and have no
leading zero. Unrelated URLs, explicit ports, user information, encoded paths,
and empty cursors are rejected before a request.

View returns `post_id`, `post`, `ancestors`, `replies`, `next_cursor`, `partial`,
and `warnings`. Replies preserve X's order. Continue explicitly with the same
post and X's unchanged `next_cursor`. A later page can omit `post` and ancestors;
retain the first page if you need that context. See the
[conversation page contract](view-page-contract.md) and
[JSON Schema](view-page.schema.json) for partial-result and failure semantics.

Existing installations may need `twt contract refresh` before using View.
Search and Bookmarks retain their existing contract requirements.

## Browser setup

```text
twt setup
```

Checks for the required Rod-managed Chromium revision. If it is absent or only
an older revision exists, `twt` describes the installation or update and asks
before downloading anything. After an update, cleanup of older managed
revisions is offered separately and defaults to no.

## Authentication

```text
twt auth login
```

Checks the isolated application profile headlessly and opens managed Chromium
only when X requires interactive login or a challenge. After authentication,
the command captures and validates the Search, Bookmarks, and View contracts before
activating the new local authentication and contract state.

## Contract maintenance

```text
twt contract status
twt contract refresh
```

`status` validates the local authentication and contract files without opening
Chromium or contacting X. It does not prove that X will still accept them.

`refresh` explicitly recaptures and validates current operation contracts. It
uses the existing application profile headlessly when possible and opens
Chromium only if X requires interactive authentication. Data commands never
refresh contracts or retry themselves.

## Output, warnings, and failures

Each successful data command writes one JSON document to standard output.
Usable partial pages may include structured warnings and still exit zero.
Argument, authentication, local-state, transport, rate-limit, stale-contract,
and incompatible-response failures write a stable JSON error document to
standard error and exit nonzero.

When a failure reports `CONTRACT_FAILED`, run `twt contract refresh` and then
retry the original command once. Authentication failures require
`twt auth login`. Preserve other failures as reported instead of retrying in a
loop.

Raw authenticated X responses are private diagnostic evidence and are not
available through a data-command option.

## Work in progress

Home timeline transport code exists internally, but v1 has no public Home
command. Home is not required for setup, authentication, contract refresh,
Search, or Bookmarks.
