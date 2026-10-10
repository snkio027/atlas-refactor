# Ordinary application delivery — local validation

Scope: proposed ADR-0018, single-owner local development. This records development
checks, not Runtime PASS or production approval.

## Executed

- Deterministic static linux/arm64 OCI construction and verification, with rejected
  extra-image/tag aliases, malformed references and damaged closures.
- Ordinary single WebService compilation with and without a Binding; no required
  unbound peer or /roundtrip endpoint; repeated output identical.
- Application purpose rejects the platform acceptance Probe before any effect.
- Provider finalization failure leaves no application success marker.
- Partial/STOP/contradictory attempt blocks replacement by a fresh plan.
- Recorded read-only views cannot approve a mutation; unpublished reviews retain
  the previous deployed view, but partial new attempts cannot hide behind it.
- Workspace symlink/output protection, strict config, escaped history rejection,
  public D1 extraction layout, nested layout, ambiguity rejection.
- Exact target confirmation, irrelevant flags rejected before instance loading.
- Existing Probe replay/terminal tests, image import identity tests and original
  S2 compiler/publication contracts remain part of task quality.

A real experiment-archive binary was also packaged and used by the CLI:

| Binding | Value |
| --- | --- |
| Application source | ed1a13fe3b61431fe195a632e889d4f5025ce7d9 |
| linux/arm64 ELF SHA256 | 529b6b59a2d54a32bfe432f4df84c989b866690f72c273af69c6707275432193 |
| OCI archive SHA256 | 10a1afbbcd4e651f2d584d0222f65f59f0cfe4054495e394526075c3851134d6 |
| Image | atlas.local/experiment-archive:v1@sha256:7461927fe48447076e9c3610075517463281c1767a0126331c662e63c2180dd8 |

Local init/check generated one Project, one WebService and one S3 Binding.
An impossible resource request exited 1 and named the authored file and resources
field. Restoring the input restored successful validation.

A read-only plan against the existing r7 deployment correctly exited 1:
the branch already owns Project demo. No image import, credential use, Git
publication or Kubernetes mutation occurred. This explicitly verifies the
single-Project boundary rather than treating r7 as an arbitrary hosting target.

The final candidate passed the complete task quality after all workspace/status
changes: race tests, 348 GitOps resources, real Helm/Kustomize, eight Python
contract tests and the frozen OT-1 entry points. The source tree passed
go vet and formatting checks. No runtime fixture was enabled.

## r2 runtime result and publication recheck fix

The upgraded `atlas-app-r2` run used implementation
`9271110a975260dde2eb3d94f7dcf906c3773e1a` and approved review plan
`631e6e6f78b7bd960ba2b15f493b713581c27251ad37c6a61ed5b9969dbc134a`.
D1 installation and its independent read-only verification passed. Ordinary v1
published permissions, project and infrastructure and retained all three Ready
Gates. Before consumer publication, a closing-read change in Application
`foundation` correctly produced `Pending`, but the predecessor recheck called a
single-shot observer and returned it as a terminal failure. The attempt is STOP;
consumer had no intent/receipt and no business upload ran. Later convergence does
not change that history. Full private evidence remains in the r2 result bundle.

The fix implements the already accepted ADR-0017 read-wait contract at this
missing boundary: predecessor rechecks use `ObserveFor` with the existing
15-minute limit, bounded by the caller deadline, and one rollout session for UID
and comparison continuity. Only explicit `Pending` reads repeat. Identity/spec/
ownership/Git drift, unavailable reads, cancellation and timeout still stop;
intent creation, Git push and all other effects remain outside the wait. Existing
Ready evidence stays immutable; the latest read records a terminal failure.
No compiler output, publication order, retryable mutation, recovery command or
STOP continuation is introduced.

`TestPublicationRecheckPendingPreservesGateAndUIDHistory` uses compiled source
trees and receipt/Gate fixtures to check Pending → Ready, retained UID history,
API failures, immutable prior Gates and the dependent effect boundary.
`TestPublicationRecheckDeadlineAndCancellationStopBeforeEffects` checks the fixed
budget and cancellation, including a late successful read. These exercise the
shared production read poll with synthetic reads/effects, not a real push.
Existing receipt/unknown-outcome tests continue to reject publication replay.

## r3 completed application acceptance

