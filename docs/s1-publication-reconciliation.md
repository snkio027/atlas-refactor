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
platform conformance and immutable OT-1 contracts. Runtime remains pending.
