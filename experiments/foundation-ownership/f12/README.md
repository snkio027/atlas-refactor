# One-off F12 recovery

This executable is excluded from normal builds. It completes only the F12 STOP
on cluster UID `b886f730-ea3b-44b9-a904-8bd55ac345f2`, whose immutable manifest is
`57a0e0f74aaa11220a0f37625da0f2b0035060b5faf6e66d6259355534a7a806`.
It cannot select another predecessor or stage, and is not a lifecycle interface.

The plan retains the original SOURCE_RELEASED binding as historical lineage.
The dedicated binary digest additionally binds the fixed F12 incident constants;
normal `continue` still rejects the F12 lock and state. No evidence schema changes.
The original seven desired Git revisions and 29-stage schedule are reused.

Before handoff, the executable verifies both immutable bundles, the current STOP
lock, Git, cluster/kubeconfig identity, audit continuity, all 13 UID/content/SSA/
tracking facts (3 secrets, 4 observability, 6 old source), persistent App/Bootstrap/
AppProject identities, four nodes, and a coherent fresh successful comparison.
Observability must retain UID `00afbd19-65db-4fba-8018-468dfd59d181` and the exact
operation result that finished at `2026-09-27T14:56:57Z`. Only its window is open.

The first mutation is the guarded observability window-to-strict patch. The
original index 7 sync and indices 8..28 then run with existing UID/RV/full-spec/
Git guards, 300-second stage deadlines and Gate-B checks. No adoption replay,
tracking patch, rebuild, rollback, retry, or automatic stale-lock recovery exists.
STOP preserves the complete authority attempt; success removes only its own lock.

Build from a clean checkout with the locked Go toolchain:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
  go build -trimpath -tags ot1_f12_recovery -o .state/latest/ot1-f12/atlas-ot1-f12 \
  ./experiments/foundation-ownership/f12
```

The binary accepts `-mode plan`, `-mode check` (read-only), and `-mode execute`.
All paths to the two original bundles, desired source, runtime and tools are fixed
relative to `-root`. A new private `-output` is required for each mode. Check and
execute need `-plan`; execution additionally requires the exact `-approve-plan`.
Plan compilation and check do not authorize live execution. Record the reviewed
exact binary/plan and user decision before running; retain final authority evidence.

F12 operation/comparison semantics follow the upstream wait predicate:
[Argo CD v3.5.1 app wait](https://github.com/argoproj/argo-cd/blob/v3.5.1/cmd/argocd/commands/app.go).
The same predicate remains in
[v3.5.3](https://github.com/argoproj/argo-cd/blob/v3.5.3/cmd/argocd/commands/app.go),
so a dependency upgrade does not replace the verifier fix.
