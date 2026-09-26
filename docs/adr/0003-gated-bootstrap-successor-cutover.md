# ADR-0003: Gate Bootstrap successor cutover independently of language selection

- Status: Proposed
- Date: 2026-09-27
- Deciders: repository owner and required CODEOWNERS
- Supersedes: none
- Runtime authorization: none
- Scope: candidate identity, compatibility decisions and authority transfer Gates

## Context

The independent Go experiment passed its fourth disposable integration Gate.
It cannot operate on existing Atlas targets: names, identity/configuration,
resource layouts, status behavior and recovery capabilities differ. The current
original Shell implementation also has a documented gap between its live self
heuristic and Accepted ADR-0002's protected adoption proof. Neither copying that
gap nor calling a stronger experimental health gate equivalent is acceptable.

## Decision proposed

Treat `atlas-refactor` as a successor candidate, not a permanent parallel engine.
Adopt the [Go Bootstrap Cutover Contract](../go-bootstrap-cutover-contract.md)
as the proposed release and rehearsal requirements. Select at most one normal
Bootstrap engine with Tier-0 mutation capability in any target authority scope;
use a zero-writer interval during transfer. After adoption, neither normal
engine regains Seed/Tier-0 mutation authority. Independent recovery retains its
own authorization and Operation Fence and cannot run concurrently with normal
target mutations.

Separate language acceptance, parity approval, rehearsal, target runtime
authorization and old-engine retirement. No Gate implies the later Gates.
The candidate must either conform to the existing Accepted Atlas adoption and
recovery contracts or first obtain an authoritative amendment. This proposal
does not amend the original ADR-0002 through ADR-0005 and does not approve a
specific identity migration, principal/RBAC implementation or configuration map.

Require a dedicated reviewed migration workflow for legacy state, distinct
from normal apply and independent recovery. Preserve Root and GitOps resource
identities by default. Configuration/CLI breaks need an explicit versioned
contract and caller migration; they cannot be hidden by a language shim before
the original superseding ADR is accepted. All runtime work remains prohibited
until the exact target and allowed operations are separately approved.

## Rollback and retirement

Rollback before a state transition may restore the old engine selection only
after fencing the new engine and validating the unchanged identity. Once an
incompatible Identity or Receipt exists, use a verified compatible fallback or
independent recovery; never delete proof or restore v1 to make the old engine
run. Exercise both a legitimate pre-transition rollback and a post-transition
compatible fallback, plus old-engine rejection. A refusal alone is not proof
that rollback works.

Retire the old normal engine's entry points, credentials and distribution after
target acceptance and a defined rollback window. Preserve audit archives and
separately governed Recovery/Drill components. Archive custody never grants
runtime authority. There will be no permanent user-selectable dual-engine
mutation mode.

## Acceptance and current disposition

This record may define migration policy after owner/CODEOWNER review, but does
not itself authorize any cutover. Changes in the authoritative Atlas repository
need its own superseding ADR and required reviews. Target execution additionally
requires the mode-specific prerequisites in the contract and independently
bound runtime Gates. The first rehearsal requires G0–G4 and a rehearsal-only
G6, then produces G5; a later target cutover requires G0–G6. Rehearsal approval
never authorizes that later target.

Current outcome: technical feasibility proven for the disposable Go profile;
static parity audit **BLOCKED**; cutover rehearsal **NOT RUN**; candidate,
fallback, target and runtime authorization **UNSET**. Status remains Proposed.
