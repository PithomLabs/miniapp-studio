# Development parity report

**Status:** Stage 1 skeleton — structure ratified; all result cells PENDING Stage 2 live proofs.
No cell in this report may be filled from design, fakes, or recalled runs. Each VERIFIED-equivalent cell cites the
proving run (date, commit, branch IDs, counts).

## Contract under test

Neon-first development contract per ADR-004/005/006: pinned project + known branch + TLS + least-privilege role;
canonical `studio dev` verbs via Taskfiles; fingerprint carries R312 fields
(project/branch/parent IDs, configured + actual version, extensions, compute config).

## Parity matrix (one row per environment per proof round)

| Proof round / env | Dev branch | CI branch | Notes / deviations |
|---|---|---|---|
| Commit SHA | PENDING | PENDING | Same commit across envs |
| Neon project ID | PENDING | PENDING | Non-prod project (ADR-005) |
| Branch ID + parent | PENDING | PENDING | Dev-persistent vs per-run ephemeral |
| `SHOW server_version` / `_num` | PENDING | PENDING | Must read 19.x both envs |
| Extension inventory proof | PENDING | PENDING | Re-proven "none" on PG19 (N2) |
| Migrations applied (both chains) | PENDING | PENDING | Full chain from zero |
| DB suite counts (pass/fail/skip) | PENDING | PENDING | Zero unexplained skips |
| RLS + tenant isolation | PENDING | PENDING | Green both envs |
| Full suite counts | PENDING | PENDING | Stabilized repeats |
| Branch cleanup verified | PENDING | PENDING | CI branch deleted; dev branch retained |
| Fingerprint record | PENDING | PENDING | Machine-readable artifact per env |

## Deviation policy

Any cell that cannot match across environments is a DEVIATION entry with root cause and owner decision — never a
silent edit. Preview-environment rows (per-PR branch + app) are added when T3 lands; until then this report covers
dev + CI only.
