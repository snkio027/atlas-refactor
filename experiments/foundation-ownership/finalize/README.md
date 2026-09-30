# Fixed S1 finalization

Build only with `-tags ot1_finalization`; this entry is absent from ordinary
binaries. It accepts no stage, cluster, predecessor or Git-selection override.
[ADR-0011](../../../docs/adr/0011-desired-identity-and-durable-handoff.md) binds the
stage-23 STOP, frozen manifest/lock, three owner UIDs and original plan. Modes are
`plan`, `check`, and `execute`, with create-only private output and exact plan SHA.

`check` proves current forward Ownership/Gate-B without changing the STOP lock.
`execute` repeats that anchor, then hands off the lock and runs only original
24..28. Nine guarded Application writes, two existing Git publications, 300 seconds
per phase, 40 minutes total; any unexpected state stops without retry/rollback.
The historical stage-23 STOP remains STOP. There is no new stage-23 checkpoint.

A passed final reverse Gate-B ends S1 feature work. Preserve final authority
evidence and keep this code as an incident-specific ceremony reference.
