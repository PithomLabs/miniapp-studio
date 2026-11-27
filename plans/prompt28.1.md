Agreed. This is now a mandatory part of the current execution mission, not a future improvement. Add the following amendment to Prompt28. It strengthens Phases 1 and 4 and introduces clean CI as the third independent verification environment.

# PROMPT28 AMENDMENT — Mandatory Repeatable Development Environment Contract

**Status: REQUIRED — incorporated into the current execution mission**

This amendment is part of the original mission. It is not optional, deferred work, or a separate future initiative.

The objective is to eliminate "works on my laptop" as an acceptable engineering outcome.

Miniapp Studio must establish a repeatable, reproducible development and database-testing process that works across three independent environments:

1. Local developer workstation.
2. Remote Fly Builder Sprite.
3. Clean CI environment.

All three must verify the same repository, migrations, database contracts and applicable test suite.

## 1. Non-negotiable architectural principle

**One development contract. Three independently verified environments.**

Local and remote development must not become separate implementations with subtly different behavior.

The following must remain consistent:

- Repository and source code.
- Go toolchain version and relevant dependency versions.
- PostgreSQL major version.
- Required PostgreSQL extensions and database configuration.
- Migration chain and migration ordering.
- Database schema and constraints.
- RLS and tenant-isolation behavior.
- Test fixtures and test semantics.
- Test commands and acceptance criteria.

Environment-specific differences are restricted to infrastructure adapters, service lifecycle and credentials.

For example:

| Property              | Local workstation                             | Builder Sprite                | CI                          |
| --------------------- | --------------------------------------------- | ----------------------------- | --------------------------- |
| PostgreSQL            | Existing local PostgreSQL 19 Docker container | Native PostgreSQL 19          | Clean PostgreSQL 19 service |
| Database endpoint     | `127.0.0.1:5434`                              | `127.0.0.1:5434`              | CI-local service endpoint   |
| Database credentials  | Local test credentials                        | Sprite-local test credentials | Ephemeral CI credentials    |
| Repository migrations | Same                                          | Same                          | Same                        |
| Test suite            | Same                                          | Same                          | Same                        |
| Acceptance criteria   | Same                                          | Same                          | Same                        |

CI may require an environment-specific connection endpoint. That does not justify changing application or test semantics.

Do not modify the existing endpoint allowlist merely to make an environment-specific workaround pass. Any necessary exception requires investigation, demonstrated justification and separate Project Owner approval.

## 2. Create a version-controlled development interface

Inspect the repository's existing conventions and choose the smallest maintainable implementation.

The repository must provide a version-controlled, documented command interface for:

- Environment diagnosis.
- Dependency and toolchain verification.
- Database readiness checks.
- Development database initialization.
- Test database initialization and reset.
- Migration execution and verification.
- Database-dependent test execution.
- Full relevant test execution.
- Safe cleanup.

Use one canonical interface across local and remote development.

The interface may delegate to environment-specific infrastructure adapters:

- Local adapter uses the existing host PostgreSQL installation/container.
- Sprite adapter manages native PostgreSQL using `sprite-env services`.
- CI adapter uses an isolated CI PostgreSQL service.

The interface must not require Docker inside a Sprite.

Do not introduce an unnecessary framework, generalized orchestration engine or elaborate abstraction. A small version-controlled script or existing repository-native task mechanism is sufficient.

Do not assume Taskfiles already exist. Inspect the repository and avoid creating an unnecessary second command system.

The exact implementation is a technical decision based on the existing codebase, but the repeatable command contract is mandatory.

## 3. Environment pinning and verification

Every environment must verify, not merely assume, its prerequisites.

At minimum, record and verify:

- Go version.
- Dependency/module integrity and availability.
- PostgreSQL server version.
- PostgreSQL extensions and versions.
- Database encoding and locale.
- Timezone.
- Relevant server configuration.
- Migration version and migration status.
- Required environment variables.
- Test database connectivity and isolation.

Use repository-controlled version declarations and configuration wherever practical.

Do not silently accept an incompatible database version, missing extension, different schema state or toolchain.

The environment verification command must fail with a useful explanation when a required invariant is violated.

## 4. Clean-bootstrap requirement

A repeatable environment must be reproducible from a clean starting state.

Demonstrate that:

1. A fresh repository checkout can be prepared using documented instructions.
2. Required development dependencies can be installed or verified.
3. The development/test database can be initialized.
4. All required migrations can be applied in the correct order.
5. The database-dependent tests can run without undocumented manual intervention.
6. The environment can be reset safely.
7. A subsequent run produces the same expected schema and test behavior.

Do not use an existing developer database state as proof of clean bootstrap.

Test destructive reset operations only against explicitly identified disposable development/test databases. Never apply test cleanup to production or unrelated databases.

## 5. Phase 1 — Local reference verification (strengthened)

The existing Phase 1 acceptance criteria remain in force, with these additional requirements:

- Run the repository's canonical environment diagnosis command.
- Verify local toolchain and database configuration against repository declarations.
- Verify the actual PostgreSQL 19 server and required extensions.
- Demonstrate clean test database initialization.
- Demonstrate migration from a clean starting state.
- Run the representative and complete applicable database-dependent test suites through the canonical command interface.
- Verify safe cleanup and repeatability with a second run.

