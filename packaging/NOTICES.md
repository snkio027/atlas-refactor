# Preview package notices

Atlas D1 is a development preview. The repository currently has no top-level
license grant; this package does not invent or change one. Public distribution
and the project's intended license must be reviewed before promoting a preview.

The binary statically links the Go standard library (Go BSD license, included).
The runtime manifest includes Helm-rendered Kubernetes resources and the locked
Argo CD chart. The corresponding upstream Apache-2.0 license is included.
Upstream attribution and source locations are recorded below and in the source
repository's pinned artifact locks:

- Argo CD / argo-helm: https://github.com/argoproj/argo-helm
- Cilium: https://github.com/cilium/cilium
- cert-manager: https://github.com/cert-manager/cert-manager
- Envoy Gateway: https://github.com/envoyproxy/gateway
- Gateway API: https://github.com/kubernetes-sigs/gateway-api
- Local Path Provisioner: https://github.com/rancher/local-path-provisioner
- Sealed Secrets: https://github.com/bitnami/sealed-secrets
- kube-prometheus-stack: https://github.com/prometheus-community/helm-charts

Runtime tools and OCI images are downloaded separately from their locked upstream
sources; they are not relicensed by Atlas. Image and tool references/digests are
included in runtime.json. Review upstream notices for those separate artifacts.
