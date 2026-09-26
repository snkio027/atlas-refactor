# ADR-0001: Establish an independent Go Bootstrap foundation

- Status: Proposed
- Date: 2026-09-26
- Deciders: repository owner
- Supersedes: none
- Runtime authorization: none

## Context

The owner requested a new sibling repository, `atlas-refactor`, to absorb the
strong documentation and practical lessons of Atlas and implement a Go-native
Bootstrap loop. The current delivery scope is local implementation and tests;
the owner has since authorized creating the public remote and preparing a
bounded disposable integration run. Exact live targets remain separately gated.

The reference is `snkio027/atlas` at
`aca4ff137a1d254cfeceaec24526e0699b585e92`. Its ADR identities and runtime
approvals are not inherited. This record does not supersede old Atlas ADR-0001.

## Decision proposed

Use Go as the default implementation language and begin with the standard
library only. Use strict JSON configuration, explicit typed lifecycle states,
and an injectable subprocess boundary. Initially retain locked Helm, Kind and
kubectl tools rather than simultaneously changing every API mechanism.

The first target is an owner-local, disposable, IPv4-only, single-node Kind
cluster on OrbStack. Its name and local target state are distinct from Atlas.
Go builds for other OS/architecture combinations do not grant runtime support.

Preserve the separation of Definition, Instantiation, GitOps Reconciliation,
and Runtime Reconciliation. Use an External Root, a minimal macro DAG, an
explicit AppProject boundary, deterministic rendering and shared Seed/self
manifests. Do not introduce Helm release state or plaintext credentials in Git.

Use an immutable create-only handoff latch before Root creation to revoke normal
Seed authority. Confirm GitOps readiness and tracked Signal provenance before
committing a create-only Receipt. The normal path may not overwrite Root drift,
repair missing adopted resources, or infer freshness from unavailable evidence.

Separate local review, implementation testing, and live authorization. Cluster
creation and initial Tier-0 instantiation require explicit target approval.
Runtime commands never acquire missing tools or images. Missing artifacts stop
execution; Git reconciliation itself still requires access to the configured
source, and the first apply verifies its published revision.

## Consequences and limits

The project can test a complete Bootstrap lifecycle without taking over old
Atlas resources or preserving its CLI and Shell layout. Minimal dependencies
make the initial implementation buildable offline with a preinstalled compiler.

The latch and Receipt are conventions enforced by this CLI, not Admission
protection against administrators. There is no cross-host concurrency guarantee,
recovery binary, production support or private Git credential workflow yet.
Strict identity binding also means changes to configuration or lock inputs do
not imply a supported in-place upgrade.

## Alternatives

- In-place Shell-to-Go migration: not selected because the owner requested a
  fresh repository and independently designed interfaces.
- Full translation of Recovery, Drill and future Operators: deferred until a
  concrete capability requires those components.
- client-go immediately: deferred to keep the first behavioral model separate
  from a simultaneous authentication/discovery/request-semantics rewrite.
- One general privileged executable: rejected; future recovery capability
  must remain unreachable from normal Bootstrap.

## Verification and review

The implementation is experimental evidence for this Proposed decision.
Owner review is required before accepting the architecture or authorizing live
operations. Unit and simulated integration contracts cover effect order,
idempotency, forbidden writes, drift, unavailable reads and interrupted handoff.
Real locked-Helm rendering checks determinism and Seed/self parity.

Runtime closure additionally requires a published reviewed Git source and an
explicitly approved disposable-cluster run. Production scope requires separate
recovery/admission design, exercised failure recovery, and verified compiler,
dependency and release distribution provenance.

## Integration verification preparation

The proposed disposable substrate now includes Metadata-only API mutation
auditing, with no request/response bodies. Its profile version is bound into
cluster identity. This supplies evidence for the first integration gate; it
does not activate a recovery or production capability. See
[the integration plan](../integration-first-slice.md). Status remains Proposed.

## Correction after the second live gate

The second baseline produced healthy Applications and a Receipt while 38
ordinary Seed objects remained untracked. Tighten this proposal's adoption
evidence: self-sync must cover matching Seed objects, all four Applications
must report the resolved commit, and every durable rendered Seed object must
have its exact Argo tracking ID and Argo SSA manager before Receipt commitment
or an ADOPTED report. A Receipt cannot mask later ownership loss. Ephemeral
hooks are not durable ownership evidence. This enforces the proposed one-way
handoff; it does not restore Seed authority, accept the ADR or authorize a new
runtime target. See the [second-run finding](../integration-20260927-02-result.md).

## Correction after the third live gate

The third observer assumed that every durable resource receives an annotation
and that cluster-scoped tracking uses an empty namespace. Argo CD 3.5.1 uses
the Application destination namespace for cluster-scoped tracking, and its
manifest generator deliberately does not inject CRD tracking metadata.
The run therefore stayed HANDOFF_PENDING despite a completed self-sync.

All 39 durable resources remain subject to adoption verification. For the 36
non-CRD objects require the exact Argo tracking ID and Argo SSA fields. For each
of the three CRDs require no contradictory tracking, Argo SSA ownership of spec,
a current Synced inventory entry with exact identity/version, and a successful
self-sync result at the resolved commit containing that CRD. Neither existence,
health alone nor a manager name alone satisfies the CRD gate. This demonstrates
reconciliation evidence, not exclusive or admission-enforced CRD ownership.

This corrects evidence interpretation; it does not change control ownership or
restore Seed permissions. A read-only probe and regression tests do not replace
a fresh full integration run. Status remains Proposed, pending owner review.
See the [third-run finding](../integration-20260927-03-result.md).
