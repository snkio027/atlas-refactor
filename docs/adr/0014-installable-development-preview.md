# ADR-0014: Installable development preview (D1)

Status: Proposed

## Decision and boundary

D1 precedes S2. Ship one darwin/arm64 installation package for an Apple Silicon
Mac with running OrbStack, Git and authenticated access to a dedicated public
GitHub repository with a new dedicated deployment branch. The product and
deployment may share a repository; their commits and roles remain separate.
Never repurpose an existing branch. Fix the topology at one control plane and gateway, compute,
data workers. Install the existing Web/TLS, retained local storage, monitoring,
S3 and Sealed Secrets capabilities; add no controllers or recovery framework.
Support versions and resource minima are acceptance results, not assumptions.

Separate product identity (release, source and package digests), deployment
identity (user repository, branch, path and generated commit), and installation
identity (random installation ID, cluster UID and certificate/credential binding).
The source commit never substitutes for the deployment commit. Preserve schemas
1–3, their frozen snapshots and the completed S1 evidence. Schema 4 is available
only through the bounded installer using authenticated release templates.

The release builder projects two fixed GitOps templates from the checked
platform: a base with Sealed Secrets and a full platform. Neither includes author
ciphertexts or private state. User projection changes only approved Git bindings
and loopback ports. Consumers are published only after independent per-install
credentials have been sealed with this installation's verified certificate.
The base activates the secrets, observability and storage namespace owners
before the fixed sealing-controller payload, which already contains RBAC for
all three consumer namespaces. The core foundation runs at wave -110 before
domain foundations at -100 because storage's NetworkPolicy uses workload-web.
These D1 catalog annotations leave resource ownership and frozen S1 inputs intact;
monitoring/S3 consumers still wait for verified backup and new ciphertexts.
During initial D1 handoff, an idle Failed/Error sync for the exact base revision
returns an error immediately, including when a parent only reports Progressing.
Old-revision terminal records and an already active retry do not prove failure.

Bootstrap creates its finite authority and permanently relinquishes Seed writes
at the existing durable latch. Post-handoff health failure cannot restore them.

One reviewed plan authorizes the exact repository/branch, new cluster, loopback
ports, private state and backup destination, finite writes and generation rules.
Random values are recorded during execution, not reused from another instance.
Changing a bound input invalidates approval. Preparation may download verified
pinned tools and images; execution must not. No global tool replacement, hosts,
system trust or default kubeconfig modification is implicit.

Backup readback, private-key/public-certificate match and an actual local seal /
unseal round trip precede publishing consumers. Record the declared isolation
class; a same-host backup is never represented as physically isolated. Sealed
Secrets scope binds namespace/name; separate keys and verified instance binding
provide cross-installation separation, not a cryptographic cluster-UID claim.

Confirmed steps are not repeated. An ambiguous external request is reconciled
against its saved exact intent before any retry. Inconsistent target, Git, key
or resource identity stops execution. A completed installation verifies without
rotating credentials or creating a gratuitous commit. Concurrency exclusion and
terminal evidence have distinct purposes; no S1 STOP lock is cleared by D1.

## Bounded D1 reconciliation notification

The initial full-platform publication may race Argo's normal status updates.
After Git publication, D1 may request a hard refresh on an existing Application;
it does not own the Application spec or run a reconciliation loop. Keep atomic
UID, resourceVersion and full-spec JSON Patch tests. Check exact deployment Git
and cluster identity before every fresh GET, placing slow Git reads before that
GET to minimize the optimistic-concurrency window.

Classify a rejection from the authenticated API's structured Kubernetes Status,
not kubectl stderr. Only confirmed HTTP 409/Conflict or 422/Invalid may be
re-observed within the same invocation (at most five requests and one minute per
Application). Retry only when resourceVersion changed and the full object is
unchanged except status, managedFields, resourceVersion and a recognized refresh
annotation. A pending normal/hard refresh is coalesced, not replaced. UID, spec,
tracking, labels, deletion or Git/cluster drift stops. Other failures, including
lost responses, are unknown outcomes and never retried.

Persist an exclusive intent and each exact guarded request before its possible
effect. Record definite rejection, acknowledged completion or coalescing after
rejection separately. An intent without confirmed completion cannot reacquire
write permission, even if the Application subsequently becomes healthy. This is
a bounded notification protocol, not automatic recovery or a continuation API.
Existing S1 tooling and historical installation evidence remain frozen.

## Supply chain and completion

A release contains the executable, runtime projections, pinned dependencies,
checksums, provenance, documentation and license notices. GitHub attestation
verification binds the expected repository and release workflow as well as the
artifact digest. The archive used for the selected runtime acceptance is promoted unchanged.
Local builds, simulations and the existing S1 cluster are insufficient for D1 PASS.

On 2026-09-30 the owner selected the current Mac and current public repository
for acceptance, and chose MIT for Atlas code. The first `atlas-d1` instance
stopped before consumer publication and was retired after evidence/backup review.
The successful fresh retry used the independent `atlas-d1-r2` deployment branch
and four-node cluster, new installer state and credentials. Its result is
explicitly same-host acceptance,
not evidence of independent-user or second-machine reproducibility. Existing
Docker image layers may be reused after digest checks; old Atlas runtime state
and keys are not installation inputs.

The owner separately approved the D1-only same-host backup exception at
`/Users/nekoreb/Workspace/01_Vault/atlas-refactor-d1`. Record it as reduced
isolation; do not call it physically isolated or inherit older exceptions.

The selected runtime gate requires the actual package, exact deployment identity,
service access, repeat-install and failure checks, and retained final and Trust
Root evidence. The attested v0.1.0-d1.4 archive passed those selected same-host
checks; exact evidence and unexercised interruption limits are recorded in
[the acceptance record](../d1-validation.md). This ADR remains Proposed and the
archive awaits review/promotion; a runtime PASS does not accept the decision.
Release provenance remains a separate required check.
Independent-machine testing remains unproven and must be named as a limitation,
rather than silently claimed by deleting clusters on the development host.
This decision neither authorizes production cutover nor introduces upgrade,
retirement or recovery capabilities.
