# Atlas Refactor contributor contract

This is an independent Go implementation, not the authoritative engine of the
original Atlas repository. Read `docs/architecture.md` and `docs/adr/` first.
Design proposals and implementation status must remain distinguishable.

- Inspect existing changes; preserve unrelated work.
- Keep normal Bootstrap, recovery, and drill capabilities separate. This first
  milestone implements normal Bootstrap only, for disposable test clusters.
- No plaintext credentials, kubeconfig, private keys, or generated Secret data
  may enter Git. `.state/` is local private state.
- Parse configuration strictly as data. Never source or evaluate configuration.
- Treat unavailable reads and unknown states as failures, not absence.
- Normal Bootstrap must not overwrite a drifted External Root or regain Seed
  authority after the durable handoff latch exists.
- Pin external tool versions and image digests; verify vendored artifacts.
  Runtime commands never download dependencies or execute recovery commands.
- Run `task quality` for changes and real rendering verification when rendering
  changes. Record which tests actually ran; simulated success is not live proof.
- Cluster creation, Tier-0 writes, credential use, and cleanup require a
  separately reviewed exact target and explicit user authorization. Ordinary
  build/test work does not authorize those operations.
- Do not import old Atlas live state, credentials, approvals, or cluster names.
  Do not alter the old repository as part of work here.
- Review changes to authority, configuration, deployment scope, or recovery in
  a dedicated ADR. Initial design ADRs remain Proposed pending owner review.

No production-readiness claim is permitted until admission protection,
independent recovery, runtime integration, and release supply-chain verification
have been designed, implemented, and exercised.
