# Fixed stage-12 continuation

This separate build-tagged executable completes only the stage-12 STOP defined
in [ADR-0012](../../../docs/adr/0012-fixed-stage12-continuation.md). It has no
selectable stage, cluster or predecessor and adds no normal lifecycle command.
F13 authority/desired identity/runtime semantics stay unchanged.

It verifies the original baseline and exact immutable STOP bundle, then repeats
read-only stage-12 Ownership/Gate-B and lock/Git/audit/identity fences. The anchor
is saved without creating a stage-12 checkpoint. Only then can the existing lock
handoff run, followed by original indices 13..28. Stage 12's publication is never
replayed. Any mismatch or unexpected runtime outcome stops without automatic retry.

Build from the clean implementation with locked Go:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
  go build -trimpath -tags ot1_stage12_continuation \
  -o .state/latest/ot1-stage12/atlas-ot1-stage12 ./experiments/foundation-ownership/stage12
```

`-mode plan -output <new plan>` prepares the original plan with the new executable
binding. `-mode check -plan <plan> -output <new directory>` performs the full
read-only anchor. `-mode execute` additionally needs `-approve-plan <exact SHA>`
and a new attempt output directory. All other paths are fixed relative to `-root`.

The owner's current request covers completing these remaining original stages.
Record the resulting exact binary/plan and this request before execution. Keep
old STOPs immutable; retain actual mutation/final Gate evidence. After full forward
and reverse Gate-B pass, freeze S1 experiment code and consolidate review; no S2,
CI infrastructure, generic resume, recovery or ownership framework belongs here.
