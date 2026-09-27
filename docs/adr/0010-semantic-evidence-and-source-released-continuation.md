# ADR-0010: Semantic evidence and SOURCE_RELEASED continuation

Status: Proposed

## Decision under review

Keep the normal Bootstrap authority and all mutation fences unchanged. Evidence
uses two independent reads of canonical proof facts (atlas.semantic-proof/v1).
UID, content, tracking, Argo SSA field ownership, readiness, full Application
spec, revision, operation request/outcome and comparison resources must agree.
RV values, managedFields timestamps, reconciliation timestamps and Node heartbeat
bookkeeping may advance. Both raw reads remain private evidence. Any unavailable
read or changed proof stops; no resampling to hide a disagreement.

For OT-1 transitional gates only, a persistent unrelated Application may report
the immediately preceding distinct planned revision if its complete local source
inputs are byte-identical. The plan's locked render hashes prove this equivalence;
UID/spec must persist and ordinary idle/Synced/Healthy/condition checks still pass.
All foundation owners, platform-control, changed source inputs and every Atlas
Gate retain exact current revision. Unknown revisions are rejected. No timeout
increase, refresh writes or changes to the original seven desired commits.

F13's [ADR-0011](0011-desired-identity-and-durable-handoff.md) extends the same
bounded source-equivalence proof to full read-only gates after initial adoption.
It supersedes only the blanket full-gate exact-revision rule above; critical
owners and all mutation fences remain exact. Historical STOPs remain STOPs.

The sole continuation supported is an original attempt stopped in SOURCE_RELEASED
after a verified BASELINE_ADOPTED and successful parent detach/orphan release.
Bind the immutable predecessor manifest, plan, terminal, checkpoint, snapshot and
STOP lock; validate the request journal and audit, and independently observe the
current source-released state. A new CONTINUATION_ANCHOR_SOURCE_RELEASED receipt
records that observation, never a retroactive PASS for the old timed-out stage.
A new clean binary/plan binds the same target, scope, 29-stage graph and seven
revisions. Execution begins at original index 2, MIXED_SECRETS_STRICT_REFUSED.

Explicit execution approval authorizes a compare-and-replace lock handoff after
fresh anchor verification. The predecessor lock is retained in the new attempt;
the live lock is atomically replaced, never removed to create an unlocked gap.
STOP keeps the successor lock. Unknown/crashed handoffs require human review.
No generic resume, rollback, Secret access, cluster rebuild, direct resource
apply, tracking edits or Bootstrap authority recovery are introduced.

Historical attempts retain their original rules and terminal outcomes. Local
validation is not runtime proof. This change remains subject to review and a new
exact-plan execution decision; the current cluster is preserved.