`atlas-app-r3` completed the reviewed ordinary-business route on
2026-10-10 (UTC), with documented interruptions and usability limits.
The runtime implementation remains `91896f7a3713d36c3f1953d088e1d12e60693640`,
with [exact-head Quality 38071119902](https://github.com/snkio027/atlas-refactor/actions/runs/38071119902).
Total reviewed plan: `f6f8465ea72143dc17cce520e89473c0e6b322f18b07f0d57499644c6eff9d7d`.
D1 used the unchanged attested `v0.1.0-d1.5` package; installation and its
independent read-only verification passed. The runtime binary was never replaced.

| Version | Plan SHA256 | Deployment commit | Deploy duration |
| --- | --- | --- | --- |
| v1 | `5dc0b7c09b7d5dc8b983bb63f67584902145bba74eebcdfba15250da0354d6c7` | `e0eb7884e7fb9d4b33d446194640257c3808e81e` | 1164 s |
| v2 | `ae05a6167af575c904d9912dc45cdb1a0f33d5549152f5d4b5f354f986dcaaf2` | `5d7bb4e37ac014d38ec04e11c3ddc42ae230e5e6` | 196 s |
| v3 | `18123a3326a04f93d99d88c5597475eeab8051dda6d42ab081a24e1fb7762f4f` | `18d517c12f94f535dfbbb71c8f0e9c8865aac300` | 288 s |

Each version reached `DEPLOYED / functional=UNPROVEN`, then independently passed
one business upload, metadata query, download, SHA-256 equality, expected 400 and
expected 404. v2/v3 read prior versions' retained objects; v2's digest headers and
v3's safe Unicode download filename were checked. Three records remain retained.
No ordinary application was required to implement the S2 demo probe interface.

v1 completed four publication phases; v2/v3 were consumer-only updates. Both
update workspaces correctly showed the prior deployed version before publication,
including the new unpublished plan after planning. Authored inputs matched the
reviewed files byte for byte; no generated YAML, Secret, tracking or live spec was
edited. Binding/provider credentials remained byte-identical across both updates.

v3 reached two Ready replicas with requests 150m/192Mi and limits 600m/384Mi per
replica. Two raw Prometheus targets returned `up=1`. New CLI processes could query
the recorded deployment, reopen verified-TLS access and download all three
retained records. Business logs from the exact workload Pods showed expected
400/404 routes. Logs from both Pods were collected explicitly; automatic
multi-Pod aggregation is not claimed.

The three successful repeat deployments took 19, 20 and 40 seconds. Git, final,
credentials, persistent UID/owner/tracking/SSA and Pod UID stayed unchanged within
each repeat. No business/probe intent was added and POST counters stayed unchanged.
Their audit windows contained 53, 56 and 119 events respectively, with zero covered
administrator management writes. Metadata audit excludes GET/streaming: absence
of exec events is not proof of no business effects; independent counters and logs
support that separate statement.

Final read-only checks verified original D1 HTTPS and S3 access, all 145 baseline
UIDs, frozen D1 record/credentials, Bootstrap authority records and External Root
UID/spec. The 164 preserved old files and prior deployment refs were unchanged.
A post-acceptance Docker snapshot showed approximately 5.14 GiB across the four
Kind containers; this is a single development snapshot, not capacity or SLO proof.

The private final evidence index has SHA256 `e2fb0f7321e2cf47afc20b18f290a48b03336ae9afa88fda7159947b3f068e91`.
It binds immutable receipts/finals, functional records, repeat audit windows,
final observations, frozen inputs and a complete-line final audit prefix. Raw
logs, kubeconfigs, private credentials, keys and the private bundle are not part
of this repository report.

### Interruptions and validation boundaries

The private outer recorder initially used the business script's exclusive intent
filename. The script stopped before its first POST. The owner explicitly approved
correction `6e1276617f0e3dfa20052c3826885a639b02c8ea53184341fb403ec879406e73`:
archive the exact mistaken outer marker, retain failure/zero-POST evidence, and
use distinct outer step names. The business script then acquired its own first
intent. No real business intent was replayed and no historical final was rewritten.

After successful v2 deployment, `app open` returned an explicit closing-read
`Pending` before starting a proxy. Its `Current` path uses one observation rather
than a bounded wait. A subsequent status was READY. The owner approved the exact
access retry and original remaining scope under
`bc39145b9a1acb4a1fd92cfaaf2bc9847d88acf100d8a53fd018bfb28f65f45d`.
OrbStack also slept, independently causing API/Docker unavailability; after the
owner woke it, fresh exact UID/Git/final/credential and zero-v2-POST checks passed.
The single approved access retry succeeded. No cluster rebuild, duplicate v2
publication, credential rotation, manual refresh or deadline extension occurred.
The single-shot access Pending behavior remains a documented usability limitation;
this report does not describe an uninterrupted first try.

`app logs` in the frozen runtime also rejected legitimate Kubernetes
`lastProbeTime: null`. The authored no-null decoder had been used on a live List.
Incremental repair `24652edf93b150a0790e0114db13ead79b2d0a00` uses normal JSON
decoding while retaining namespace/UID/ServiceAccount checks and deterministic
Pod choice. Regressions cover nullable ready/unready status, malformed/trailing
JSON, foreign identity and unchanged authored no-null rejection. Full locked
`task quality` passed, including race tests, real Helm/Kustomize and 348 resources;
[exact-head Quality 38077693312](https://github.com/snkio027/atlas-refactor/actions/runs/38077693312)
completed successfully.

A separate read-only integration test invoked the repaired production
`ApplicationLogs` method against r3 with exact UID/v1 plan/config/intent and the
installation read lock. It read real Pod logs successfully. This did not replace
the frozen CLI or relax historical compiler binding. Runtime acceptance remains
bound to 91896f7; the log fix has incremental tests/CI/read-only method evidence,
not a reassigned full runtime run.

### Remaining limits

This is one single-owner local-development application. App platform finals remain
`functional=UNPROVEN`; the independent business records provide business PASS.
S2 r7 Runtime PASS stays bound to d011813. Trust Root backup used the explicitly
approved same-host development exception, not physical isolation or DR proof.
Only the five reviewed ciphertext versions were published. Production readiness,
multi-tenancy, recovery, retirement, cross-version historical-plan compatibility
and general log aggregation are not established. ADR-0018 remains Proposed and
PR #10 remains Draft pending final review; this report does not approve a merge.
