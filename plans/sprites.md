# Get Sprites-ready

These are instructions for you, an AI coding agent, to start working with
Fly.io Sprites. Read them once, then act on them. The human who handed you
this document wants you to do the setup, not to explain it back to them.

A Sprite is a full Linux computer with a persistent filesystem. Files you
write and packages you install stay put between sessions. When nobody is
using it, it sleeps; a command or an HTTP request wakes it in well under a
second. Use one anywhere you would otherwise run builds, test suites,
migrations, or dev servers on the human's laptop.

## 1. Install the CLI

Check first: if `sprite --version` prints a version, the CLI is already
installed, so skip to step 2. Otherwise install it. Run the installer as its
own command, not chained with other commands, so the human can see what it
does before approving it.

macOS and Linux:

```sh
curl https://sprites.dev/install.sh | bash
```

Windows (PowerShell):

```powershell
iwr https://sprites.dev/install.ps1 -useb | iex
```

The installer puts `sprite` in `~/.local/bin`. Check it with `sprite --help`.

## 2. Authenticate

```sh
sprite login
```

This opens a browser, or prints a URL if it cannot. Show that URL to the human
and wait for them to approve; the CLI is authorized once they do. Do not try
to work around the approval step.

Signing up costs nothing: start on the Free plan with $1 of starter credit,
no card required, and the Free plan gives $5 of usage every month after you add a card, which can pay for a $4 Developer Managed Postgres database. Starter credit is
available for up to 7 days. Add a card to keep going when it is used up or expires. Tell the human this if cost is what makes them
hesitate.

If the human would rather you use the hosted MCP server instead of the CLI,
add `https://mcp.fly.io` to your MCP configuration (for Claude Code:
`claude mcp add --transport http sprites https://mcp.fly.io`). Plugins
for specific agents are listed at https://fly.io/sprites/ecosystem.

## 3. Create a Sprite and make it the default

```sh
sprite create --skip-console my-sprite
sprite use my-sprite      # so you can drop -s my-sprite from every command
sprite info               # its URL and settings
sprite list
```

Name Sprites after the project you are working on. Only destroy Sprites you
created yourself, and only when the human asks: `sprite destroy -s my-sprite`.

## 4. Run commands in it

```sh
sprite exec -- ls -la
sprite exec -- npm install express
sprite exec -- bash -c "cd /home/sprite && git clone https://github.com/example/repo"
```

`sprite exec` blocks until the command finishes and returns stdout and stderr.
The Sprite comes with Node.js, Python, Go, and Git installed. Your working
directory is `/home/sprite`.

For something long-running and interactive, start a TTY session; it is
detachable with `Ctrl+\` and you can come back to it:

```sh
sprite exec --tty -- npm run dev
sprite sessions list
sprite sessions attach <id>
```

Anything started this way stops when the Sprite sleeps. For processes that
should survive that, use a service (below).

## 5. Serve HTTP

Every Sprite has a URL of the form `https://<name>-<org-id>.sprites.app`;
`sprite info` prints the exact one. Traffic routes to port 8080 by default, and
a request wakes the Sprite. The URL requires a Sprites token unless you make
it public:

```sh
sprite config update --url-auth public   # anyone with the URL
sprite config update --url-auth sprite   # back to token-only (the default)
```

To reach a private port from the human's machine instead, forward it:

```sh
sprite proxy 5432          # Sprite's 5432 at localhost:5432
sprite proxy 3001:3000     # local 3001 to the Sprite's 3000
```

A web server that should be up whenever the Sprite wakes belongs in a
service. Services are managed from inside the Sprite with `sprite-env`:

```sh
sprite exec -- sprite-env services create web --cmd node --args server.js --http-port 3000
sprite exec -- sprite-env services list
sprite exec -- sprite-env services restart web
sprite exec -- tail -n 50 /.sprite/logs/services/web.log
```

`--cmd` is the binary only; arguments go in `--args`, comma-separated. Only
one service can own the HTTP port.

## 6. Checkpoint before anything risky

A checkpoint snapshots the whole filesystem in about a second. Take one before
a migration, a big refactor, a dependency upgrade, or anything you are not
sure about, and restore if it goes wrong:

```sh
sprite checkpoint create --comment "before migrating to Postgres 17"
sprite checkpoint list
sprite restore v4
```

From inside the Sprite the same commands are `sprite-env checkpoints create`,
`list`, and `restore`.

## 7. Talking to other services

Do not paste API keys into the Sprite. Connectors hold the credential in the
human's organization and route calls through
`https://api.sprites.dev/v1/gateway/<provider>/<connection_id>/<path>`, so the
Sprite never sees the token. The human sets these up in the dashboard under
Connectors; ask them to, rather than asking for the secret. Details:
https://docs.fly.io/sprites/concepts/connectors.md

## 8. Doing this from code instead of the CLI

REST API, authenticated with an organization-scoped Sprites token. Create a
Sprite by posting its name as JSON to the collection endpoint:

```sh
curl --fail-with-body -X POST https://api.sprites.dev/v1/sprites \
  -H "Authorization: Bearer $SPRITES_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-sprite"}'
```

Successful creation returns `201 Created` with the Sprite's JSON metadata.

For non-interactive execution, pass the executable and each argument as
repeated `cmd` query parameters, in order:

```sh
curl --fail-with-body -X POST \
  'https://api.sprites.dev/v1/sprites/my-sprite/exec?cmd=echo&cmd=hello' \
  -H "Authorization: Bearer $SPRITES_TOKEN" \
  --output exec-output.bin
od -An -tx1 exec-output.bin
# 01 68 65 6c 6c 6f 0a 03 00
```

The response is `application/octet-stream`: `01` marks stdout (`hello\n`
above), `02` marks stderr, and the final `03 00` means exit code 0. HTTP 200
alone does not mean the command succeeded: check the exit frame.

SDKs: `@fly/sprites` (JavaScript), `github.com/superfly/sprites-go` (Go),
`sprites-py` (Python), `superfly/sprites-ex` (Elixir). Use an SDK for interactive
execution; full HTTP and WebSocket protocol details are in the API reference:
https://docs.fly.io/sprites/api

## Read next

- Sprites docs, overview: https://docs.fly.io/sprites/index.md
- Everything in the Fly.io docs, Sprites included, as one index:
  https://docs.fly.io/llms.txt
- Working with Sprites (sessions, ports, persistence):
  https://docs.fly.io/sprites/working-with-sprites.md
- CLI reference: https://docs.fly.io/sprites/cli/commands.md