# Fourth integration result: PASS for the disposable Bootstrap contract

The approved baseline `0ef918801dac84ac913416f4c1756bf08045a485`, tag
`integration-20260927-04`, passed the complete minimum Bootstrap vertical slice
on `atlas-refactor-test-vs0927d`. The final strict status was `ADOPTED`, exit 0.
This establishes the normal Go Bootstrap behavior for this disposable target;
ADR-0001 remains **Proposed** and authority cutover is not authorized.

## What passed

| Gate | Observed result |
| --- | --- |
| Repository and artifacts | Clean exact commit; anonymous HTTPS ref matches; pinned tool/artifact hashes and two identical renders |
| First apply with SIGINT after Root creation | Exit 1; latch retained; one Root created; Receipt absent; strict status HANDOFF_PENDING/1 |
| Resume | Exit 0; exactly one Bootstrap API mutation, creating Receipt; no Seed reinstall or Root rewrite |
| Four Applications | Idle, Synced/Healthy, all reporting the approved commit |
| External Root | No parent tracking, owner reference or finalizer; exactly one successful creation across the run |
| Durable Seed | All 39 checked: 36 exact tracking/Argo SSA records and 3 CRD spec SSA/inventory/exact-revision sync records |
| Repeated apply | Exit 0; zero Bootstrap API mutation requests; control and durable resource identities/content/tracking stable |
| Missing artifact | Doctor/apply exit 1; zero writes; original chart restored byte-for-byte; status ADOPTED/0 |
| Configuration drift | Status DRIFTED/2, apply exit 1; zero writes; original profile unchanged; status ADOPTED/0 |
| Unhealthy GitOps | Status ADOPTED_DEGRADED/1, apply exit 1; zero Bootstrap writes; fixture restored; status ADOPTED/0 |
| Final restoration | Controller 1/1, node Ready, all four Apps healthy at the baseline, all 39 persistent identities/content unchanged |

API audit separates the four authorized fault-fixture patches from Bootstrap
commands. After Root creation the only Bootstrap mutation was Receipt creation.
The explicit fixture actions were controller pause, health injection, health
restoration and controller restoration. No unexplained administrator mutation
remained. The controller's managed-fields list gained `kubectl-patch` from that
fixture; its UID, spec/content and Argo tracking/SSA evidence remained valid.

## Why this proves handoff

The Root is external and instantiated once. The two intermediate Applications
and `argocd-self` have the expected parent tracking, source paths and projects.
The immutable Receipt binds the identity, Root, self and GitOps Signal UIDs;
the Signal is tracked by self and present in its inventory. All Applications
report the exact source commit.

Resource checks cover the complete locked durable Seed inventory. For the three
CRDs, the pinned controller intentionally uses reconciliation evidence rather
than tracking annotations; the verifier checks current Synced inventory,
a successful sync at the exact commit, and Argo SSA fields covering spec.
These CRDs are not omitted from the 39-resource Gate. The four ephemeral Helm
hook objects are excluded only from durable identity assertions.

The durable latch survives interruption and permanently denies normal Seed
writes. Resume submits only Receipt, and repeated apply performs only reads,
including client-side manifest decoding. The three negative cases preserve that
boundary. Green pods or a Receipt alone were never the acceptance criterion.

## Initial local refusal and recovery

The first launch was rejected before cluster creation because the separately
prepared `.state` parent directory was mode 0755. No kubeconfig or audit directory
existed and no Tier-0 write occurred. The failed command and error are retained.
The directory was tightened to 0700, then the same binary, commit, tag and target
were retried. No source change, tag movement, cluster replacement or runtime
recovery was used to pass the Gate.

## Test scope and retained evidence

The unhealthy case paused only this cluster's application-controller, then
patched only self's health with UID/resourceVersion preconditions. The CRD has
no status subresource, so the patch used the Application endpoint; checks proved
spec stayed unchanged. Health and one ready controller replica were restored.
This validates refusal of an observed unhealthy state, not every natural Argo
failure mode.

See the [sanitized evidence index](evidence/integration-20260927-04.json) for
binary/tool/render hashes, cluster/control UIDs, per-resource records, command
windows, exit codes and audit counts. Raw private outputs, driver sources and
Metadata-only audit are retained in the sibling repository under
`.state/evidence/integration-20260927-04/`. The bound execution checkout and its
private kubeconfig remain at `/private/tmp/atlas-refactor-integration-04`.
The tested binary SHA-256 is
`5cdb0e380078528ba02a6d39504029ef8a1a24a9ec687a8849f2d37548e4c28f`.

All four experiment clusters and all four tags are retained. Original Atlas
was not changed or operated on. The execution checkout stays at the approved
commit; main may advance with this result documentation without changing the
running GitOps ref. Do not substitute main for the recorded baseline on rerun.

`task quality` with locked real Helm passed again. This proof is limited to the
owner-local macOS ARM64 CLI and one-node Linux ARM64 Kind/OrbStack substrate.
It does not establish complete old Shell CLI parity, production readiness,
Admission-enforced authority protection, independent Recovery/Drill, or release
provenance. No feature expansion or ADR acceptance is part of this test result.
