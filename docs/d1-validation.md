# D1 acceptance record

Status: LOCAL CANDIDATE; owner-selected same-host runtime acceptance pending. No new live
cluster, Trust Root, ciphertext publication, GitOps push or release has been
executed for D1. S1 remains the last completed live milestone. ADR-0014 remains
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
| Release provenance | Pinned Actions, locked quality, attested archive, draft promotion | Workflow implemented; NOT RUN remotely |
| Independent clean machine | Actual macOS/OrbStack versions, capacities, empty state/cache | Deferred by owner; this run uses the current Mac and fresh state |
| Dedicated deployment | New isolated deployment branch and two generated commits | Selected: snkio027/atlas-refactor, new atlas-d1 branch |
| Live Trust Root | New controller key and declared independent backup destination | Selected: independent D1 directory; same-host exception explicitly approved |
| Installation and services | Four nodes, ownership, Web/PVC/monitoring/S3 acceptance | NOT RUN |
| Repeated install | Stable UIDs, credentials, ciphertext, Git; no unnecessary mutation | Local intent/credential reuse coverage; live NOT RUN |
| Failure matrix | Artifact, ports, permissions, backup, interruption checkpoints | Selected local negatives PASS; full live matrix NOT RUN |
| User-only procedure | Second person using packaged README without repair commands | NOT RUN |
| Final package | Accepted archive digest equals distributed archive digest | NOT RUN |

## Local validation and lessons

The first remote candidate (`v0.1.0-d1.1`, source `32b5322`) exposed a release
checkout prerequisite before any installation began: Actions' default depth-1
checkout omits the two immutable S1 commits read by `contract.py`. A depth-1
local clone reproduced their absence; fetching full history restored the existing
contract. The release checkout now requests full history. The first run was
cancelled rather than weakening or skipping the frozen regression checks.


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
and Application UID/Sync/Health. This evidence has not yet been produced by a
real D1 installation.

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

Run the documented procedure against an attested candidate archive, record the
actual environment, timings/capacity and exact archive digest, and exercise the
failure matrix. In particular, interruptions around Kind creation, Root/latch
creation, key backup and consumer publication remain live acceptance cases;
local tests do not prove their complete behavior. Uncertain or contradictory
Bootstrap identity still stops and cannot be repaired by deleting state files.

Promote that same archive only after acceptance; do not rebuild it. Do not call
D1 complete or proceed to S2 on the strength of local tests or an S1 cluster.

Implementation: `internal/installation` coordinates fixed phases;
`internal/atlas` retains Seed/Latch/Receipt and schemas 1–3; `cmd/atlas-release`
builds the complete archive; `cmd/atlas-install` loads authenticated adjacent
runtime data without a source checkout.
