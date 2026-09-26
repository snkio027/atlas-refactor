# Resource-scope integration gate

Status: PREPARED, not executed. ADR-0001 remains Proposed.

The [third run](integration-20260927-03-result.md) exposed incorrect observer
assumptions about Argo CD's CRD and cluster-scoped tracking behavior. The next
candidate corrects interpretation without changing the GitOps topology,
restoring Seed authority, or excluding any of the 39 durable Seed resources.

- Immutable source ref: `integration-20260927-04`; exact commit recorded before approval.
- Public repository: `https://github.com/snkio027/atlas-refactor.git`.
- Profile: `profiles/integration-04.json`.
- New cluster: `atlas-refactor-test-vs0927d`, one Linux ARM64 Kind control-plane.
- Docker: owner-local `orbstack`; loopback IPv4 Kubernetes API.
- Namespace: `argocd`; External Root: `atlas-refactor-root`.
- Independent execution clone: `/private/tmp/atlas-refactor-integration-04`.
- Preserve every previous cluster, immutable tag and evidence set.

Execute only after explicit owner approval of the exact commit and this target.
Run doctor, deterministic render, first apply with SIGINT just after Root create,
resume, strict status, ownership verification, repeated apply, then the missing
artifact, configuration drift and paused-controller unhealthy fixtures with
restoration. Capture command exits, tool and render hashes, cluster identity,
Application revisions/Sync/Health and Metadata-only API audit.

All four Applications must be idle and Synced/Healthy at the approved commit.
For 36 non-CRD durable resources require exact tracking plus Argo SSA, using
`argocd` as the tracking namespace for cluster-scoped resources. For each of the
three CRDs require current Synced inventory, successful exact-revision sync
result and Argo SSA spec fields. Keep the four ephemeral hooks separate.
Check durable UIDs/content and control UIDs/specs/data before and after repeat.

Audit must prove exactly one successful Root create, only Receipt creation
on resume, and zero Bootstrap API mutations on repeat and each refused failure
path. The unhealthy fixture may touch only this controller's replicas and this
self Application's health, with UID/resourceVersion guards and unchanged-spec
checks; restore one ready replica and healthy reconciliation afterwards.

Any failed criterion keeps the overall Gate failed. No tag movement, baseline
substitution, normal-path repair, cluster deletion or original Atlas mutation.
Keep ADR Proposed after test execution; architecture acceptance is owner review.
