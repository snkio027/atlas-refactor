# F15: publication and reconciliation

Implementation work is confined to OT-1 publication notification/completion.
The prior [clean run](s1-clean-revalidation-20260928.md) remains STOP at index 12.
Historical cross-attempt completion remains distinct from clean single-run PASS.

## Recovered timeline (UTC, 2026-09-27)

| Event | Time / evidence |
| --- | --- |
| Stage 12 starts | 18:42:56.573396, original stage record |
| Git push | 18:42:58.297173 → 18:43:00.038251, request-024; 1.741 s |
| Remote fence completed | After push, before fetch intent at 18:43:00.908459; exact completion was not separately recorded |
| Runtime checkout complete | 18:43:01.002385, request-026 |
| Parent controller refresh, recent comparison | 18:43:17.991461, level 1 |
| Parent periodic latest comparison | 18:43:46.395682, level 2; completes in 63 ms, git_ms=8 |
| Parent next periodic latest comparison | 18:45:50.693357, level 2; completes in 46 ms, git_ms=8 |
| Original stage deadline | Approximately 18:47:56.573396 |
| Parent next periodic comparison and target automated sync | 18:48:01.195807 / 18:48:01.256189; 8809173 |
| Argo begins applying child Application | 18:48:02.371083, controller log; request body not logged |
| Child target manifest build | 18:48:02.446, repo-server |
| Child reports target comparison | reconciledAt 18:48:02 |
| Parent target operation completes | 18:48:22.093 |
| Parent final comparison | 18:48:22.098 → 18:48:22.115 |

The parent did compare during the five-minute interval: this was not five
minutes spent exclusively waiting for its first queue turn. Its two level-2
comparisons were fast yet did not initiate the new target sync. The first
confirmed target sync was about 301 seconds after push completion. This points
to branch discovery/resolution as the main waiting interval, rather than Git
push or the final manifest build. These logs do not expose the returned revision
for every cache lookup, so they do not isolate cache TTL from every queue/cache
interaction. No CPU throttling metric was captured; 500m limits are not a proven
cause. Metadata-only audit also cannot establish the exact request body of the
first automated-spec write. Those unknowns remain explicit.

Read-only controller/repo-server logs are in
`.state/latest/ot1-notify/diagnosis/`. The immutable original bundle
`.state/authority/ot1-clean-67f00b25` is unchanged. New execution diagnostics record
the missing Git-confirmed/parent-compared/child-spec/child-compared observations,
alongside existing request timestamps and controller logs.

## Correction and verification boundary

[ADR-0013](adr/0013-bounded-publication-notifications.md) explicitly records the
new limited annotation authority. New canonical plans bind that authority;
historical plans/continuations do not gain it. Every one of six publications
notifies the parent once, then conditionally the extant foundation children
after Argo has restored their exact spec. Maximum: 16 notification writes.
No application source/syncPolicy repair, tracking patch, hard refresh, automatic
retry or public recovery interface is added.

F12 comparison freshness, Desired Identity, critical-owner exact checks and
full Gate-B remain intact. The normal 300-second shared stage deadline remains.
Synthetic temporal tests are local coverage, not proof of runtime completion.
A new clean full-platform setup and one uninterrupted 0..28 attempt are required.

Locked `task quality` passed: Go vet/race, actual Helm/Kustomize rendering,
platform conformance and immutable OT-1 contracts. The fresh runtime outcome is recorded below.

## Fresh single-run result: STOP, 2/29 checkpoints

The owner-requested clean test used implementation
`190f004e59f67ffcae74107ba3ebfd22666fe5e0`, clean Go 1.27.1,
CGO_ENABLED=0, -trimpath. Executed atlas-ot1 SHA256:
`2992853e90e4fd92d9f78178aaa963f2695a8a7ce7dac741bfe60c97334c38db`.

| Binding | Value |
| --- | --- |
| New four-node cluster UID | `ce4e11e4-5116-4373-a49c-c85a8731e9af` |
| Plan SHA256 | `fd2b9bc3616cec80b26d7916aa67a5baa3b34fda351edef81542d157dbe761c4` |
| Baseline Git | `6618eae414328aac42d98aac3d7c45574f3586f0` |
| Published/current Git | `93984aa5703fb2ce84e3cffc9fea440960dcded8` |
| Run | 2026-09-27 19:55:41.844299–19:57:01.880381 UTC, exit 2 |
| Last checkpoint | 1 / SOURCE_RELEASED |
| Terminal | STOP / NextIndex=2 / MIXED_SECRETS_STRICT_REFUSED |
| STOP lock SHA256 | `20b4d327b31ab99b0150c1f1042fec847a3d62184e4af7653a4b46869635fb2c` |
| Evidence manifest | `76a1dddadf4bfeed006b9213022321c3756c148b707d8e73e4334f5cc01b927a` |

