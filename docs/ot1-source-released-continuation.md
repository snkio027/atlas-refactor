# SOURCE_RELEASED continuation candidate

The existing cluster is preserved: atlas-refactor-test-ot1,
UID b886f730-ea3b-44b9-a904-8bd55ac345f2. The old attempt remains 1/29, STOP
at SOURCE_RELEASED. Its post-STOP observation is not a retroactive checkpoint.
[ADR-0010](adr/0010-semantic-evidence-and-source-released-continuation.md)
is Proposed; [Failure Journal](s1-failures.md) records F8/F11.

## Scope

Only predecessor plan b195dc63e4ab2910bf14c2f351d7da6d7cfac91d9363ba08ecc1ff038f3fba70
and authority manifest b48895b8a871c0d685b66a7bfc5fe829f7f7408d8e954542d8c2fa06bff1eb74
are supported. This deliberately narrow implementation pins that concrete STOP,
its audited source DELETE and original executable. Another stopped attempt is
not an eligible input. No general recovery authority is granted.

Preparation validates every manifest file, baseline Gate-B, checkpoint, terminal,
post-STOP snapshot, successful request results and executable provenance. The
manifest binds the DELETE intent hash and metadata-only audit; it cannot
reconstruct or independently prove request-body fields. Orphan/UID/RV intent
comes from the archived bounded executable, not a fabricated audit payload.

A new plan changes only implementation identity, evidence model and continuation
binding. It preserves seven desired commits, scope, phases, tool/artifact locks
and the 300-second stage budget. Baseline identity stays referenced from the
immutable predecessor. The anchor compares resource, Application, project and
Node proof facts against the reviewed post-STOP observation; it rechecks Git,
cluster/kubeconfig identity, lock and audit continuity, and permits no additional
operator write since the recorded orphan DELETE.

## Gate semantics

- Evidence: two reads must have the same UID/content/tracking/Argo SSA fields,
  readiness and Application control facts. Named timestamps and RV bookkeeping
  can advance; both originals are saved. A changed fact or unavailable read
  fails once, without resampling it away.
- Transitional revision equivalence: only the preceding distinct planned Git
  revision is eligible, within the current publication epoch. The Application
  must preserve UID/spec and pass normal idle/Synced/Healthy/condition checks.
  Source is one locked local Kustomization with direct local resource files;
  all source files must have identical hashes. Remote/base/generator/plugin and
  multi-source inputs are unsupported. No component-name allowlist exists.
- Every foundation owner and platform-control require exact current revision.
  All Applications require it at Atlas gates. Normal Bootstrap/Seed adoption
  rules remain unchanged.
- Mutation: fresh exact UID/RV/full-spec/current-Git preconditions remain in
  existing request guards. Evidence equivalence never permits a stale-RV patch
  or an owner operation against a previous revision.

## Commands and execution boundary

Use the clean built binary and explicit private paths. Preparation and checking
perform no live mutation or Git publication:

    bin/atlas-ot1 plan-continuation --root "$IMPL" --repo "$DESIRED" \
      --predecessor-bundle "$AUTHORITY" --predecessor-manifest "$MANIFEST_SHA" \
      --output "$LATEST/plan.json"
    bin/atlas-ot1 check-continuation --root "$IMPL" --repo "$DESIRED" \
      --runtime-repo "$RUNTIME" --tool-dir "$TOOLS" --plan "$LATEST/plan.json" \
      --predecessor-bundle "$AUTHORITY" --predecessor-manifest "$MANIFEST_SHA" \
      --output "$LATEST/preflight"

After reviewing the new exact plan, replace check-continuation with continue,
provide --approve-plan <new-plan-SHA>, and choose a new create-only attempt path.
The continue command repeats the live anchor, records
CONTINUATION_ANCHOR_SOURCE_RELEASED, then atomically hands off the matching STOP
lock under an exclusive guard. It saves the old lock in the new attempt. A crash
leaves a lock/guard for review; no automatic stale-lock cleanup occurs.

It then executes original indices 2..28 (27 stages), beginning with
MIXED_SECRETS_STRICT_REFUSED, preserving mixed rollback, forward, reverse and
three remaining full Atlas gates. It never reruns baseline or source release.
On error, STOP retains the successor lock and authority evidence. On complete
success, only the matching successor lock is released. Normal run and baseline
verify-attempt reject continuation plans; full offline continuation replay is
not implemented, so predecessor and successor bundles must be reviewed together.

## Validation status

Local regressions cover semantic drift, inventory loss, stale/changed sources,
Atlas exact revision, lock conflicts, manifest tampering and simulated continuation.
The exact 53c82ce / 3e8db44c execution passed anchor/lock handoff and four remaining
stages, then stopped at original index 6 on F12. The original source-release STOP
and the new post-mutation STOP are both retained. The observability window remains
open; no automatic retry or closure occurred. See the Failure Journal for facts.

The F12 root fix separates operation proof from comparison. Successful syncs
require `reconciledAt >= finishedAt` on both checkpoint reads; known comparison
progress waits within the existing deadline. Strict refusal remains unchanged.
The previous SOURCE_RELEASED entry cannot execute at the F12 state. The one-off
[F12 experiment](../experiments/foundation-ownership/f12/README.md) is compiled
separately, pins the later STOP manifest and starts only at index 7 with strict
window closure. It adds no normal lifecycle command or evidence schema.
