# First disposable integration gate

Status: FAIL at baseline `84ce710a22be2becb8d5a0f87ce3ba2e06dc055f`.
See the [first-run result](integration-20260927-01-result.md). The following is
the retained execution plan for that baseline. ADR-0001 remains Proposed.

## Exact scope

- Repository: `https://github.com/snkio027/atlas-refactor.git` (public).
- Source ref: `integration-20260927-01`; its full commit SHA is recorded in the
  execution approval and evidence. The ref must resolve to that SHA before and
  after the run. No tag movement or source updates during the experiment.
- Profile: `profiles/integration.json`.
- Cluster: `atlas-refactor-test-vs0927`, one IPv4 control-plane node.
- Docker: `orbstack`, owner-local OrbStack socket; API listener: `127.0.0.1`.
- Namespace: `argocd`; External Root: `atlas-refactor-root`.
- No use or modification of old Atlas clusters, credentials or resources.
- Retain the test cluster and private evidence after the run. Deletion is not
  part of this approval.

The generated GitOps manifests reference a dedicated integration tag to avoid
the circular requirement that a commit contain its own SHA. The run approval,
binary build information, source-ref resolution and Argo observed revisions
must all agree on the recorded commit. A differing resolved revision fails the
gate even if health is green.

## Sequence and acceptance criteria

1. Record tool versions and executable SHA-256 values; verify locked artifacts,
   the exact public source ref, clean repository and fresh test target.
2. Run `doctor`, render twice, compare all hashes with committed `gitops/test`,
   and record the resulting binary hash and Git baseline.
3. Run the first approved `apply`. Enable Kubernetes Metadata-only audit records
   for mutation requests; no request/response bodies enter the audit log.
4. Once Root creation succeeds, a test-only kubectl wrapper pauses the return to
   Bootstrap. Send SIGINT to the Go process. Capture the existing Identity,
   handoff latch and Root, and confirm the Receipt has not been created.
5. Resume `apply` using the same source, profile and binary. It may commit a
   Receipt after GitOps readiness; it must not reinstall Seed or recreate Root.
6. Run `status --check`. Verify Root and every child Application resolve to the
   approved source SHA, are idle/Synced/Healthy, and have expected projects and
   source paths. Verify Seed object tracking/managed fields show Argo ownership,
   and Receipt UID bindings match the actual objects.
7. Run a second complete `apply`. Compare control-object UIDs, specs, Receipt,
   and Seed tracking before/after. Kubernetes audit must show zero Bootstrap
   mutation requests in this invocation's window.
8. Exercise the three negative cases below, restore the deliberately changed
   fixture, then require `status --check` to pass again.

## Bounded negative cases

- **Missing artifact:** move the local vendored chart to private evidence
  storage, invoke doctor/apply, require nonzero exit and zero API writes, then
  restore the original bytes and SHA. Never download a replacement at runtime.
- **Configuration drift:** use a private profile copy with a different Git ref.
  Require `status --check` to classify drift and apply to fail before mutation.
  The committed profile and remote tag remain unchanged.
- **GitOps unhealthy:** on this test cluster only, temporarily scale the Argo
  Application Controller StatefulSet to zero. With it stopped, inject a
  Degraded health value into the `argocd-self` status subresource. Run status and
  apply, require degraded/nonzero results and zero Bootstrap mutations, then
  restore the recorded health and replica values with UID/precondition checks.
  Wait for genuine controller reconciliation and readiness to recover.

The unhealthy case is explicitly a live API status fault while the real
controller is stopped. It proves the CLI refuses an unhealthy observation; it
does not claim to test every naturally occurring Argo failure classifier.

## Evidence and reporting

Private `.state/evidence/` holds timestamps, command/exit-code records, local
render and artifact hashes, exact Git SHA, binary/tool hashes, non-secret
resource projections, UID/spec/managed-field comparisons, Application revisions
and health, and filtered Kubernetes audit events. Kubeconfig remains outside
the evidence export. Never export Secrets, private keys or complete environment.

The report distinguishes initial attempt, interruption, resumed adoption,
idempotent repeat and each injected failure. Runtime failures remain failed
gates; fixes must have their own reviewed source identity and cannot be silently
folded into the original baseline. Successful isolated behavior does not imply
production recovery, hostile-administrator resistance, or authority cutover
readiness. ADR acceptance remains a subsequent owner decision.
