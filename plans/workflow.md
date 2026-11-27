## question

brief reply only, how does the transition work from local dev env to remote dev env to staging env to prod env, where do the workspace preview and application preview figure in the process, elaborate

## answer

The lifecycle is a progression from **development workspaces to verified, immutable application releases**. Local and remote development are not production deployment environments.

## **The complete transition**

**1. Local Development**

Developer works locally, using local PostgreSQL. Runs migrations, unit tests and integration tests.

Same repository-controlled development and testing contract.

**2. Remote Development — Builder Sprite**

Code and task handoff to the persistent Sprite. The resident coding agent implements, tests, checkpoints and reports evidence.

Workspace Preview lives here

Mutable application running inside the Sprite, accessible through its Sprite URL.

**3. Application Preview — Fly Machine + Neon branch**

Reviewed source is built into an immutable OCI artifact, deployed to a temporary Fly Machine and connected to an isolated Neon database branch.

Formal deployment verification

Tests the actual application deployment rather than the mutable development workspace.

**4. Staging — Fly Machine + Staging PostgreSQL**

Deploy the approved artifact into production-like infrastructure, using a separate persistent staging database. Verify migrations, integrations, health and operational behavior.

**5. Production — Fly Machine + Production PostgreSQL**

Explicit production approval. Deploy the same verified artifact, execute authorized migrations, verify health and retain rollback capability.

Live application

## **Where the two previews fit**

|          | Workspace Preview                | Application Preview        |
| -------- | -------------------------------- | -------------------------- |
| Runs on  | Fly Sprite                       | Fly Machine                |
| Code     | Mutable working copy             | Immutable reviewed build   |
| Database | Development/test DB              | Isolated Neon branch       |
| Purpose  | Fast feedback during development | Verify release candidate   |
| Evidence | Development feedback             | Release-readiness evidence |

**Important:** Workspace Preview is not a mandatory release stage. It is a convenience for inspecting work in progress. Application Preview is the formal pre-release verification environment.

## **The governing rules**

1. **Git preserves source history.** The Sprite workspace and its checkpoints are not release artifacts.
2. **One active writer per app.** Local and remote agents must not independently modify the same workspace.
3. **Promote reviewed source, never a Sprite checkpoint.** Build a new immutable artifact from the reviewed commit.
4. **Promote the same artifact digest** through Application Preview, Staging and Production, rather than rebuilding separately in each environment.
5. **Databases remain isolated.** Preview, Staging and Production never share authoritative database state.
6. **Each consequential transition requires its own authorization.** Approval to develop or preview does not authorize staging or production deployment.

The central idea: **Local and Remote Development create the software; Workspace Preview lets us inspect it; Application Preview proves a release candidate; Staging validates operational readiness; Production serves customers.**