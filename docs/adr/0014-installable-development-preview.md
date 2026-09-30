# ADR-0014: Installable development preview (D1)

Status: Proposed

## Decision and boundary

D1 precedes S2. Ship one darwin/arm64 installation package for an Apple Silicon
Mac with running OrbStack, Git and authenticated access to a dedicated public
GitHub repository. Fix the topology at one control plane and gateway, compute,
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

## Supply chain and completion

A release contains the executable, runtime projections, pinned dependencies,
checksums, provenance, documentation and license notices. GitHub attestation
verification binds the expected repository and release workflow as well as the
artifact digest. The archive accepted on the second machine is promoted unchanged.
Local builds, simulations and the existing S1 cluster are insufficient for D1 PASS.

D1 remains incomplete until the published acceptance matrix passes using that
same archive on another supported clean machine, independent Git and credentials,
with only public documentation and no maintainer repair commands. This decision
neither authorizes production cutover nor introduces upgrade/retirement/recovery.