Record the exact Git commit and environment fingerprint.

A developer's already-running database is not enough. Prove the documented bootstrap path works.

**Phase 1 gate:** Local reference environment is reproducible and the applicable suite passes with exact results before remote database implementation proceeds.

## 6. Phase 4 — Remote parity verification (strengthened)

Inside the Builder Sprite:

- Use the same repository-controlled development interface.
- Use the same migration and test code.
- Use the same PostgreSQL major version and relevant extensions.
- Bootstrap the test database from a clean state.
- Run the same applicable database-dependent test commands.
- Verify RLS, tenant isolation, transaction behavior and cleanup.
- Repeat the run to check that results are stable.
- Record the Sprite environment fingerprint and Git commit.

Compare local and Sprite results.

Investigate every difference. A different result must be explained by evidence and resolved or explicitly classified as a genuine infrastructure limitation.

A test skipped because the environment cannot execute it is not a passing test.

Testcontainers-specific tests remain explicitly identified as requiring Docker-capable infrastructure. Native PostgreSQL inside a Sprite must never be presented as verification of Docker or Testcontainers behavior.

**Phase 4 gate:** Local and remote environments produce equivalent results for the same applicable test suite from clean database initialization.

## 7. Clean CI — Mandatory third proof

CI must independently validate the same development contract.

Inspect the existing GitHub Actions workflows and modify the smallest appropriate workflow rather than building a parallel CI system.

Requirements:

- Provision an isolated PostgreSQL 19 service or equivalent clean CI database environment.
- Verify required extensions and configuration.
- Obtain dependencies using the repository's pinned declarations.
- Execute the repository-controlled canonical database test workflow.
- Apply migrations from a clean database.
- Run relevant RLS, tenant-isolation and database-dependent integration tests.
- Preserve exact test results and useful failure logs.
- Associate the result with the tested Git commit.

CI must not reuse the local developer database or Sprite database.

CI must not report green when required integration tests were skipped or silently excluded.

If a particular test genuinely requires Docker/Testcontainers, classify it separately and run it through an appropriate Docker-capable CI job. Do not conflate that job with the native-PostgreSQL parity result.

**CI acceptance:** A clean CI environment can independently reproduce the applicable database test result from the same Git commit.

## 8. Cross-environment parity report

Produce one concise, version-controlled report containing:

| Evidence                       | Local    | Builder Sprite | Clean CI |
| ------------------------------ | -------- | -------------- | -------- |
| Git commit                     | Required | Required       | Required |
| Go version                     | Required | Required       | Required |
| PostgreSQL version             | Required | Required       | Required |
| Required extensions            | Required | Required       | Required |
| Clean database bootstrap       | Required | Required       | Required |
| Migration verification         | Required | Required       | Required |
| Database test count            | Required | Required       | Required |
| Pass/fail/skip counts          | Required | Required       | Required |
| RLS and tenant-isolation proof | Required | Required       | Required |
| Cleanup verification           | Required | Required       | Required |

Record deviations explicitly. Do not hide differences behind a single green badge.

Every VERIFIED claim must have reproducible evidence.

## 9. Reporting and failure policy

For each environment, report:

- Exact commands executed.
- Git commit.
- Environment fingerprint.
- PostgreSQL and Go versions.
- Migration state.
- Test counts.
- Pass, fail and skip counts.
- Skip reasons.
- Relevant logs and failure evidence.
- Cleanup and isolation results.

Distinguish:

- VERIFIED — executed and passed.
- BLOCKED — execution prevented by a concrete limitation.
- FAILED — executed and exposed a defect.
- DEFERRED — deliberately excluded with an explicit reason.

Do not claim environment parity simply because all three environments compile the application.

Do not fix an environment-specific failure by weakening the shared test suite, adding unexplained skips or changing a security invariant.

Investigate the root cause and make a durable correction.

## 10. Deadline and scope discipline

This requirement is mandatory, but it must not become an excuse to delay the core B2 demo indefinitely.

Implement the smallest complete and maintainable solution.

Prioritize:

1. Local reproducibility.
2. Builder Sprite reproducibility.
3. Clean CI verification.
4. Immediate return to the B2 resident-agent and GitHub Connector demo path.

Do not introduce a new database orchestration product, generalized environment-management framework or unrelated infrastructure.

Do not install or activate HostExec merely to satisfy this requirement.

Preserve local-first development. The local environment must remain capable of developing and testing independently of Fly.io.

## 11. Updated definition of success

The development-environment work is complete only when:

- A clean local checkout can bootstrap and test.
- A clean Builder Sprite workspace can bootstrap and test.
- A clean CI run can bootstrap and test.
- All three use the same repository-controlled development contract.
- Database versions, schema, migrations and relevant test semantics agree.
- Differences are explicit, explained and evidenced.
- The result is reproducible by another developer or coding agent without relying on undocumented knowledge.

**The standard is no longer "it works on my laptop." The standard is "the repository can prove that it works in every supported development environment."**

**Decision:** This amendment is mandatory for the current mission. Phases 1 and 4 now include clean-bootstrap and repeatability gates, and CI becomes the third independent proof. The B2 demo remains the deadline-critical deliverable.