Only old OT-1 UID 478c4e71 was deleted, after verifying retained evidence and
Trust Root backup. Default kubeconfig stayed byte-identical. The new key uses the
approved OT-1 same-host development exception, in the independent private folder
`/Users/nekoreb/Workspace/01_Vault/atlas-refactor-ot1/rebuild-f15-20260928`.
Three new strict namespace/name-bound SealedSecrets were published; no plaintext
or private key was uploaded. Historical bundles/STOPs remain unchanged.

### Initial platform acceptance passed

- Four Ready nodes: control-plane, gateway, compute, data; loopback 18080/18443.
- Bootstrap ADOPTED, HTTPS/PVC/placement, repeat apply zero Kubernetes writes.
- S3 signed CRUD, presigned GET, multipart, cleanup; anonymous/cross-bucket 403.
- Grafana login, 26 dashboards, working Prometheus datasource.
- Prometheus 27 targets UP, 31 rule groups, four-node metrics.
- Local Alertmanager test alert fired and resolved; no external notification test.
- Baseline stage 0 Ownership and Atlas Gate-B VERIFIED.

### First notification chain has runtime proof

| Event | UTC |
| --- | --- |
| Git push | 19:56:16.779895 → 19:56:18.654066 |
| Git-confirmed marker | 19:56:19.932529 |
| Parent normal refresh request | 19:56:20.659972 → 19:56:20.749606 |
| Controller consumes parent notification | 19:56:20.750605; normal refresh, level 3 |
| Parent exact comparison observed | 19:56:24.011000 |
| Child normal refresh request | 19:56:25.559203 → 19:56:25.648023 |
| Controller consumes child notification | 19:56:25.646187; normal refresh, level 3 |
| Child exact comparison observed | 19:56:26.228107 |

The first detach publication reached critical parent/child comparison in 6.296 s
from the Git-confirmed marker, with exactly two annotation requests. Stage 1
completed its guarded orphan deletion and checkpoint. This proves the first
notification path; it does not prove reattachment or the other five publications.

### F16: leaf changes during coherent capture

Stage 2 created secrets-foundation and submitted the planned strict operation.
Argo refused the shared resources as intended. During the complete capture,
ordinary leaf edge advanced its observed revision from baseline 6618eae to
published 93984aa. Its UID, full spec and source closure were unchanged. Existing
Desired Identity rules accept either stable observation. However, opening and
closing inventory proofs differed, so Capture recorded
`INVENTORY_CHANGED_OR_UNAVAILABLE`, and Converge stopped immediately.

This was a coherence failure, not a timeout, old-revision eligibility failure,
refresh rejection or missed parent restoration. There is no stage-2 checkpoint.
No mutation retry, continuation, rollback, manual refresh, spec repair, lock
clearance or further cluster rebuild followed STOP.

Offline replay confirms the historical assessment is STOP with inventory proof
mismatch. A new read-only sample verifies the expected stage-2 strict refusal,
all 13 original identities/content/SSA/tracking and zero windows. That sample
does not replace historical evidence. All four nodes remain Ready. The ordinary
Bootstrap status command rejects the deliberately detached Git projection as
a capability-render mismatch; this intermediate phase is not an Atlas Gate,
and no final ADOPTED/platform-ready claim is made for it.

The run issued one Git publication (three Git requests) and five Kubernetes
writes: parent refresh, source refresh, source orphan delete, secrets owner
creation and strict sync request. All eight client requests exited zero.
Audit replay retains all 7,855 baseline audit IDs within 9,062 collected IDs,
matches the five kubectl writes and verifies scoped writer authority.

`TestPublicationLeafAdvanceRequiresFreshCoherentCapture` preserves the discovered
boundary: either stable eligible sample can pass, but combining observations
across a revision change cannot. Acceptance rules and F12/F13/F14 were not
changed. A bounded, read-only recapture after discarding a failed sample is a
possible follow-up to review; it has not been implemented or executed here.

### Retention and disposition

Immutable private bundle `.state/authority/ot1-notify-fd2b9bc3` contains 105 files,
102,251,689 bytes: executed binaries, canonical plan/seven desired commits,
setup/credential backup receipts, attempt, request records, publication timing,
controller/repo logs, audit and offline replay. No plaintext credential, private
key or kubeconfig is included. Runtime checkout and STOP lock stay in place;
`.state/latest/ot1-notify/result.json` points to the retained bundle.

Clean single-attempt 29-stage completion remains **unproven**. PR #6 stays draft;
ADR-0009 through ADR-0013 remain Proposed. No S2, generic recovery, new evidence
schema or production/release claim is included.
