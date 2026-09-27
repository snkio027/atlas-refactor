# ADR-0012: Fixed OT-1 stage-12 continuation

Status: Proposed
Date: 2026-09-28
Scope: the existing isolated OT-1 experiment only

The owner request in attachment 421ab0e1-9582-44c5-896f-1c982f423761
asks to complete the original forward/reverse stages from the current stable
stage-12 state. Authority/desired-identity/runtime semantics from F13 are frozen.
This dedicated record satisfies AGENTS.md's requirement for reviewing recovery
scope; it adds no general recovery model, state, evidence schema or public CLI.

## Exact boundary

A separately compiled experiment entry is hard-bound to:

- cluster atlas-refactor-test-ot1, UID b886f730-ea3b-44b9-a904-8bd55ac345f2;
- STOP plan 8aace8f17eeed6736700f07b3e9c9c77258ba2c4f35a8790b45201a636f8b0a2,
  terminal STOP / NextIndex 12 / MIXED_ROLLBACK_VERIFIED;
- immutable manifest 3ada495c8740562e9b06fe4b855b245b6e062c2d48a9ec9c9f6c5c40d2d96f18;
- active lock d7e4a9d78fc284f55b1e34b3aafe9b67998fd7789eb0201814cec2c280a7b1d2;
- current Git 220a113b0570b781cbfd6415c39c527ede93f9fb and source Application UID
  7c4d4e95-1ea2-4874-8325-ef5ab522cd3a;
- the original seven desired commits, 13 resources, four owner names and 29 stages.

The new plan changes only clean implementation/binary binding relative to the
existing semantic plan. Its existing SOURCE_RELEASED continuation field remains
historical lineage, not a selectable execution entry. This executable alone
binds stage-12 constants and enters original index 13, SECOND_SOURCE_RELEASED.
Normal atlas-ot1 and the older fixed entries cannot resume this STOP.

## Fresh anchor and lock handoff

Verify the immutable STOP bundle, exact terminal, prior checkpoint, original
baseline and source-released lineage. Before any lock handoff, repeat read-only
stage-12 Ownership and full Gate-B against the current cluster: all 13 original
UID/content/SSA facts, capability-foundation tracking, no window, stable control
identities/permissions, desired App state, node/runtime and repeat-apply zero
writes. Verify audit continuity, current Git, cluster/kubeconfig and exact STOP
lock before and after this read. Any mismatch stops without live mutation.

The request to continue authorizes this bounded remaining-stage workflow; the
concrete clean binary and canonical plan digest are recorded before execution.
Use the existing compare-and-replace lock handoff, retaining the predecessor
lock in the new attempt. No lock deletion creates an unlocked execution gap.
The fresh anchor is not a retroactive checkpoint for the historical STOP.

Execute only original indices 13..28 with existing exact UID/RV/full-spec/Git
request guards and 300-second phase deadlines (100-minute total budget including
anchor/overhead). Do not replay stage 12's Git publication or any adoption step.
Publish only the original four remaining commits to codex/ot1-desired-state;
operate only the approved foundation Applications. Argo remains the sole writer
of transferred resource content/tracking. No new credentials, trust-root action,
Tier-0 write, cluster deletion/rebuild, forced conflict or refresh is permitted.

Unexpected behavior stops immediately and retains the successor lock and full
mutation evidence. Success removes only the exact lock owned by this attempt.
No retry, arbitrary resume or automatic rollback is added. Existing historical
STOPs, manifests and the read-only F13 verification stay immutable.

## Completion

Test exact entry/no stage-12 replay, failed-anchor/no handoff, owner/lock/audit
rejection and remaining-stage STOP behavior; run locked task quality and tagged
race/vet. A fresh anchor and runtime result are distinct from local tests.

After forward and reverse Gate-B pass, stop S1 feature development. Consolidate
final evidence, check architecture consistency and record ADR disposition for
owner review. This does not approve production/cutover, merge, CI changes or S2.
ADRs remain Proposed until an explicit disposition; remote CI is a separate gap.
