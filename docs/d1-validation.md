# D1 acceptance record

Status: **OWNER-SELECTED SAME-HOST RUNTIME PASS**. Engineering review and release
promotion remain pending; ADR-0014 is Proposed. The actual attested CI archive
completed a fresh installation, independent verification, supported access and
repeat-install checks. No manual cluster repair was required.

The owner explicitly selected the current Apple Silicon Mac and current public
repository, MIT licensing, old-cluster cleanup and an independent D1 backup
subdirectory with a same-host development exception. This result does not prove
second-machine, independent-user or empty-Docker-cache installation. S2 and
production cutover are outside this change.

## Exact accepted runtime artifact and target

| Identity | Value |
| --- | --- |
| Candidate | `v0.1.0-d1.4`, darwin/arm64, Go 1.27.1, `developmentOnly=false` |
| Product source | `6f22e66c00841d605f70eda8c0ea29b8ecd437df` |
| Archive SHA256 | `e24077b4354f07634be441e2abf3caad2a441a7856a25638fbd2abb044378028` |
| Binary SHA256 | `6e0cb9930633d1f4f834a26b0e1bb1cd708a80d0bc1063f218a155104bb00865` |
| Product SHA256 | `ce28f76918b7b8d7be1dbfbf1d5e79911c6dfaf93ef00819965ff10863f09aec` |
| Plan SHA256 | `f76d2a10c232ec2f7a959a8895bbf7a1b31a597c3c95ea89399298525f1ef47f` |
| Installation ID | `facc0ecfb6484e814b92313d42324cf0` |
| New cluster | `atlas-d1-r2`, one control plane and gateway/compute/data workers |
| Cluster UID | `00d80d67-dde2-4941-abb7-86aa2c257eb4` |
| Public deployment branch | `snkio027/atlas-refactor`, `atlas-d1-r2` |
| Base deployment | `59bb45c3a6d98215766b5221cace8e31bad31075` |
| Full deployment | `70147b26fddb356bfed6f1e3270baf0f35a855df` |
| Certificate SHA256 | `4747a0b5f7fc8516deb3d8f0a9c668f1f48ee9c16b50fdc5d5ae2a0c8f375302` |
| Sealed payload SHA256 | `bc4b90055b2aef4282457cd21bdb4b72d292b9adcd4b9d51dbe3a04ba4c63c21` |

