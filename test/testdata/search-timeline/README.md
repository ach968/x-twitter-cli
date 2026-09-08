# SearchTimeline fixtures

This directory contains reviewed, deterministic SearchTimeline fixtures used by tests.

Live captures are written to the user-local evidence directory and must not be copied here until they have been reviewed for credentials, account-identifying data, personal content, and unnecessary payload bulk. A promoted fixture should be reduced to the smallest source-payload shape that preserves the behavior under test.

The current fixtures were derived from live captures on 2026-09-05 and contain synthetic scalar values:

- `populated-initial.json` preserves added tweet, prompt, module, and cursor entries, including representative media and long-form post shapes.
- `populated-page-2.json` preserves added entries and the replacement instructions used for page-two cursors.
- `empty-initial.json` preserves the successful empty-result shape containing only cursors.
