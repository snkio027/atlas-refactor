# ADR-0002: Propose Go as the default Atlas control-plane implementation language

- Status: Proposed
- Date: 2026-09-27
- Deciders: repository owner and required CODEOWNERS
- Supersedes: none
- Runtime authorization: none
- Scope: language selection; not Bootstrap replacement

## Context

The independent Go implementation has demonstrated a bounded disposable
Bootstrap loop: one Root instantiation, interrupted handoff resumed by Receipt
creation only, all 39 durable Seed resources checked for Argo reconciliation,
zero Bootstrap API mutations on repeated apply, and fail-closed negative paths.
Implementation `0ef918801dac84ac913416f4c1756bf08045a485` and report
`9df41568bafcaa98d6548d2089766d83081c3374` are bound by the
[baseline manifest](../evidence/bootstrap-baseline-20260927.json).

This supports technical feasibility. It does not establish Shell behavioral
parity, existing-cluster migration, production protection or rollback readiness.
The [audit](../bootstrap-behavioral-parity-audit.md) records those distinctions.

## Decision proposed

Choose Go as the default for new first-party Atlas control-plane programs and
for a future Bootstrap successor. This is a default for reviewed implementation
work, not permission to rewrite every script, Operator or recovery component.
Third-party controllers retain their own implementations. Small build and
administrative scripts may remain Shell when they do not become a second engine.

Go offers typed configuration/state and explicit error handling, a standard
library suitable for this boundary, a testable subprocess interface, context
cancellation, race testing and a distributable executable. The completed slice
shows these mechanisms can preserve a bounded authority handoff. It does not
prove that language choice alone prevents authority bugs or improves performance.

Keep the initial dependency surface at the standard library. Retain reviewed,
locked Helm/Kind/kubectl boundaries until a separate change justifies replacing
them. Introducing client-go or another module requires supply-chain review;
moving to an SDK must preserve authentication, request and failure semantics.

Preserve Definition → Instantiation → GitOps Reconciliation → Runtime
Reconciliation, External Root independence, canonical trust boundaries and
projects, Helm-as-renderer, offline artifact verification and separate recovery.
Go does not change any existing architecture invariant or authorize a new tier.
The experimental latch/Receipt semantics in ADR-0001 are not accepted by
accepting this language proposal; the original ADR-0002 discrepancy must be
resolved independently before migration.

## Supply-chain obligations

Treat the compiler, standard library, modules and published binary as supply
chain inputs, alongside the existing tools/charts/images. The current local Go
hash and standard-library-only build are useful evidence, not release provenance.
Before distribution for authority cutover, complete G3 of the
[Cutover Contract](../go-bootstrap-cutover-contract.md): verified distribution
inputs, dependency/SBOM inventory, approved vulnerability handling, offline
build/runtime acquisition constraints, controlled repeat builds, provenance and
fallback artifact verification. Future Go/module upgrades must be independently
pinned and reviewed; no automatic toolchain download in build or runtime paths.

## Alternatives and consequences

Retaining Shell has the smallest immediate migration cost and remains the
current Atlas authority. Go is proposed for clearer state/error boundaries and
maintainability as the control surface grows, with compiler/release custody and
contract migration as explicit added costs. A second systems-language prototype
has no demonstrated need for this milestone. Maintaining Shell and Go as two
permanent mutating Bootstrap implementations would duplicate authority and
contract maintenance; this proposal instead selects a successor direction.

Shell remains selected for existing Atlas until a separately accepted
superseding decision and cutover Gates authorize replacement. Rollback is a
checkpoint-bound migration concern, not an assumed ability to rerun any old
Shell release against a new identity schema.

## Acceptance boundary

Accepting this record decides the default language only. It does not accept
ADR-0001 wholesale, authorize importing old state/credentials, accept the
cutover proposal, or permit Tier-0 writes. In this independent repository it
cannot supersede `snkio027/atlas` ADR-0001. The authoritative Atlas repository
must review and accept its own superseding language decision before its engine
changes. No such acceptance is claimed here.

Owner/CODEOWNER language review is pending. Status remains Proposed.
