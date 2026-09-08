# 03: People results

**What to build:** Make People searches return full normalized user results through the Search command, including useful profile identity, links, privacy, observed account signals, professional information, and metrics without leaking viewer-specific state.

**Blocked by:** 01: CLI foundation and empty Search pages.

**Status:** ready-for-agent

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, and machine-readable schema. Treat them as authoritative.

- [ ] Direct People entries become flat results with `type` equal to `user` and retain their upstream order.
- [ ] Each user exposes string ID, name, username, canonical X URL, biography, avatar URL, banner URL, expanded website URL, and protected state using the contract's null rules.
- [ ] User metrics expose followers, following, posts, and media-post counts while excluding the upstream favorite count.
- [ ] Blue Premium state maps to semantic `premium` verification without treating it as identity verification.
- [ ] Observed business-affiliation data exposes available name, URL, and badge URL without inventing an organization ID.
- [ ] Observed automated-account data exposes its operator through the compact user-reference shape.
- [ ] Observed professional data exposes normalized type and category names.
- [ ] Missing identity-verification, Parody/Commentary/Fan, affiliation, automation, professional, and non-blue verification signals remain null rather than false or inferred.
- [ ] Viewer-control, direct-message permission, notification, relationship-perspective, experiment, and presentation fields remain outside normalized output.
- [ ] All twenty reviewed People examples can be represented deterministically, including zero counts and partial optional profile data.
- [ ] Exact golden output protects the complete user result shape and account-signal nullability.
- [ ] The behavior is demonstrable through `twt search --tab people` with a controlled requester and remains independent of live X.
- [ ] Deterministic repository checks remain green.
