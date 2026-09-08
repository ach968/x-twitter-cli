# SearchTimeline reserved-signal evidence audit — 2026-09-08

This record summarizes a read-only review of the pending private evidence under
the local application state directory. It names only the public targeted
queries and excludes captured result identities, list names, IDs, payload
excerpts, and authentication material.

## Captures reviewed

- The 2026-09-04 and 2026-09-05 populated initial/later-page/empty capture
  sets.
- The 2026-09-06 explicit Top, Latest, People, Media, and Lists initial-page
  capture set.
- The 2026-09-06 Community Note capture.

## Reserved signal findings

| Signal | Positive evidence | Observed source state | Normalization decision |
| --- | --- | --- | --- |
| Gold organization verification | No | The targeted People attempt for `@Microsoft` supplied `verification.verified_type: "Business"` with `verification.verified: false`; prior captures had the same non-positive combination. | Keep `verification` null; do not treat `Business` as organization verification. |
| Grey government verification | No | The targeted People attempt for `@POTUS` supplied `verification.verified_type: "Government"` with `verification.verified: false`. | Keep `verification` null; do not treat this source value as a government primary checkmark. |
| Explicit identity verification | No | No `id_verified`, `identity_verified`, `is_identity_verified`, or equivalent explicit identity key occurred. | Keep `identity_verified` null. |
| Parody/Commentary/Fan | Parody and Fan only | Earlier captures supplied `None`; a targeted People attempt for a parody account subsequently supplied `Parody` and `Fan` values. `Commentary` was not observed. | Map only `Parody` and `Fan` to their lowercase public values. Keep `Commentary` and `None` null. |
| Private list | No | The targeted Lists attempt for `private` and the earlier explicit Lists capture supplied only `list.mode: "Public"`. | Map observed `Public` to `private: false`; all other or absent modes remain null. |

## Targeted capture attempts

All attempts below were made on 2026-09-08. The captures remain private; this
record intentionally contains neither raw payload nor personal identifiers.

| Attempt | Expected signal | Outcome |
| --- | --- | --- |
| People `@Microsoft` | Organization verification | `Business` with `verified: false`; no positive organization mapping. |
| People `@POTUS` | Government verification | `Government` with `verified: false`; no positive government mapping. |
| People `ID verified` | Identity verification | No explicit identity-verification key. |
| People `parody account` | Parody/Commentary/Fan | The initial attempt returned a sanitized 404; an explicit repeat succeeded and supplied positive `Parody` and `Fan` values, but not `Commentary`. |
| Lists `private` | Private list | Public rows only; no positive private-list shape. |

## Positive structural evidence retained

The Lists capture establishes only the public-list shape. Its list result is
inside a vertical module at
`...entries[].content.items[].item.itemContent`, with
`itemType: "TimelineTwitterList"`; the list object sits at `.list` and exposes
`mode`, `member_count`, `subscriber_count`, `custom/default banner media`, and
`user_results`. Viewer-specific fields present alongside these values remain
outside normalized output.

Blue Premium and the separately handled affiliation, automated-account, and
professional-account shapes were observed elsewhere in the reviewed set; they
are not evidence for any reserved state above.

## Follow-up

The reviewed synthetic People fixture covers the positive Parody and Fan
shapes. No positive minimized fixture or mapping exists for organization,
government, identity verification, Commentary, or private lists. A future
capture should be recorded here by date and scenario only, then reduced to a
reviewed synthetic fixture before decoder support is added.