[CI run 36739765209](https://github.com/snkio027/atlas-refactor/actions/runs/36739765209)
passed in 18m14s. Archive checksum, all ten packaged files and GitHub attestation
were verified against the exact repository, release workflow, tag and source SHA.
The package ran outside the checkout with only Git, gh, Docker/OrbStack and system
utilities on PATH; Go, Task, Lua and preinstalled Atlas runtime tools were absent
from that PATH. A Python harness measured results; the installer did not call it.

## Runtime gates

| Gate | Observed result |
| --- | --- |
| Finite Bootstrap authority | PASS: Identity/Latch/Receipt, Seed adoption and four Ready nodes; no manual namespace/tracking/Seed repair |
| GitOps | PASS: exact specs and 26 Applications at full commit, idle, Synced/Healthy |
| Trust Root | PASS: fresh key, 0600 backup readback, key/certificate match and actual seal/unseal before consumer publication |
| Credential publication | PASS: exactly grafana-admin, seaweedfs-auth and s3-client strict namespace/name SealedSecrets; no plaintext credentials/private key in the generated Git tree |
| Web and storage | PASS: approved HTTP-to-HTTPS redirect, CA-verified HTTPS content, Pod placement, required PVCs Bound and PV Retain/claim-UID binding |
| Monitoring | PASS: authenticated Grafana dashboards, anonymous rejection, required healthy targets/four-node metrics, Watchdog rule and delivery to Alertmanager |
| Alert functional probe | PASS: per-install test alert posted, observed, and resolution request accepted; no external notification-channel claim |
| S3 functional probe | PASS: authorized bucket access, anonymous/cross-bucket denial, object write/read/delete, presigned GET and multipart upload |
| Independent verify | PASS / exit 0, 5.62 seconds |
| Foreground access | PASS: Web, Grafana, Prometheus, Alertmanager and S3 concurrently reachable through product commands; all five closed with exit 0 |
| Repeated install | PASS / exit 0, 4.17 seconds; 174 checked persistent object UIDs, Git, credentials, ciphertext, CA, kubeconfig and final evidence unchanged |
| Repeated-install audit | PASS: zero resource writes by certificate-bound installer user kubernetes-admin; blocking Metadata audit covers create/update/patch/delete/deletecollection |
| Host configuration | PASS: default kubeconfig and hosts unchanged; system trust was not modified |

The install ran from 2026-09-30 16:34:03 UTC to 16:51:31 UTC, **1,048.47 seconds**.
Fresh tool/image preparation took **209.57 seconds**. During initial convergence,
Gateway API briefly reported Degraded; its CRDs and admission policy were valid,
and normal Argo comparison/retry recovered without operator writes. No health
predicate was relaxed. The installation continued through the complete functional
probe and wrote its final evidence before marking itself complete.

The successful run did not inject interruptions around Kind creation, Root/latch,
key backup or full publication. Those live interruption checkpoints remain
unproven; targeted local publication/credential regressions and the earlier real
interrupted preparation are separate evidence. A current-host PASS does not erase
that limit or establish automatic recovery support.

## Measured environment

| Fact | Measurement |
| --- | --- |
| Host | Apple Silicon, macOS 26.7 (25G229), 32 GiB RAM |
| OrbStack | 2.2.3 (2020300) |
| Docker | 29.4.0; 10 CPUs and 16,819,609,600 bytes reported VM memory |
| Disk at plan | 342,899,437,568 bytes available / 994,610,155,520 total |
| Runtime tools | Helm 4.2.3, Kind 0.32.0, kubectl 1.36.3, kubeseal 0.40.0 |
| Kubernetes / Argo CD | 1.36.1 / 3.5.1 |
| Local ingress | 127.0.0.1:8080 / 8443 |

These are observations, not minimum requirements. Existing digest-verified Docker
layers were reused. No old installation record, kubeconfig, key or ciphertext was
an input. The base/full package projections each contain 63 files and lock 25
images; the published full Git tree contains 59 files under gitops/ only.

## Failure checks and lessons

The same D1.4 archive rejected a modified manifest, damaged runtime tool, occupied
ingress, repository without push permission and unwritable isolated backup
fixture before publication. The tool was restored byte-for-byte; no Trust Root
backup directory was altered for a negative test. D1.3 preparation was interrupted
during a real download and rerun with the same installation identity and complete
digest verification. It is not represented as a new D1.4 interruption run.

| Finding | Durable lesson / regression |
| --- | --- |
| Release shallow checkout omitted frozen S1 commits | fetch-depth 0; preserve historical contract checks (`657eb19`) |
| Hosted ARM race suite exceeded Go's default ten-minute timeout | explicit bounded 25-minute package budget; no skipped checks (`4cb6354`) |
| First live base omitted namespace owners required by fixed controller RBAC | activate observability/storage foundations and order core before domain consumers; TestActivePhasePayloadNamespaceClosure (`6f22e66`) |
| Parent Progressing hid terminal descendant sync failure | D1-only read guard; TestInstallationHandoffStopsOnCurrentTerminalSyncFailure and TestInstallationHandoffUnknownReadFailsClosed (`6f22e66`) |
| Lost full-publication acknowledgement could reenter base Bootstrap | reconcile saved exact intent first; TestFullPublishAcknowledgementPrecedesBootstrapReentry |
| Access readers and installer exclusion | shared read locks and connection termination; TestAccessReadersShareButExcludeInstallation / TestForwardProcess |

Local locked task quality and remote quality passed, including Go vet/race, real
Helm/Kustomize/conformance, frozen S1 contracts and experiment entrypoints. CI's
installation race package took 378.448 seconds; OT-1 took 947.907 seconds.

The first live D1.3 target atlas-d1 stopped before consumer credential generation
or publication. Its UID was 9080d70e-176b-4048-8bd7-2ba5843da6d2 and base commit
was e34b68e8a593608bf516105ac0186581ba80a3fc. It was deleted only after exact
Git/Root/Identity/Latch, audit continuity, absent PVC/consumer state and verified
incident key backup checks. Its 19-entry failure bundle, archive, Git branch and
backup remain retained. The earlier S1 cluster was separately retired after all
366 evidence entries and its backup were checked; S1 evidence is unchanged.

## Retention and release decision

Final private bundle: /Users/nekoreb/Atlas/d1-r2/evidence/final-authority/.
Its 47 entries were read back and hashed; MANIFEST.json SHA256:
`ff66daa09330d0f78b2e239883ce7af1dce412e667e57838a8dd8715cefa0b55`.
It binds archive/provenance, plan and owner decision, Git bundle, runtime snapshots,
authority records, blocking audit, functional install result, independent verify,
access, repeat checks and negative tests. No private values are committed here.

The independent backup is under the approved D1 vault root / installation ID.
Its receipt declares same-host-development-exception and records the round trip;
this is not physically isolated backup or a disaster-recovery exercise.

The candidate remains a draft pending owner review. Promote only the archive with
the digest above; do not rebuild or replace it. The immutable archive's embedded
README was written before acceptance; this record supplies its subsequent runtime
result. Second-machine/user validation, unexercised live interruption checkpoints,
upgrade/recovery and production readiness remain outside the proven result.
