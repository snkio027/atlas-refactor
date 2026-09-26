# First integration result: FAIL

The first real run did not reach GitOps handoff. Go Bootstrap behavioral parity
is **not established**. ADR-0001 remains Proposed.

- Baseline: `84ce710a22be2becb8d5a0f87ce3ba2e06dc055f`.
- Public source tag: `integration-20260927-01`; retained without movement.
- Target: `atlas-refactor-test-vs0927`, owner-local OrbStack, one Kind node.
- First apply: 2026-09-26 16:37:14–16:37:47 UTC, exit **1**.
- Final observation: **FRESH**, `status --check` exit **1**.
- Identity UID: `a3e2bba4-c081-4fe5-ae35-1af69337f212`.
- Argo namespace, Application CRD, handoff latch, Root and Receipt: absent.
- Cluster, private kubeconfig and Metadata-only API audit: retained.

## Failure and implementation corrections

The three locked images existed locally, so doctor passed. The real
`kind load docker-image` then failed because its containerd import requested
`--all-platforms` from a Docker archive with absent platform content:

```text
ctr: content digest sha256:b38d7a16bee1e757d96a99fa44168c4f260b98ca200948cdafa8e2b822995224: not found
```

The error was reproduced on the same approved test node. This is consistent
with the [Kind known issue](https://github.com/kubernetes-sigs/kind/blob/main/site/content/docs/user/known-issues.md#unable-to-kind-load-docker-images).
Original Atlas already avoids this convenience loader in
`bootstrap/registry/local.sh`; its conformance suite prohibits reintroducing it.
That practical constraint was missed in the first Go implementation.

The correction follows the existing Atlas approach: export the locally locked
image, stream the private archive into containerd with an explicit
`linux/arm64` import, verify the original index digest, and restore the exact
locked reference without force-overwriting an existing reference. The archive
is not buffered in Go memory and is removed on success or failure. Doctor now
checks the supported image platform, and node validation checks OS,
architecture and kubelet version before loading Seed images. No runtime pull,
daemon storage reconfiguration or registry was introduced.

Reviewing the next installation step also found a rendering defect: replacing
the substring `data:` matched the end of `metadata:` first, putting
`application.resourceTrackingMethod` into ConfigMap metadata. The correction
anchors the complete top-level `data` key. The live run stopped before applying
this invalid manifest, so this is a source/render finding rather than an
observed Kubernetes installation failure.

Regression tests cover explicit-platform imports, streaming and archive
cleanup, missing image platforms, failed imports, missing target digests,
reference restoration, and the exact ConfigMap data placement. These tests
do not constitute a second successful live run.

## Evidence and remaining gates

The [sanitized evidence index](evidence/integration-20260927-01.json) records
the baseline and binary hashes, tools, all render hashes, cluster identity,
exit codes and the reached phase. Complete private command and API audit
records remain under `.state/evidence/integration-20260927-01/` in the original
checkout; kubeconfig is outside the evidence export.

Missing chart and configuration-drift checks returned nonzero and produced
zero Bootstrap API mutations. The original chart bytes were restored. Both
checks occurred in **FRESH**, before adoption, and cannot substitute for the
requested post-adoption negative tests.

Root interruption/resume, GitOps ownership, exact Argo revision, repeated
apply and unhealthy GitOps behavior were **not reached**. No pod-health or
simulator result is used to mark those gates passed. The corrected source
requires a new immutable baseline and a separately approved test target;
the failed baseline and its evidence remain unchanged.
