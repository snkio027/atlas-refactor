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

## r2 runtime result and publication recheck fix

The upgraded `atlas-app-r2` run used implementation
`9271110a975260dde2eb3d94f7dcf906c3773e1a` and approved review plan
`631e6e6f78b7bd960ba2b15f493b713581c27251ad37c6a61ed5b9969dbc134a`.
D1 installation and its independent read-only verification passed. Ordinary v1
published permissions, project and infrastructure and retained all three Ready
Gates. Before consumer publication, a closing-read change in Application
`foundation` correctly produced `Pending`, but the predecessor recheck called a
single-shot observer and returned it as a terminal failure. The attempt is STOP;
consumer had no intent/receipt and no business upload ran. Later convergence does
not change that history. Full private evidence remains in the r2 result bundle.

The fix implements the already accepted ADR-0017 read-wait contract at this
missing boundary: predecessor rechecks use `ObserveFor` with the existing
15-minute limit, bounded by the caller deadline, and one rollout session for UID
and comparison continuity. Only explicit `Pending` reads repeat. Identity/spec/
ownership/Git drift, unavailable reads, cancellation and timeout still stop;
intent creation, Git push and all other effects remain outside the wait. Existing
Ready evidence stays immutable; the latest read records a terminal failure.
No compiler output, publication order, retryable mutation, recovery command or
STOP continuation is introduced.

`TestPublicationRecheckPendingPreservesGateAndUIDHistory` uses compiled source
trees and receipt/Gate fixtures to check Pending → Ready, retained UID history,
API failures, immutable prior Gates and the dependent effect boundary.
`TestPublicationRecheckDeadlineAndCancellationStopBeforeEffects` checks the fixed
budget and cancellation, including a late successful read. These exercise the
shared production read poll with synthetic reads/effects, not a real push.
Existing receipt/unknown-outcome tests continue to reject publication replay.

## Still requires a separately authorized target

Ordinary first deployment, two real business updates, HTTPS business uploads and
downloads, business error diagnosis and return usage have not completed.
The app mode's DEPLOYED result deliberately leaves functional=UNPROVEN.
The existing S2 r7 Runtime PASS remains bound to d011813; it is not reassigned
to this implementation. The r2 D1 instance created its own Trust Root and three
public D1 ciphertexts; its two application ciphertext versions remained private
and unpublished at STOP. A corrected execution needs its own exact plan.
