Yes, both are true: Fly Sprites do not require Docker to deploy or run, and installing Docker inside a Sprite is generally not advisable unless you have a hard dependency on running a specific container image.

1. Sprites do not require Docker (by design)

Unlike standard Fly Machines—which require you to build and push an OCI/Docker image—Sprites intentionally drop Docker/OCI packaging altogether. [1] (https://sandbox.watch/p/fly-io-sprites), [2] (https://fly.io/blog/design-and-implementation/)
• Every Sprite boots directly from Fly's managed Ubuntu base image.
• You install dependencies directly onto the persistent Linux filesystem (using apt, go, npm, etc.) or restore from a saved checkpoint.
• Background processes are managed natively using Fly's built-in service manager (sprite-env services create ...) rather than systemd or Docker Compose. [1] (https://fly.io/blog/design-and-implementation/), [2] (https://boat.dev/compare/sprites-vs-docker-cloud-sandboxes), [3] (https://sandbox.watch/p/fly-io-sprites), [4] (https://northflank.com/blog/e2b-vs-modal-vs-fly-io-sprites), [5] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995)

2. Why installing Docker inside a Sprite is not advisable

Because a Sprite already runs user code inside an inner container namespace inside a Firecracker microVM, running Docker inside it is essentially "Docker-in-Docker" and introduces significant friction and limitations: [1] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995), [2] (https://fly.io/blog/design-and-implementation/)
• Breaks out of the box: A standard apt install docker.io and docker run will fail due to cgroupv2 restrictions (the Sprite's /sys/fs/cgroup is a namespaced threaded cgroup that blocks child domain controllers) and overlay filesystem stacking limits. [1] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995)
• Requires heavy workarounds: To make Docker work, as documented on the Fly.io Community Forum, you have to switch the runtime to crun with cgroup management disabled, turn off containerd-snapshotter to force overlay2 onto /var/lib/docker, disable bridge networking/iptables (--network host only, meaning no -p port mapping), and manually wire dockerd into sprite-env services. [1] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995)
• Loss of resource isolation: With cgroup management disabled, Docker flags like --memory and --pids-limit are silently ignored. [1] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995)
• Checkpoint mismatch: Sprite checkpoints only snapshot the writable root overlay. Because /var/lib/docker sits on a separate mounted volume, rolling back a Sprite checkpoint will not roll back your Docker images or container states. [1] (https://rywalker.com/research/sprites), [2] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995)

How this fits your Miniapp Architecture

In your app_category_guidelines framework, avoiding Docker inside Sprites aligns cleanly with your rules:
• In a Sprite (dev/workspace or untrusted agent execution): Install Go, run tests, or execute generated scripts directly on the Sprite's Ubuntu environment and manage processes via sprite-env services. [1] (https://boat.dev/compare/sprites-vs-docker-cloud-sandboxes), [2] (https://community.fly.io/t/how-to-run-docker-inside-a-sprite/28995)
• Promoting to Fly Machines (production): Do not attempt to build or export a Docker container inside the Sprite. Instead, promote the reviewed source code through your CI pipeline to build an immutable OCI/Docker image for Fly Machines.
Are you trying to run local dependencies (like Postgres/Redis) inside a Sprite during development, or looking for a way to test your production Dockerfile before deploying to Fly Machines? Let me know and I can suggest the cleanest workaround.



## question

read above, does it mean we can discard Docker in Sprites and opt for installing Postgres on Sprites instead? the problem is local development testing of everything database-related and also with the host broker thing, the coding agent is stuck now


## answer

Yes. I recommend we make native PostgreSQL inside the Builder Sprite our default database for development and integration testing, and stop trying to make Docker-in-Sprite work.

This is a particularly good fit for Miniapp Studio. It also gives us a way to unblock the coding agent without completing the Host Execution Broker installation.

The attachment points in the right direction, and Fly's own documentation provides stronger confirmation: its engineering blog explicitly describes installing PostgreSQL on a Sprite with apt-get. Sprites have persistent Linux filesystems, and their native service manager handles long-running processes. 

fly.io+2

1. The important distinction: Docker is not going away
We should distinguish two environments:
Builder Sprite
Native dependencies









x.com


Install PostgreSQL, Go, development libraries and other tools directly into the Sprite.

The coding agent runs tests directly against PostgreSQL.


Fly Machine deployment pipeline
Keep Docker / OCI









medium.com


Build immutable OCI images using CI or a dedicated build environment. Deploy reviewed images to Fly Machines.

Fly deliberately designed Sprites without user-managed OCI images. Their intended model is to install dependencies directly and use persistent filesystem state. Fly Machines retain the container-based deployment model. 

The Fly Blog+1

So our decision should be:
Component	Decision
Docker inside Sprites	Reject as default
Native PostgreSQL inside Sprites	Adopt
Native Go toolchain inside Sprites	Keep
sprite-env services	Use for PostgreSQL lifecycle
Docker / OCI for Fly Machines	Keep
CI image builds	Keep

Docker-in-Sprite is technically possible through complicated workarounds, but the documented configuration has limitations involving networking, resource enforcement and checkpoint state. That is unnecessary complexity for our database testing requirement. 

Fly.io

2. How PostgreSQL should work inside our Builder Sprite
I recommend one dedicated PostgreSQL installation per Builder Sprite, associated with its miniapp.
Builder Sprite
Persistent Linux filesystem

Go toolchain · Git repository · dependencies · test scripts


Native PostgreSQL service
Loopback endpoint

127.0.0.1:5434

Dedicated development/test cluster, not production data


Coding agent

go test · migrations · RLS tests · transaction tests · concurrency tests · DBOS integration tests


Miniapp Studio

Receives execution status, test results, logs and evidence.


Several details matter:
- Install the same PostgreSQL major version used by our existing test baseline. Our current host fixture uses PostgreSQL 19, so we must preserve version parity rather than blindly accept whatever version the default Ubuntu repository installs.
- Configure PostgreSQL to listen only on 127.0.0.1, port 5434. This preserves our existing test endpoint restriction.
- Use a dedicated test role and disposable databases with generated credentials.
- Register PostgreSQL through sprite-env services, not systemd or Docker Compose.
- Make the database test harness create/reset isolated test databases so successive runs do not contaminate one another.
Fly documents that installed packages, services and on-disk databases are included in Sprite checkpoints. Restoring a checkpoint rewinds the filesystem. This makes the environment naturally recoverable, although it should not replace proper database backups or clean test-database lifecycle management. 

Linux computers for agents+1

3. What happens to the Host Execution Broker?
This is where I think we can make a significant simplification.

Recall why we built it: the restricted coding-agent sandbox could not access the host Docker socket or host PostgreSQL. We created a privileged host-side broker to give the agent a narrowly authorized way to run database tests.

But if the coding agent executes inside its Builder Sprite, it has its own Linux environment and can run PostgreSQL locally.

That changes the requirement.
Existing approach	Recommended approach
Agent sandbox → HostExec broker → Docker → PostgreSQL	Agent inside Sprite → native PostgreSQL
Privileged host daemon	No host daemon needed for DB tests
Docker runner	Native PostgreSQL test harness
Host state and path attestation	Sprite filesystem and service management
Agent blocked by host permissions	Agent operates within its own environment

My recommendation: stop making HostExec installation a prerequisite for B2 or database testing.

Do not delete the hostexec/ or testrunner/ implementations. Preserve them as optional infrastructure for future local-host automation or workloads that genuinely require Docker on the host.

The broker is a separate privileged trust boundary. We should not pay that complexity cost just to provide PostgreSQL connectivity to an agent that already has its own persistent Linux computer.

One important qualification: PostgreSQL on a Sprite only unblocks testing when the task and test process actually run inside that Sprite. It will not magically provide connectivity to a separate local sandbox. We should move the coding agent's development execution to the Builder Sprite, which is already the B2 architectural direction.
4. What does this unblock?
Almost our entire database-dependent development loop:
Testing capability	Native PostgreSQL
Schema migrations	Yes
SQL queries and constraints	Yes
Transactions and rollback	Yes
Concurrent transactions and locking	Yes
Row-Level Security	Yes
Tenant isolation	Yes
Database integration tests	Yes
DBOS/PostgreSQL integration	Yes, subject to DBOS dependencies
Repository-wide Go tests	Yes
Testcontainers-specific behavior	Not directly; retain a separate Docker-capable test environment

This is not a mock database. It is real PostgreSQL running in the agent's actual development environment.

5. The implementation decision I recommend
I would make this the next bounded engineering task:
Recommended next task
B2 — Sprite-native PostgreSQL development environment
1. Verify the actual Builder Sprite and its current environment.
2. Install the matching PostgreSQL major version natively.
3. Initialize a dedicated test cluster.
4. Configure loopback-only access on port 5434.
5. Register it using sprite-env services.
6. Create a repeatable database bootstrap/reset script.
7. Adapt the existing Taskfile test targets to use this PostgreSQL instance.
8. Run the complete database-dependent test suite, including migrations, RLS, tenant isolation and transaction tests.
9. Return machine-readable test results and logs to Studio.
Acceptance means the agent can start from a clean Sprite, bootstrap its test database and independently run the complete database test suite without Docker, HostExec or manual owner intervention.

One additional rule: do not expose PostgreSQL through the public Sprite URL, inject production credentials, or use real customer data. It is a local development/test database.
My final architectural recommendation
This is more than a workaround for the broker problem. It is a better match for what Fly Sprites were designed to provide: a persistent, recoverable computer where the coding agent owns its development environment.

I would lock in these decisions:
- Native PostgreSQL in Builder Sprites is the standard development database.
- Docker-in-Sprite is not part of our standard architecture.
- HostExec is no longer a prerequisite for B2 database testing.
- Docker/OCI remains part of the trusted Fly Machine build and deployment pipeline.
- The coding agent must be able to bootstrap, test and recover its own development environment.
The underlying problem was that we were trying to give an isolated agent access to somebody else's computer to obtain a database. The Sprite gives that agent its own computer. We should use it.


## question

one more clarification, just reply briefly, how does my local dev environment work now for database testing if we are not going to use locally? remember local-first, the goal is to draft guidelines later per findings above for both local dev environment and remote one


## answer

We are **not abandoning local database testing**. That would contradict our local-first architecture.

We should support two equivalent development environments:

|                 | Local development            | Remote development              |
| --------------- | ---------------------------- | ------------------------------- |
| Execution       | Your workstation             | Builder Sprite                  |
| PostgreSQL      | Local PostgreSQL             | Native PostgreSQL inside Sprite |
| Docker          | Optional; usable on the host | Not required                    |
| Database access | Local loopback               | Sprite loopback                 |
| Tests           | Same test suite              | Same test suite                 |
| Results         | Local execution output       | Evidence returned to Studio     |

Your existing local setup remains valid: PostgreSQL 19 in the host Docker container, exposed at `127.0.0.1:5434`. We don't need to discard it or install another database just because the remote environment changes.&#x20;

**The principle for our future guidelines:**

1. Local-first: the complete development and database-testing workflow must work on the local workstation without Fly.
2. Remote parity: the same workflow must work inside a Builder Sprite, using its native PostgreSQL.
3. Shared contracts: same PostgreSQL major version, migrations, test commands, isolation rules and acceptance criteria.
4. Environment-specific infrastructure: Docker may be used locally; native services are preferred in Sprites.

So we are establishing **two supported development environments, not replacing one with the other**. We can draft the guidelines after validating both against real test runs.


