# Ordinary application delivery — local validation

Scope: proposed ADR-0018, single-owner local development. This records development
checks, not Runtime PASS or production approval.

## Executed

- Deterministic static linux/arm64 OCI construction and verification, with rejected
  extra-image/tag aliases, malformed references and damaged closures.
- Ordinary single WebService compilation with and without a Binding; no required
  unbound peer or /roundtrip endpoint; repeated output identical.
- Application purpose rejects the platform acceptance Probe before any effect.
- Provider finalization failure leaves no application success marker.
- Partial/STOP/contradictory attempt blocks replacement by a fresh plan.
- Recorded read-only views cannot approve a mutation; unpublished reviews retain
  the previous deployed view, but partial new attempts cannot hide behind it.
- Workspace symlink/output protection, strict config, escaped history rejection,
  public D1 extraction layout, nested layout, ambiguity rejection.
- Exact target confirmation, irrelevant flags rejected before instance loading.
- Existing Probe replay/terminal tests, image import identity tests and original
  S2 compiler/publication contracts remain part of task quality.

A real experiment-archive binary was also packaged and used by the CLI:

| Binding | Value |
| --- | --- |
| Application source | ed1a13fe3b61431fe195a632e889d4f5025ce7d9 |
| linux/arm64 ELF SHA256 | 529b6b59a2d54a32bfe432f4df84c989b866690f72c273af69c6707275432193 |
| OCI archive SHA256 | 10a1afbbcd4e651f2d584d0222f65f59f0cfe4054495e394526075c3851134d6 |
| Image | atlas.local/experiment-archive:v1@sha256:7461927fe48447076e9c3610075517463281c1767a0126331c662e63c2180dd8 |

Local init/check generated one Project, one WebService and one S3 Binding.
An impossible resource request exited 1 and named the authored file and resources
field. Restoring the input restored successful validation.

A read-only plan against the existing r7 deployment correctly exited 1:
the branch already owns Project demo. No image import, credential use, Git
publication or Kubernetes mutation occurred. This explicitly verifies the
single-Project boundary rather than treating r7 as an arbitrary hosting target.

The final candidate passed the complete task quality after all workspace/status
changes: race tests, 348 GitOps resources, real Helm/Kustomize, eight Python
contract tests and the frozen OT-1 entry points. The source tree passed
go vet and formatting checks. No runtime fixture was enabled.

## Still requires a separately authorized target

Ordinary first deployment, two real business updates, HTTPS business uploads and
downloads, business error diagnosis and return usage have not been executed.
The app mode's DEPLOYED result deliberately leaves functional=UNPROVEN.
The existing S2 r7 Runtime PASS remains bound to d011813; it is not reassigned
to this implementation. No new cluster, Trust Root or ciphertext was created.
