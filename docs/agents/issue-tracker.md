# Issue tracker: Local Markdown

Issues and specs for this repo live as Markdown files in `.scratch/`.
The `.scratch/.sandbox/` directory is reserved for disposable experimental
helpers and is not part of the issue tracker.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`
- The spec is `.scratch/<feature-slug>/spec.md`
- Implementation issues are individual files at `.scratch/<feature-slug>/issues/<NN>-<slug>.md`, numbered from `01`
- Triage state is recorded as a `Status:` line near the top of each issue file
- Completed implementation issues use `Status: resolved`; the five triage
  labels apply only while an issue remains open
- Comments and conversation history are appended under a `## Comments` heading

## Publishing an issue

When a skill says to publish to the issue tracker, create the appropriate file under `.scratch/<feature-slug>/`, creating the directory when needed.

## Fetching an issue

When a skill says to fetch the relevant ticket, read the referenced Markdown file. The user will normally provide its path or issue number.

## Wayfinding

A wayfinding effort uses a map with one child file per ticket.

- Map: `.scratch/<effort>/map.md`
- Child ticket: `.scratch/<effort>/issues/<NN>-<slug>.md`
- `Type:` records `research`, `prototype`, `grilling`, or `task`
- `Status:` records `claimed` or `resolved`
- `Blocked by: NN, NN` identifies dependencies
- A ticket becomes available when all listed dependencies are resolved
- To find the frontier, select the first numbered ticket that is open, unblocked, and unclaimed
- Claim a ticket by setting `Status: claimed` before beginning work
- Resolve it by adding an `## Answer`, setting `Status: resolved`, and adding a short decision pointer to the map
