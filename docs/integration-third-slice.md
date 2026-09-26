# Seed ownership integration gate

Status: PREPARED, not executed. ADR-0001 remains Proposed.

The [second run](integration-20260927-02-result.md) proved interruption handling,
repeat-call zero writes and bounded failure refusal, but exposed premature
adoption. This baseline adds complete self-sync and live durable Seed ownership
checks. It does not add recovery or other platform features.

- Source ref: `integration-20260927-03`; exact commit recorded before approval.
- Public repository: `https://github.com/snkio027/atlas-refactor.git`.
- Profile: `profiles/integration-03.json`.
- New cluster: `atlas-refactor-test-vs0927c`, one Linux ARM64 Kind control-plane.
- Docker: owner-local `orbstack`; loopback IPv4 Kubernetes API.
- Namespace: `argocd`; Root: `atlas-refactor-root`.
- Separate execution clone: `/private/tmp/atlas-refactor-integration-03`.
- Retain both previous clusters, local target evidence and immutable tags.

Repeat the full [original sequence](integration-first-slice.md#sequence-and-acceptance-criteria)
on this target, including the Root-create interruption, resumed Receipt,
idempotent apply, and all three bounded negative cases with restoration.
The unhealthy fixture uses only a status-field JSON Patch when the CRD does not
expose a status subresource; UID/resourceVersion guards and unchanged-spec
checks are mandatory. No recovery, deletion or original Atlas mutation.

Before Receipt acceptance, require all four Application revisions to equal
the approved SHA and verify ownership of all **39 durable Seed objects**.
The four Helm hook objects have a separate transient lifecycle and are not
subject to stable-UID assertions. Compare durable Seed and control-object
UIDs/specs/tracking before and after the repeated apply. API audit must prove
one Root creation, only Receipt creation on resume, and zero Bootstrap mutation
requests on repeat and each refused failure path. Client dry-run manifest
decoding must also produce zero API mutations.

Keep the cluster and private evidence after the run. Any failed acceptance
criterion remains failed; never infer success from green health, move a tag,
silently replace a binary, or accept the ADR as part of test execution.
