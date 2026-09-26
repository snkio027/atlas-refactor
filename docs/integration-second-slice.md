# Corrected disposable integration gate

Status: PREPARED, not executed. ADR-0001 remains Proposed.

The [first attempt](integration-20260927-01-result.md) failed before Seed or
Tier-0 handoff. This second baseline contains only the resulting offline image
import and ConfigMap rendering corrections, regression tests, and the new
test identity. It retains the first baseline's acceptance criteria.

- Public repository: `https://github.com/snkio027/atlas-refactor.git`.
- Ref: `integration-20260927-02`; full SHA is recorded before authorization.
- Profile: `profiles/integration-02.json`.
- New target: `atlas-refactor-test-vs0927b`, one Linux ARM64 Kind control-plane.
- Docker context: `orbstack`; owner-local socket; IPv4 API bound to loopback.
- Argo namespace: `argocd`; External Root: `atlas-refactor-root`.
- Execution checkout: a separate local clone at
  `/private/tmp/atlas-refactor-integration-02`, with its own private `.state/`.
- Preserve the first attempt's cluster, kubeconfig, audit and source tag.
- No deletion, old Atlas mutation, credential reuse, or recovery operations.

Use a separate clone because the first checkout's `.state/` is bound to the
first cluster. Do not rename, replace or delete that evidence to reuse it.
The clone must be clean at the exact approved SHA; the anonymous remote ref,
build information and every Argo observed revision must match that SHA.

Execute the [original sequence and negative cases](integration-first-slice.md#sequence-and-acceptance-criteria)
on this new target: doctor, two matching renders, initial apply interrupted
immediately after Root creation, observation-only resume and Receipt, explicit
handoff/ownership verification, a complete repeated apply, then missing chart,
configuration drift and the bounded unhealthy-status test with restoration.
All failure tests run again after adoption, regardless of the earlier FRESH
checks. Metadata-only audit must distinguish Bootstrap from fault injection.

Retain the second cluster and evidence after the test. A new source change or
unresolved runtime failure fails this gate; never move either integration tag
or silently substitute another binary. ADR acceptance and authority cutover
remain separate decisions.
