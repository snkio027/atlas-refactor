# ADR-0013: Bounded OT-1 publication notifications

Status: Proposed
Date: 2026-09-28
Scope: new normal OT-1 plans only; no continuation or ordinary platform interface.

The owner requested a cohesive F15 correction and a fresh single 29-stage test.
A 300-second stage includes preconditions, publication, reconciliation, complete
capture and Gate-B. Git publication alone does not notify Argo. Periodic branch
discovery did not meet that budget in the retained clean run.

## Limited authority revision

New normal plans explicitly contain `publicationRefresh: true`; it participates
in the canonical plan SHA. Missing/false preserves historical behavior. A
continuation cannot opt in. The schema, 29 stages and seven revision graph stay
unchanged. The derived request schedule lists the allowed Applications.

At each of the six publications, after exact remote SHA confirmation, Executor
submits one standard `argocd.argoproj.io/refresh=normal` annotation on
platform-control. It waits for the annotation to be consumed and a comparison
of the exact published revision. It then waits for each extant foundation
child's full desired spec, written by Argo, before conditionally submitting one
normal refresh on that child. The maximum is six parent and ten child requests.
No ordinary leaf, Root, Secret, source, syncPolicy, permission or tracking write
is added. Existing ownership mode/operation/delete requests retain their guards.

Notification requires the exact cluster/private kubeconfig/current Git fences,
the 13-resource ownership/audit guards, and atomic Application UID/RV/full-spec
tests. Existing refresh requests are not overwritten. Unknown response, drift,
error or deadline stops without resubmission. Every wait rechecks Git, cluster
and critical Application identities/specs. Only the prior and target child spec
are permitted during Argo restoration; after restoration, reversion fails.

Parent health is not used as an intermediate barrier: a detached parent may
wait for prune confirmation, and a reattached parent's health depends on its
children. Child revision, full spec, health, idle and post-operation comparison
are checked before returning. Existing full Capture, critical-owner exact
checks and Gate-B remain the final authority. Refresh success alone is no proof.

## Why normal refresh

In locked Argo CD v3.5.1, explicit refresh selects
CompareWithLatestForceResolve; the controller passes noRevisionCache to
CompareAppState. Periodic expiry instead uses CompareWithLatest. The existing
normal mechanism is sufficient to request fresh branch resolution; neither
hard refresh, an upgrade nor webhook infrastructure is introduced.
Sources: [controller](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/appcontroller.go),
[comparison](https://github.com/argoproj/argo-cd/blob/v3.5.1/controller/state.go).

## Evidence and acceptance

The existing request records contain notification intents/results. Small
create-only publication timestamps record remote confirmation, parent comparison,
child spec restoration and child comparison. They are execution diagnostics,
not a new evidence schema. F12, F13/F14 Desired Identity, durable handoff and
Observer read-only semantics are unchanged. The shared budget remains 300 seconds.

Controlled-clock regressions cover delayed/absent/out-of-order progress, uncertain
responses, unexpected Git/cluster/UID/spec changes and all six bounded schedules.
Only a fresh four-node normal run of 0..28, without manual refresh/spec repair or
continuation, can establish single-attempt runtime completion. Preserve previous
STOPs and final authority evidence. This ADR remains Proposed; runtime PASS is
not production, cutover, merge or release approval.

## Observation execution policy for the final S1 batch

A rejected sample is not automatically a failed transition. The predecessor
read before Transition, post-write Converge and Gate-B closing read share one
internal bounded complete-recapture helper. It uses the original phase context;
no action, notification, operation or outer Gate is repeated. Gate inputs are
collected once before its closing read. An expired phase cannot accept a late
read or create a checkpoint. Baseline is installed only after valid ownership
and audit checks, never from a discarded sample.

Unavailable reads and changed proof facts have distinct internal reasons. A
race is discardable only after all reads succeeded and all four Application
views, both resource views, AppProjects, nodes, target and audit remain safe.
The additional explained difference is monotonic ordinary-leaf comparison
within the existing continuously unchanged planned Desired Identity class.
Every remaining proof fact must match: unknown status changes, operation changes,
UID/spec/content/tracking/permission drift and critical-owner revision changes
stop immediately, even if a later read would recover. Missing evidence also
stops. Successive discarded samples cannot regress the observed comparison.

Proof/SameProof, stored envelopes, the public state/schema and mutation guards
are unchanged. Discarded samples cannot pass Assess. F12 freshness is checked
on every Application view; a coherent sample can still be pending comparison.
No globally atomic Kubernetes snapshot is claimed by this double-read protocol.

Controlled-clock tests exercise all read boundaries for the six publications,
all four Gates, predecessor/post-write/Gate closure, repeated valid races until
the original deadline, F12 across all successful sync phases, simultaneous
unsafe changes and unavailable reads. A complete scripted 29-stage execution
uses the real Executor and asserts the same exact Git/Kubernetes action sequence
with and without recapture. Bootstrap/TLS runtime inputs in that test are
fixtures; only the approved fresh-cluster run can prove the runtime acceptance.
