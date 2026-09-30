# D1 acceptance record

Status: INITIAL RUNTIME STOP; correction awaiting a fresh attested-package run.
The actual CI package `v0.1.0-d1.3` passed provenance, preparation and preflight
negatives, and created its four-node target, but base GitOps handoff failed.
No consumer credentials or ciphertexts were generated. The target, Git definition,
audit and incident key backup are retained. D1 is not accepted; ADR-0014 remains
Proposed.

| Gate | Required evidence | Current status |
| --- | --- | --- |
| Product/deployment/instance separation | Strict inputs, fixed projection, exact deployment commit | Local regression PASS; source HEAD is not deployment authority |
| Complete archive | Runtime resources, executable binding, notices, isolated execution | Local development archive PASS outside the checkout; changed manifest rejected |
| Tool preparation | Official digests, private paths, interruption/corruption handling | Real pinned tool download/preparation PASS; simulated truncated HTTP body resumed and verified |
| Go and repository quality | Vet, race tests, real Helm/Kustomize, frozen S1 fixtures | PASS, including the final run with newly prepared release tools |
| Trust Root mechanics | Key/cert match, backup readback, actual seal/unseal | Synthetic 4096-bit key and locked kubeseal roundtrip PASS; namespace rebinding rejected |
| Git publication | Closed generated tree, exact-parent commit, unknown acknowledgement | Real local Git and controlled remote-response regression PASS |
| Access process | Multiple readers; installation exclusion; lost connection exits | Process and OS-lock regression PASS; live service/browser NOT RUN |
| Release provenance | Pinned Actions, locked quality, attested archive, draft promotion | CI run 36730610488 PASS; archive and exact workflow/tag/source attestation verified |
| Independent clean machine | Actual macOS/OrbStack versions, capacities, empty state/cache | Deferred by owner; this run uses the current Mac and fresh state |
| Dedicated deployment | New isolated deployment branch and two generated commits | Base commit e34b68e8 published to atlas-d1; full consumer commit absent |
| Live Trust Root | New controller key and declared independent backup destination | Controller generated a new key; private incident backup readback PASS; installer backup stage not reached |
| Installation and services | Four nodes, ownership, Web/PVC/monitoring/S3 acceptance | Four nodes Ready; initial handoff STOP on missing namespace owners |
| Repeated install | Stable UIDs, credentials, ciphertext, Git; no unnecessary mutation | Local intent/credential reuse coverage; live NOT RUN |
| Failure matrix | Artifact, ports, permissions, backup, interruption checkpoints | Actual package rejected manifest/tool corruption, occupied port, unwritable backup and denied Git permission before publication; interrupted preparation reran with the same record |
| User-only procedure | Second person using packaged README without repair commands | NOT RUN |
| Final package | Accepted archive digest equals distributed archive digest | NOT RUN |

## Local validation and lessons

The first remote candidate (`v0.1.0-d1.1`, source `32b5322`) exposed a release
checkout prerequisite before any installation began: Actions' default depth-1
checkout omits the two immutable S1 commits read by `contract.py`. A depth-1
local clone reproduced their absence; fetching full history restored the existing
contract. The release checkout now requests full history. The first run was
cancelled rather than weakening or skipping the frozen regression checks.

The second candidate (`v0.1.0-d1.2`, source `657eb19`) reached the remote
race suite but `internal/ot1` exceeded Go's implicit ten-minute package timeout.
The stack showed active JSON fixture construction, not a reported failed
assertion. Quality now declares a bounded 25-minute package budget for the full
race suite on hosted ARM runners. No tests are skipped and no live-operation
deadline changes. A fresh candidate must pass the entire remote gate.


Final local quality exit code: **0**. This ran Go vet and race tests, all
348-resource Helm/Kustomize/conformance checks, eight Python contract tests,
four frozen layout renders, and compilation/vet of the frozen experiment entry
points. The new private CI tool set was passed explicitly to Task; no cluster
commands were executed.

The tested local development archive was
`atlas-v0.1.0-dev.d3-darwin-arm64.tar.gz`, SHA256
`168b89217a0dafa6b4b4c62382e6a0c795d85d73863f634f69e992c9714b6b82`.
It is explicitly marked `developmentOnly`, not a publishable candidate. Checksums,
manifest/config validation and help worked outside the checkout with only
`/usr/bin:/bin` on PATH; missing prepared state failed without creating state,
and a modified runtime manifest failed before configuration was read.

The local host is macOS 26.7 (25G229), Apple Silicon, OrbStack 2.2.3 (2020300).
These are development measurements, not the independent acceptance support table.
Go 1.27.1 and all runtime/build dependencies are pinned. The newly prepared CI
set contains Helm 4.2.3, Kind 0.32.0, kubectl 1.36.3, kubeseal 0.40.0,
Task 3.53.1, yq 4.53.6 and Lua 5.5.1. Preparation reused already verified Helm
and Kind archives; it is not a clean-machine acceptance claim.

- A slow official download exceeded the single-request timeout. Preparation now
  retains a non-executable partial file, requires an exact HTTP range, and checks
  the entire locked digest before extraction. Interrupted and corrupt-response
  cases are regression tests in `internal/installation/download_test.go`.
- A full Git push can succeed before the local record is saved. Reconciliation
  checks the saved full-publication intent before considering base Bootstrap;
  it does not regenerate credentials or another commit. Covered by
  `TestFullPublishAcknowledgementPrecedesBootstrapReentry`.
