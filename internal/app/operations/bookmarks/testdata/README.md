# Synthetic Bookmarks fixtures

These files are deliberately invented, minimized source envelopes and expected
normalized pages. They contain no authenticated response body, personal query,
identifier, cursor, or authentication state.

`*-source.json` files model the Bookmarks or BookmarkSearchTimeline envelope;
`*-golden.json` files model successful public output. `invalid-envelope-source.json`
is intentionally not a successful page and has no golden file.

The fixtures cover initial and later listing, bookmark search, valid empty
pages (including an empty search page with continuation), partial success with
an unknown result-bearing entry, malformed envelopes, and nested post context.
