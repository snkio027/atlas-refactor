# ADR-0011: Application desired identity and durable handoff

Status: Proposed
Date: 2026-09-28
Scope: atlas-refactor schema-3 development candidate and read-only OT-1 gates

## Requested decision

The owner requests a single S1 Finalization Change: consolidate F13/F14 into
plan-bounded Desired Identity, preserve durable authority and F12 semantics, and
complete the current isolated stage-23 STOP through a fresh forward Gate-B anchor
and original reverse stages 24..28. This dedicated ADR records both the corrected
read-only rule and the exact incident execution boundary; it remains Proposed.
No general recovery API, new evidence schema, CI infrastructure or S2 is included.

## Evidence and corrected invariant

The b0768e7 -> 220a113 diff only re-adds capability-foundation to platform-control's
child Application projection. The five healthy Applications still at b0768e7 in
the F13 deadline read have unchanged local source closures. Repository SHA records
a platform snapshot; it is not itself each Application's desired-state identity.

OT-1 uses complete local source-closure hashes and stable UID/full-spec fences
at transitional and full read-only rollout gates. Starting at the current phase,
walk backward only through the published prefix of the verified immutable plan.
Consecutive planned revisions with the same complete source digest and Application
spec form one Desired Identity class. Stop at the first changed, missing or unknown
input; a change followed by a revert does not reconnect the older class. Future
planned commits, unplanned/unknown SHAs, remote sources, plugins, generators,
multi-source and escaping Kustomization inputs remain ineligible.

F14 demonstrated why a one-commit bound was incorrect: four unchanged leaves at
220a113 passed stage 22 (849d4f8), then were rejected immediately after publication
of 6c1311f. Their source inputs are identical across all three commits. Distance
in repository history is not a desired-content boundary. UID/full spec must still
persist; Sync/Health/idle/conditions/resource readiness and ownership still pass.
This rule affects evidence only, never a mutation request's revision fence.

Initial BASELINE_ADOPTED, platform-control, every foundation owner and active
ceremony owner retain exact-current revision checks. Mutation requests retain
exact current Git, UID, RV and full-spec fences. No timeout, plan/evidence schema,
state graph, source revisions or public lifecycle interface changes.

This supersedes ADR-0010's one-preceding-revision bound and blanket exact-revision
rule for every full Atlas gate. It does not generalize source equivalence to arbitrary Git history.

## Schema-3 authority after Receipt

Initial handoff is unchanged: before Receipt, Root and the required Applications
must meet exact-current revision, health/idle and both Seeds' complete ownership
proof, including exact successful sync proof for CRDs. The durable latch revokes
normal Seed mutation authority before Receipt exists.

After a valid immutable Receipt, status answers whether that handoff and current
Seed ownership remain intact. It does not assert latest rollout or runtime health.
The proposed separation is explicit:

| Fact | Post-Receipt Bootstrap status | Full Observation / Gate-B |
| --- | --- | --- |
| Identity, latch, Receipt, Signal content and UID bindings | Required, fail closed | Required, baseline-bound |
| Root/self full spec, External Root unparented, bootstrap AppProject | Required, fail closed | Required |
| Both Seeds' persistent live objects and Argo tracking/SSA | Required; CRD still needs Argo SSA of spec | Required through authority evidence |
| Latest Seed operation / CRD sync-result revision | Initial adoption only; Receipt proves that transition | Current operation/conditions/resource readiness checked as rollout facts |
| Latest branch SHA on unrelated leaf Apps | Not an authority fact | Desired-content equivalence with bounded planned SHA |
| App health/idle/rollout, nodes/placement/PVC/HTTPS | Not an authority fact | Still mandatory; failures reject Gate-B |

An intact schema-3 handoff remains ADOPTED through an unrelated commit or workload
rollout, including a currently unhealthy workload. ADOPTED therefore cannot be
used as a platform-health assertion. Missing or changed authority records, wrong
Root/self identity/spec, invalid Signal, lost Seed resources or lost tracking/SSA
still fail closed. Unknown authority reads remain UNAVAILABLE. No failure ever
restores Bootstrap mutation rights. Schema 1/2 keep historical behavior.

