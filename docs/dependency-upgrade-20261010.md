# Repository dependency candidate — 2026-10-10

This change updates the repository's current product inputs and build tools.
It starts from `f658affeedc51097f884ec234fd257ee808b4a79` and does not deploy them.
Existing D1/S1/S2 installation records and runtime results retain their original
implementation, toolchain and image bindings. PR #10's first-user runtime gate
remains incomplete.

## Selected versions

Stable release metadata was checked on 2026-10-10. Backports with a newer
publication date do not supersede a newer stable minor release. RCs were excluded.

| Dependency | Previous | Candidate |
| --- | --- | --- |
| Go | 1.27.1 | 1.27.2 |
| Helm | 4.2.3 | 4.3.0 |
| Kind | 0.32.0 | 0.33.0 |
| Kubernetes node | 1.36.1 | 1.37.0 — approved artifact exception |
| kubectl / bundled Kustomize | 1.36.3 / 5.8.1 | 1.37.1 / 5.8.1 |
| Task | 3.53.1 | 3.54.0 |
| yq | 4.53.6 | 4.54.1 |
| Lua | 5.5.1 | 5.5.1 — current |
| Argo CD / chart | 3.5.1 / 10.3.3 | 3.5.4 / 10.10.2 |
| Redis | 8.6.4-alpine | 8.10.2-alpine |
| Cilium / chart | 1.20.2 | 1.20.2 — current |
| cert-manager / chart | 1.21.2 | 1.21.2 — current |
| Envoy Gateway / chart / egctl | 1.9.1 | 1.9.2 |
| Envoy proxy | distroless-v1.39.1 | distroless-v1.39.3 |
| Gateway API standard CRDs | 1.6.1 | 1.6.3 |
| Envoy ratelimit | 8fe6ea42 | 0482748e — Gateway 1.9.2 chart selection |
| Kind local-path provisioner | v20260521-9fb22683 | v20260820-69b56db7 — Kind 0.33.0 selection |
| Kind local-path helper | v20260131-7181c60a | unchanged — Kind 0.33.0 selection |
| Sealed Secrets / kubeseal / chart | 0.40.0 / 0.40.0 / 2.20.0 | unchanged — current |
| SeaweedFS | 4.47 | 4.48 |
| kube-prometheus-stack chart | 91.7.0 | 92.3.0 |
| Grafana | 13.2.2-distroless | 13.2.3-distroless |
| Grafana sidecar | 2.11.2 | 2.14.1 |
| Prometheus Operator / config-reloader | 0.94.1 | 0.94.1 — current |
| Prometheus | 3.15.0-distroless | unchanged — current |
| Alertmanager | 0.34.1 | unchanged — current |
| node-exporter | 1.12.1-distroless | unchanged — current |
| kube-state-metrics | 2.20.0 | unchanged — current |
| Thanos value, not installed | 0.41.0 | 0.42.4; fallback still removed from rendered Operator |
| BusyBox | 1.38.0 | unchanged — approved stability exception |
| actions/checkout | v5 commit | v7.0.1 commit |
| actions/setup-go | v6 commit | v7.0.0 commit |
| actions/attest | v4 commit | same commit, identified precisely as v4.2.2 |

Upstream chart archives remain unmodified, including their bundled subcharts and
Chart.lock. Explicit values select the listed runtime images. Host installations
of Docker/OrbStack, Git, gh, Python and global tools are outside this repository
change. Go continues to have no external module dependencies.

## Exceptions and compatibility

- The latest Kubernetes patch is 1.37.1, but its official `kindest/node` image was
  absent. The owner selected Kind 0.33.0's published 1.37.0 node and kubectl 1.37.1.
  The node's verified multi-platform digest is
  `sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5`.
- BusyBox explicitly calls 1.38.0 unstable; its newest release explicitly labelled
  stable is 1.36.1. The owner selected retention of the previously tested 1.38.0.
  It must not be described as an upstream stable release.
- Cilium 1.20.2 and the Envoy Gateway 1.9 series compatibility matrices list
  Kubernetes through 1.36, not 1.37. Latest individual releases do not establish
  compatibility of their combination. A new exact-target runtime review and
  validation are still required before claiming the upgraded product works live.
- The dependency-only commit does not fix the installer's `notify` GET/PATCH
  concurrency race. The follow-up correction is documented in
  [the D1 notification failure note](d1-notification-race.md); version changes
  alone are not evidence that the app-r1 STOP cause disappeared.

The scope registry was re-derived from Kubernetes 1.37.0 OpenAPI list operations;
namespace-scoped endpoints take precedence over all-namespaces list endpoints.
It contains 93 GVKs (previously 88): 11 added and six obsolete beta/alpha versions
removed; no retained GVK changed scope. OpenAPI provenance SHA is recorded in the
registry. This classifies scope, not API enablement or admission behavior.

