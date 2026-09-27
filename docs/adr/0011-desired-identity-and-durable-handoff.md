# ADR-0011: Application desired identity and durable handoff

Status: Proposed
Date: 2026-09-28
Scope: atlas-refactor schema-3 development candidate and read-only OT-1 gates

## Requested decision

The owner requested the F13 correction in attachment
b48e4162-5bec-4072-be10-4ce68ae0a17a: distinguish repository revision,
Application desired identity and runtime state; separate post-Receipt Bootstrap
authority from rollout verification; implement regressions and perform only a
read-only check of the current stage-12 state. No continuation, recovery, cluster
rebuild, refresh/sync request or STOP-lock handoff is authorized by this change.
This ADR is the dedicated review record required by AGENTS.md. It remains Proposed;
local implementation/verification does not constitute acceptance or deployment.

## Evidence and corrected invariant

The b0768e7 -> 220a113 diff only re-adds capability-foundation to platform-control's
child Application projection. The five healthy Applications still at b0768e7 in
the F13 deadline read have unchanged local source closures. Repository SHA records
a platform snapshot; it is not itself each Application's desired-state identity.

OT-1 reuses the existing complete local source-closure hashes and spec/UID fences
at full read-only rollout gates as well as transitional gates. Only the immediately
preceding distinct planned SHA is eligible. Both revisions must exist in the
verified plan; complete supported inputs must hash equally; UID/full spec must
persist. Sync/Health/idle/conditions/resource readiness and ownership still pass.
Changed inputs, unrecognized/two-epochs-old revisions, remote sources, plugins,
generators, multi-source or escaping Kustomization inputs remain rejected.

Initial BASELINE_ADOPTED, platform-control, every foundation owner and active
ceremony owner retain exact-current revision checks. Mutation requests retain
exact current Git, UID, RV and full-spec fences. No timeout, plan/evidence schema,
state graph, source revisions or public lifecycle interface changes.

This narrowly supersedes ADR-0010's blanket exact-revision rule for every full
Atlas gate. It does not generalize source equivalence to arbitrary Git history.

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

Regressions must prove full-gate unchanged prior source PASS; changed source,
critical owner, unknown/two-epoch revision, UID/spec drift and unhealthy/active
leaf FAIL. Schema-3 tests must prove Receipt plus a later unrelated commit remains
ADOPTED without querying branch HEAD; workload/runtime failures still fail the
separate rollout/runtime gate. Root/Receipt/latch/Identity/Signal/Seed ownership
damage and unknown authority reads fail closed without writes. First handoff
must still reject stale revision or unproven CRD sync evidence.

Run locked task quality and a new read-only stage-12 check against the retained
F13 target. Preserve the old STOP terminal, lock and immutable bundles. Current
read-only success is a new observation, never a retroactive checkpoint or
permission to execute the remaining ownership stages. S1 acceptance stays open.