This supersedes ADR-0005's post-Receipt all-Applications-current rule and the
schema-3 interpretation of ADOPTED_DEGRADED for ordinary rollout/runtime health.
ADOPTED_DEGRADED remains for missing continuing authority/ownership evidence;
DRIFTED/UNAVAILABLE remain available. No new status fields or states are added.
Gate-B explicitly invokes the existing read-only node verifier, preserving node
image, Kubernetes version, role labels, data taint and readiness checks formerly
reached through Bootstrap status. No runtime check is removed from Gate-B.
The existing atlas-dev verify command also calls the node and development-rollout
checks explicitly, preserving its prior validation coverage. It has no reviewed
multi-revision plan and therefore retains exact-current rollout checks. No CLI
command or ordinary lifecycle interface is added.
Normal repeated apply retains source/repository/tool validation and is a no-op
once durable authority is verified; it is not a workload repair or rollout command.

## Verification and limits

Regressions must prove full-gate equivalence across multiple continuously unchanged
planned revisions PASS; changed source and intermediate content/spec changes
(including change-and-revert), critical owners, future/unknown revisions, UID/spec
drift and unhealthy/active leaves FAIL. Schema-3 tests must prove Receipt plus a later unrelated commit remains
ADOPTED without querying branch HEAD; workload/runtime failures still fail the
separate rollout/runtime gate. Root/Receipt/latch/Identity/Signal/Seed ownership
damage and unknown authority reads fail closed without writes. First handoff
must still reject stale revision or unproven CRD sync evidence.

Run locked task quality plus all fixed-entry tagged race/vet. Retained runtime
snapshots should reproduce F14 and demonstrate its correction without rewriting
the original failure. The F12 operation → fresh comparison rule is unchanged.

## Fixed finalization of the current stage-23 STOP

The build-tagged entry `experiments/foundation-ownership/finalize` is bound to:

- target atlas-refactor-test-ot1, UID b886f730-ea3b-44b9-a904-8bd55ac345f2;
- predecessor plan 619e492be725fbfb7f3d24aba380479defd64cafad433a107738cf5ab1d65cff,
  STOP / NextIndex 23 / FORWARD_VERIFIED;
- immutable manifest 252baa12ed1d3b56d8c92690bf77981270e1f310c7b3ca915e3c427f62ecae68;
- STOP lock 959051a4bd141acdc59d1bf5373e3e360084ab02f2e508368affc8733f37cd36;
- already-published Git 6c1311ff9fa64a80973eb0ef4f4c3ba406ecac24;
- the three exact owner UIDs, original 13 resources, seven commits and 29-stage graph.

A clean executable and canonical plan are bound before execution. The current
owner instruction authorizes this bounded finalization workflow; its concrete
binding is recorded in the execution decision. Check mode first verifies only
current state. Execute repeats the full read-only
`CONTINUATION_ANCHOR_FORWARD_VERIFIED`: exact predecessor/baseline evidence,
cluster/Git/lock/audit continuity, strict owners 3/4/6, no window, unchanged 13
UID/content/SSA facts, exact-current critical Apps and desired-equivalent ordinary
leaves, full forward Gate-B and repeat-apply zero writes. Any mismatch leaves the
old lock intact. Historical STOP and checkpoint files remain immutable.

Only after that anchor, perform existing lock compare-and-replace and run original
indices 24..28. Do not replay 13..22 or stage 23 publication. The exact remaining
budget is nine protected foundation Application writes and two original Git
publications (dd2e4cd and 28c4dc6); stage 28 includes reverse/final Gate-B. Retain
300 seconds per phase, 40 minutes total including the anchor and preparation.
No Tier-0, direct transferred-resource/tracking, credential or Trust Root writes;
no refresh, force-conflicts, new desired history, rebuild or arbitrary resume.
An unexpected result stops immediately and preserves full authority evidence and
successor lock; only full success removes the exact owned lock.

On success freeze S1, preserve one final evidence bundle referencing historical
STOPs, and finish architecture/ADR/PR review. ADR acceptance, CI and release gates
remain distinct. This fixed entry is an experiment reference, not a new ordinary
Atlas lifecycle interface or a general recovery framework.


## Runtime disposition

Implementation `c7a09b4` completed the exact finalization: fresh forward Gate-B,
original 24..28, reverse/final Gate-B, REVERSE_VERIFIED / exit 0. Evidence and
architecture consistency are recorded in [final validation](../s1-final-validation.md).
S1 feature work is frozen. This record remains Proposed for owner disposition;
no production/cutover or remote CI/release acceptance is implied.
