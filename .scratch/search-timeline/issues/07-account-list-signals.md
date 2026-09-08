# 07: Remaining account and list signals

**What to build:** Complete evidence-backed normalization of the useful verification, identity, account-label, and list-privacy states reserved by the Search contract, while returning unknown for every signal SearchTimeline does not explicitly provide.

**Blocked by:** 03: People results; 04: List results.

**Status:** ready-for-agent

Before starting, read the parent SearchTimeline specification, domain glossary, accepted normalized contract, machine-readable schema, and evidence-handling rules. Treat them as authoritative.

- [ ] Targeted SearchTimeline evidence is collected or identified for gold organization verification, grey government verification, ID verification, Parody/Commentary/Fan labels, and private lists where X exposes them.
- [ ] Every live capture remains private and pending review; only minimized synthetic shapes free of credentials, personal data, and unnecessary payload bulk become fixtures.
- [ ] Each positively observed primary checkmark maps to the correct semantic verification value without using a generic verified boolean.
- [ ] Identity verification is populated only from an explicit SearchTimeline signal and remains null when the response is silent.
- [ ] Parody, Commentary, and Fan values map independently from primary verification, affiliation, automation, and professional state.
- [ ] Private-list state is populated only from an explicit observed list shape and remains unknown when privacy is unavailable.
- [ ] A source value such as business verification type does not imply organization verification when the accompanying verified state is false.
- [ ] For each targeted signal, the evidence record contains either one positive minimized fixture or a dated targeted capture attempt naming the query, tab, expected account or list, and observed absence; an unobserved public field remains null rather than guessed.
- [ ] New positive mappings have minimized deterministic fixtures and exact golden output; unavailable mappings have regression coverage proving null behavior.
- [ ] Existing blue Premium, business-affiliation, automated-account, and professional mappings remain distinct and green.
- [ ] Deterministic repository checks remain green; any live evidence collection remains explicit and outside ordinary CI.
