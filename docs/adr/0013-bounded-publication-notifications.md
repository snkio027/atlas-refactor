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