- Foreground access needs shared read locks. It now allows multiple services
  while excluding installation, and reports a port-forward process exit.
  Covered by `TestAccessReadersShareButExcludeInstallation` and
  `TestForwardProcess`.

The package does not carry an Atlas checkout, author ciphertexts or Go/Task/
Python/Lua prerequisites. Its 63-file base and 63-file full templates carry
25 digest-locked platform images. These counts describe package contents, not
live resource or service readiness.

The first successful preflight saves measured host versions and capacities.
Final authority evidence includes product/binary/source identities, plan and
Git commit, cluster UID, certificate/ciphertext bindings, per-file render hashes,
and Application UID/Sync/Health. Final PASS evidence has not been produced; the failed installation has a separate
retained authority bundle.

## Owner-selected acceptance

On 2026-09-30 the owner selected the current host and current public repository,
MIT licensing, and deletion of old local clusters. The new deployment branch and
cluster are both named `atlas-d1`; neither adopts an old instance. New private
state is outside the source checkout. The D1-only same-host backup exception is
approved for `/Users/nekoreb/Workspace/01_Vault/atlas-refactor-d1`.

The old `atlas-refactor-test-ot1` four-node cluster (UID
`b80928e0-71fa-450c-b3f3-3db06c6c22a7`) was deleted after checking all 366 S1
manifest entries and its key backup digest. S1 authority evidence and old backup
remain retained. Default kubeconfig was unchanged; no Kind clusters remained.
Private cleanup evidence is in `.state/latest/d1-cleanup/`.

This is a same-host run; second-machine and independent-user installation remain
unproven. Existing Docker layer cache may be reused, so it is not a whole-machine
empty-cache test. MIT is now selected and bundled; third-party grants are unchanged.

The next attempt must use the corrected attested archive, a fresh installation
identity and an explicitly reviewed cleanup/rebuild target. Do not change the
failed installation's product binding or hand-create missing namespaces to bypass
the package defect. Record the same runtime, repeat and failure checks. In particular, interruptions around Kind creation, Root/latch
creation, key backup and consumer publication remain live acceptance cases;
local tests do not prove their complete behavior. Uncertain or contradictory
Bootstrap identity still stops and cannot be repaired by deleting state files.

Promote that same archive only after acceptance; do not rebuild it. Do not call
D1 complete or proceed to S2 on the strength of local tests or an S1 cluster.

Implementation: `internal/installation` coordinates fixed phases;
`internal/atlas` retains Seed/Latch/Receipt and schemas 1–3; `cmd/atlas-release`
builds the complete archive; `cmd/atlas-install` loads authenticated adjacent
runtime data without a source checkout.

## First actual package run: namespace closure failure

- CI source: `4cb635486b847b89ac36e8aefc300a4696a69c8f`; tag `v0.1.0-d1.3`.
- [CI run](https://github.com/snkio027/atlas-refactor/actions/runs/36730610488): all gates PASS;
  hosted ARM OT-1 race suite 983.142 seconds.
- Actual archive SHA256: `e2722020ada9a1bd2a580749e6e913fb544e6bb180dfc25be064672f5477b668`.
- Manifest/binary/internal checksums and GitHub repository/workflow/tag/source
  attestation PASS; MIT LICENSE included. Runtime PATH excluded Go/Task/Lua and
  preinstalled Atlas tools, outside the source checkout.
- Preparation was interrupted during a real tool download, then completed in
  395.85 seconds with the same installation record. Four tools and 25 OCI
  archives verified; existing Docker layers reused. Same-host evidence only.
- Base deployment: `e34b68e8a593608bf516105ac0186581ba80a3fc` on `atlas-d1`.
  Cluster UID: `9080d70e-176b-4048-8bd7-2ba5843da6d2`; four nodes Ready.
- STOP: secrets-controller exhausted sync retries because its Role/RoleBinding
  targets atlas-monitoring and atlas-storage, whose namespace owners were absent
  from the base Application catalog. Parents remained Progressing. The operator
  sent SIGINT to the exact installer after observing this terminal child failure;
  installer exit 1. No full commit, Grafana/S3 credentials or ciphertext publication.
- Cause: D1 retained the full fixed controller payload but selected only the
  secrets-controller activation closure. Dormant Namespace files in the package
  did not make their owners active. The correction activates both domain namespace
  owners before the controller, while keeping consumer applications deferred.
- A reachable-source regression also exposed storage foundation's workload-web
  NetworkPolicy dependency. The D1 core foundation now precedes domain foundations;
  schema 1–3 catalogs and the S1 authority bundle are unchanged.
- Regression: `TestActivePhasePayloadNamespaceClosure` resolves active Application
  sources and checks namespace ownership/order for both phases;
  `TestInstallationHandoffStopsOnCurrentTerminalSyncFailure` and
  `TestInstallationHandoffUnknownReadFailsClosed` cover immediate D1 error reporting
  without new writes or changed historical observation rules.

Private evidence: `/Users/nekoreb/Atlas/d1/evidence/failed-install-authority/`.
The executed archive, plan, deployment bundle, dedicated kubeconfig, live snapshots
and Metadata-only audit are retained. The already generated controller key was
backed up under the approved D1 vault directory as **incident preservation**, not
as a successful installer backup/roundtrip gate. No private material enters Git.
The failed target must not be overwritten or adopted by a different product digest.
