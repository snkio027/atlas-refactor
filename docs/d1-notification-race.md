# D1 notification race — root cause and correction

The original `atlas-app-r1` attempt remains STOP. Full desired state was already
published; a guarded Application refresh failed with HTTP 422. Eventual cluster
health does not convert that attempt into an installation PASS.

The GET/PATCH gap permitted Argo to update status and resourceVersion before the
JSON Patch resourceVersion test. Kubernetes correctly rejected the stale patch.
The isolated reproduction exercised 210 interleavings; dependency upgrades do
not remove this optimistic-concurrency behavior. See the upstream
[Kubernetes JSON Patch handler](https://github.com/kubernetes/kubernetes/blob/v1.37.0/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go).

The correction preserves all mutation guards and uses a dedicated TLS-verified
D1 notification client. A structured, definitive 409/422 rejection permits a
bounded fresh observation. Only benign status/API bookkeeping churn permits
another guarded request. Metadata authority drift, unchanged RV, unknown writes,
missing acknowledgements and exhausted bounds stop. Durable exclusive intents
prevent replay after a lost response. Successful completion and coalescing are
recorded distinctly. The shared kubeconfig transport parser retains the existing
inline-certificate, explicit-loopback requirements; Observation remains GET-only.

Regression coverage includes HTTP outcome classification, actual TLS request
interleaving, unknown write/no replay, UID/spec/tracking/label/deletion fences,
Git/cluster fences, pending refresh coalescing, retry exhaustion, intent and
receipt persistence failures, and repeat invocation. See
`internal/installation/notification*_test.go` and
`internal/kubeconfig/transport_test.go`.

This source correction does not claim live validation. A new clean instance must
bind its candidate, package, target, Trust Root and plans independently. Historical
r7 PASS and app-r1 STOP, their deployment refs and private evidence are retained
when the owner authorizes deletion of their disposable cluster containers.
