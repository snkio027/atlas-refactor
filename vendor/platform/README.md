# Development platform inputs

These are unmodified, fixed-version upstream inputs. Rendering never downloads
charts, CRDs or images. Provenance and SHA-256 are recorded in
`platform/development/versions.lock.json`.

- Cilium 1.20.2 and cert-manager 1.21.2 chart tarballs: SHA-256 checked against
  their publishers' Helm indexes on 2026-09-27.
- Envoy Gateway 1.9.1 chart: pulled from the locked OCI manifest documented in
  the lock; local tarball SHA-256 recorded. All installed and dynamically
  configured image references are overridden with resolved digests.
- Envoy Gateway 1.9.1 `install.yaml`: checked against GitHub release asset digest;
  used ONLY to extract `gateway.envoyproxy.io` CRDs. Its experimental Gateway
  CRDs and release-default Deployment are not installed.
- Gateway API 1.6.1 `standard-install.yaml`: checked against GitHub release asset
  digest; supplies Gateway CRDs plus upstream safe-upgrade policies.
- Kind v0.32.0 `const_storage.go`: tagged Apache-2.0 source with original license
  header. Renderer extracts its literal storage manifest, substitutes the two
  locked image references, and excludes Namespace and the default StorageClass.
  Foundation owns the namespace, and a separate Retain class is explicit.

The upstream projects and charts are Apache-2.0 licensed. Upstream provenance is
not a claim that image signatures or a release SBOM were verified. OCI image
manifest bytes were anonymously fetched, hashed, compared with registry digest
headers, and checked for linux/arm64 support. The node image and Argo/Redis chart
inputs retain their existing baseline locks.

The generated files retain Helm labels/hooks where upstream needs them. No
`helm install`/`upgrade` or Helm release state is introduced: Argo executes the
Envoy certificate-generation Job as its supported Helm hook equivalent.