## Preserved boundaries

The rendered upgrade keeps resource identities and ownership, AppProjects,
External Root, namespace boundaries and RBAC rules. The public
[input/diff inventory](evidence/dependency-upgrade-20261010.json) records each
changed component's object counts, additions/removals and RBAC changes. Changes
include upstream CRD schema fixes, monitoring rules and workload templates.
Kind's new local-path source widens helper-pod taint tolerations to NoSchedule
and NoExecute; retained-volume policy and controller RBAC stay unchanged.

Historical schema-3 snapshots are not regenerated. Running that profile against
changed dependency inputs must continue to fail closed; reproduce it using its
original commit. Current product rendering uses `task platform:render`, while
D1 generates the adoption signal from the new installation's identity.
The two real OT-1 historical render tests use Helm 4.2.3, kubectl 1.36.3 and
yq 4.53.6 in a separate directory. CI reads their checksum locks from immutable
already-merged `697ebf0` and prepares them under `atlas-ci-tools/ot1`. They are frozen test
inputs, not current product dependencies or installation-package tools. Set
`OT1_TOOLS=/absolute/path/to/historical-tools` for `task quality`; the local
default is `.state/tools/ot1`. Tests reject accidental reuse of current tools.

The renderer selects exactly one component archive from the checksum-verified
lock, so historical inputs still render with their historical archives. Missing,
ambiguous, prerelease or floating archive selections fail closed. This removes
version literals from render code without fetching or choosing a newer version.

Simulator tests bind only their temporary synthetic snapshots to current inputs;
the repository snapshot digest remains independently tested and unchanged.

Chart paths accept exact patch-version Argo archives under `vendor/charts/`, with
the existing mandatory checksum verification. Paths, other charts, floating tags
and prerelease names remain rejected. This permits both historical products and
new release packages without a hardcoded chart-version branch in the engine.

## Validation

| Local check | Result |
| --- | --- |
| `task quality` | PASS: format, vet, race tests, platform checks, OT-1 contracts and frozen entry-point build/vet |
| Current Helm renders + `platform:check` | PASS: 348 resources; hashes, deterministic output, project boundaries, CRD schemas and Kustomize |
| Frozen OT-1 real renders | PASS with separate historical tools; original snapshots unchanged |
| `task build` + `task build:matrix` | PASS: Darwin arm64, Linux amd64/arm64 Bootstrap targets |
| Real Web ELF → OCI twice | PASS: byte-identical archives |
| Local D1 development package | PASS: build, internal checksums and current Envoy image |
| Remote CI / cluster runtime | NOT RUN |

The local D1 archive is a dirty-tree `-dev` build sanity check, not an attested
release or runtime acceptance candidate. Its digest and the other local results
are in the public inventory. One stale EnvoyProxy image reference was rejected by
`platform:check` and corrected to the locked digest before the final package was
rebuilt; that check remains strict.

No cluster operation, deployment publication, credential use, STOP cleanup or
global tool replacement is part of these checks.

Artifact checks verify publisher SHA256 for tool/chart/release downloads. Image
checks hash registry manifest bytes and verify the linux/arm64 child manifest
and config. Tagged source inputs have recorded content hashes. These checks do
not claim image signature verification, remote CI success or runtime acceptance.

## Official sources

- [Go downloads](https://go.dev/dl/), [Helm 4.3.0](https://github.com/helm/helm/releases/tag/v4.3.0),
  [Kind 0.33.0](https://github.com/kubernetes-sigs/kind/releases/tag/v0.33.0),
  [Kubernetes 1.37.1](https://github.com/kubernetes/kubernetes/releases/tag/v1.37.1).
- [Argo CD 3.5.4](https://github.com/argoproj/argo-cd/releases/tag/v3.5.4),
  [Argo chart 10.10.2](https://github.com/argoproj/argo-helm/releases/tag/argo-cd-10.10.2),
  [Redis 8.10.2](https://github.com/redis/redis/releases/tag/8.10.2).
- [Cilium matrix](https://docs.cilium.io/en/stable/network/kubernetes/compatibility/),
  [Envoy Gateway 1.9.2 tagged matrix](https://github.com/envoyproxy/gateway/blob/v1.9.2/site/content/en/news/releases/matrix.md),
  [Gateway API 1.6.3](https://github.com/kubernetes-sigs/gateway-api/releases/tag/v1.6.3).
- [Monitoring chart 92.3.0](https://github.com/prometheus-community/helm-charts/releases/tag/kube-prometheus-stack-92.3.0),
  [SeaweedFS 4.48](https://github.com/seaweedfs/seaweedfs/releases/tag/4.48),
  [BusyBox release labels](https://busybox.net/).
