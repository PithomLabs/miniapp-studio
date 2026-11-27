The solution is to **fix the coding agent's execution environment, not PostgreSQL or Docker.**

Your evidence establishes that the host and the agent operate in different security and network boundaries. The agent cannot reach the host's loopback or Docker socket. Changing credentials, reinstalling PostgreSQL, changing file permissions, or rewriting `TEST_DATABASE_URL` will not solve that.

## **My recommendation: give the agent controlled access to local PostgreSQL**

Preserve the existing Docker PostgreSQL installation and sandbox isolation. Introduce a narrowly scoped network path.

**Recommended execution architecture**

**Coding agent sandbox**

Agent shell and Go test processes

Approved network path — specific destination and port

**Controlled PostgreSQL bridge**

Host gateway or restricted TCP proxy

**Existing Docker PostgreSQL 19**

`127.0.0.1:5434` on host

The bridge should expose only PostgreSQL to the authorized agent environment. It should not expose the Docker daemon.

### **Three possible solutions, in order**

1. **Sandbox network configuration (preferred):** provide a route to a host-side PostgreSQL endpoint, allowlisted to the agent and database port. Preserve network isolation for everything else.
2. **Dedicated integration-test container:** run a test runner attached to the existing Docker Compose network, with access to PostgreSQL but no Docker socket. Mount the working repository and let it execute the DB-backed test commands.
3. **Host-side test execution:** temporarily have the owner execute database tests from the host terminal. Useful as an immediate workaround, but not an ideal permanent developer experience.

### **What we must not do**

- Do not grant the agent unrestricted Docker socket access. Docker daemon access is effectively highly privileged host access.
- Do not use host networking as a blanket workaround without assessing its security implications.
- Do not expose PostgreSQL on all host interfaces.
- Do not use production credentials or data for local integration tests.
- Do not count skipped database tests as passed tests.

## **Directive to the coding agent**

# Engineering Directive: Restore Local Database Test Connectivity

**Objective:** Establish a durable, secure and repeatable execution path through which the coding agent can run PostgreSQL-backed integration tests against the existing local PostgreSQL 19 Docker installation.

**Governing principles:** First-principles investigation, local-first engineering, reality-first iteration, least privilege and root-cause remediation.

1. Preserve the existing host PostgreSQL installation. Do not reinstall, recreate or replace it.
2. Do not request unrestricted Docker socket access or weaken sandbox isolation as a shortcut.
3. Investigate the coding-agent runtime's supported network and execution configuration. Identify the precise host-to-sandbox connectivity mechanism.
4. Design the smallest secure solution that makes the existing PostgreSQL service reachable from the agent's execution environment.
5. Restrict connectivity to the required database endpoint and port. Use dedicated least-privilege test credentials and an isolated test database.
6. Do not modify application database architecture to accommodate a sandbox limitation.
7. Implement the durable environment solution, documenting configuration, startup, shutdown, health checks and recovery.
8. Verify actual connectivity from the agent shell using `pg_isready` and a real SQL query.
9. Run migration 000025, the receipt-store tests, RLS/grant tests, privilege-parity tests and the complete serial test suite.
10. Report exact commands, results, skipped tests and any remaining limitations. A zero exit code with skipped DB tests is not full verification.

**Acceptance criterion:** The coding agent can repeatedly execute database-backed tests from its normal authorized environment against the existing local PostgreSQL service, without requiring the owner to manually execute every test and without granting the agent unrestricted host or Docker access.

Stop and request an owner decision before making any security-boundary change whose scope exceeds this directive.

**My architectural decision:** establish this connectivity path once and make it part of the standard Miniapp Studio development environment. Every future coding agent should inherit the same tested local-first execution capability.

We should not have to solve this problem again for every implementation milestone.