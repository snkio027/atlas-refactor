# Second integration result: FAIL at Seed ownership

Baseline `881448cc75214760a9145feb96c119807dd78de6`, tag
`integration-20260927-02`, ran on `atlas-refactor-test-vs0927b`. Its earlier
image-import and ConfigMap corrections worked, but the GitOps ownership gate
failed. Behavioral parity remains **unproven** and ADR-0001 remains Proposed.

## What the real run established

| Check | Observed result |
| --- | --- |
| Initial apply and deliberate SIGINT after Root create | Nonzero exit; one successful Root create; latch retained; Receipt absent |
| Resume | Exit 0; exactly one Bootstrap mutation: Receipt creation; no Seed reinstall |
| Four Applications | Idle, Synced/Healthy, all at the exact baseline SHA |
| External Root | No parent tracking, owner reference or cascading finalizer |
| Repeated apply | Exit 0; zero Bootstrap API mutations; control UIDs/specs/data stable |
| Missing artifact | Doctor/apply nonzero; zero writes; original chart restored |
| Configuration drift | Status DRIFTED/exit 2; apply exit 1; zero writes |
| Injected unhealthy state | Status ADOPTED_DEGRADED/exit 1; apply exit 1; zero writes; fixture restored |
| Persistent Seed ownership | **FAIL: 38 ordinary Seed objects lacked Argo tracking and Argo SSA ownership** |

The negative tests occurred after the CLI had issued a Receipt. That Receipt
was premature, so their successful refusal behavior does not establish a valid
authority transition.

## Why healthy was insufficient

`argocd-self` inherited `ApplyOutOfSyncOnly=true`. Its existing Seed objects
already matched Git content, so Argo skipped them. The Application health and
sync state were green and the GitOps-created Signal was valid, but those facts
did not demonstrate adoption of the existing resources. Argo documents that
this option [applies only out-of-sync resources](https://argo-cd.readthedocs.io/en/latest/user-guide/sync-options/#selective-sync).

Of 42 Seed objects present during the observation, 38 ordinary objects were
untracked. The tracked objects were `argocd-cm` and three Redis hook helpers.
The complete rendered Seed has 43 objects: 39 ordinary durable objects and four
Helm hook objects (Job, ServiceAccount, Role and RoleBinding). Hook helpers can
also be replaced by Argo and must not be treated as stable durable identities.

The observer accepted a healthy Signal and Application inventory without
checking the actual Seed. Consequently it created a Receipt and returned
ADOPTED despite the failed ownership gate. This is a Bootstrap correctness bug,
not a successful handoff with merely incomplete documentation.

## Correction and verification scope

The correction removes selective sync from `argocd-self` so its initial sync
applies matching Seed objects. Receipt commitment and subsequent ADOPTED
observations now additionally require:

- Root, self and both intermediate Applications at the exact resolved Git SHA;
- every durable object in the locked rendered Seed present in the cluster;
- exact `argocd-self` tracking annotations and Argo controller SSA managed fields;
- External Root outside another Application's tracking.

The inventory comes from the verified renderer. `kubectl create --dry-run=client
--validate=false` decodes the manifests locally; it is not a create API request.
The observer then performs an ordinary read of those objects. This avoids a
second handwritten inventory, a new configuration format, or a runtime YAML
dependency. Missing or unreadable evidence cannot commit a Receipt or restore
Seed authority. Ephemeral hook objects are excluded from durable ownership.

Regression tests reproduce green Applications with missing tracking, missing
Argo managed fields, missing Seed objects, stale revisions, unavailable reads,
and ownership loss after Receipt. A read-only probe of this actual failed
cluster confirmed the corrected ownership observer rejects its untracked Seed;
Metadata audit showed zero mutations during that probe. This is not a successful
new-bootstrap test: the corrected full handoff still needs its own baseline.

## Fault-injection detail and evidence

The installed Application CRD does not expose a `status` subresource. The first
health injection therefore returned NotFound; its finally block restored the
controller, and that failed test attempt is retained. The successful retry used
a JSON Patch affecting only `/status/health` on the Application endpoint, with
UID/resourceVersion preconditions and checks that spec stayed unchanged. The
controller was stopped during the fault and restored to one ready replica.
This tests refusal of an observed unhealthy state, not every natural Argo
failure mode.

See the [sanitized evidence index](evidence/integration-20260927-02.json). The
private execution checkout is `/private/tmp/atlas-refactor-integration-02`;
command exits, API audit, resource projections and test-driver source hashes
are retained under its `.state/evidence/integration-20260927-02/`. Kubeconfig
is not part of the evidence export. Both earlier clusters and tags remain
unchanged; there is no in-place recovery or baseline substitution.
