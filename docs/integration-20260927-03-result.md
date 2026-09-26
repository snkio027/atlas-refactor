# Third integration result: FAIL at resource-scope observation

Baseline `a3369c0d2dfe8c0a6ca36195395e794c52a3cef5`, tag
`integration-20260927-03`, ran on `atlas-refactor-test-vs0927c`.
The stronger observer failed closed, but its CRD and cluster-scoped tracking
assumptions prevented a valid handoff from being acknowledged. Resume reached
its configured deadline, returned 1 and created no Receipt. Behavioral parity
remains **unproven**; ADR-0001 stays Proposed.

## Observed results

| Check | Result |
| --- | --- |
| Doctor / two deterministic renders | Passed; fixed tool/artifact/render hashes recorded |
| SIGINT after External Root creation | Exit 1; one successful Root create, latch retained, Receipt absent |
| Resume | Exit 1 at deadline; zero Bootstrap API mutations |
| Four Applications | Synced/Healthy, idle, exact approved commit |
| External Root | No parent tracking, owner reference or finalizer; created exactly once |
| Durable Seed | 36 with expected Argo tracking and SSA; 3 CRDs with Argo SSA spec fields, current inventory and successful exact-revision sync results |
| Strict status using approved binary | HANDOFF_PENDING, exit 1 |
| Missing artifact while pending | Doctor/apply exit 1; zero writes; identical chart restored |
| Configuration drift while pending | Status DRIFTED/exit 2; apply exit 1; zero writes; original profile unchanged |
| Adopted repeat / post-adoption unhealthy | Not executed: no Receipt or ADOPTED prerequisite |
| Restoration | Controller remains 1/1; control and 39 durable resource identities/content stable |

The supplemental fault tests occurred in HANDOFF_PENDING. They do not replace
the planned post-adoption tests. No health or controller-replica fixture was
injected in this round, and no Seed or Root repair was attempted.

## Root cause

Argo CD's [resource-tracking implementation at v3.5.1](https://github.com/argoproj/argo-cd/blob/v3.5.1/util/argo/resource_tracking.go)
uses the Application destination namespace when a resource has none. The four
cluster-scoped RBAC objects therefore correctly carry `argocd` in their tracking
IDs. The observer incorrectly expected an empty namespace.

The [v3.5.1 manifest generator](https://github.com/argoproj/argo-cd/blob/v3.5.1/reposerver/repository/repository.go)
explicitly skips injecting tracking metadata on CRDs. All three live CRDs were
SSA-applied by Argo and recorded in the successful self-sync at the approved
commit, but lacked tracking annotations by design. The observer incorrectly
required those annotations. Green health still cannot replace resource evidence;
the evidence must also match the pinned controller's actual semantics.

## Correction and its limits

All 39 durable resources stay in the Gate. Non-CRD resources require exact Argo
tracking and SSA, with the destination-namespace rule for cluster-scoped objects.
Each CRD instead requires its current Synced inventory entry, successful
exact-revision sync result, and Argo SSA fields covering spec. Foreign tracking,
missing inventory, stale/failed sync, metadata-only fields or non-SSA managers
are refused. The four Helm hooks retain their separate transient lifecycle.

Regression cases before and after Receipt, race tests, vet, locked real Helm,
all four Kustomize directories and the three-platform build passed. A separately
hashed read-only probe of this third cluster accepted the observed 39 resources
under corrected semantics; Metadata audit showed zero writes. It did not run
corrected apply, create a Receipt, or change the third baseline's failure result.
A [new exact-target integration gate](integration-fourth-slice.md) remains required.

## Retained evidence

See the [sanitized evidence index](evidence/integration-20260927-03.json) for
commit/binary/tool/render hashes, command windows and exit codes, node/control
identity, all Application revisions, 39 per-object records and audit summaries.
Private command outputs, Metadata-only API audit and driver sources are retained
under `.state/evidence/integration-20260927-03/` in the sibling repository and the
separate execution checkout `/private/tmp/atlas-refactor-integration-03`.
No kubeconfig or generated Secret data enters the published evidence. All three
new test clusters and their tags are retained; original Atlas was not operated on.
