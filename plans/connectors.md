https://fly.io/sprites/ecosystem/#connectors

Works with your agent. Yes, that one too.

Connect your coding agent to Fly.io — with plugins, Fly MCP, Sprites SDKs, API connectors, and native integrations. Build, test, and run services on a real Linux box instead of your laptop.

Point any agent at
https://mcp.fly.io

Manage Sprites and other Fly.io resources. Sign in with OAuth and review the requested access.

Fly MCP can request permissions including deleting apps, reading secrets metadata, and running commands on Fly Machines. Approve only the access you need. See the guide's Scope of access.
Give your Sprite access to external APIs

Connect through the Sprites API gateway while keeping provider credentials outside your Sprite. Choose a service and grant access in the dashboard.

Available providers can vary by organization during rollout. Your dashboard shows which connectors you can enable.
Use Sprites with Meta (Muse Spark)
fly.io/dashboard

Call Muse Spark from a Sprite while keeping your Meta API key outside the environment.

    1

    Open your organization's Sprites Connectors

    In the Fly.io dashboard, select your organization, then open Sprites → Connectors → Add connector.
    2

    Configure Meta

    Use your Meta developer team's API key. Availability varies by organization during rollout. The gateway keeps the credential off your Sprite.
    3

    Grant your Sprite access

    Open access settings and turn off Allow all sprites before restricting access by name prefix or labels. If you set both, a Sprite must match both. Save the policy. A connector with no access policy denies all Sprites.
    4

    Test from your Sprite

    Run the API Usage example from the connector's detail page inside an authorized Sprite. The gateway attaches the stored credential; do not copy it into the Sprite.

Supports streaming and tool calls through OpenAI-compatible chat completions and responses, and Anthropic-compatible messages. The gateway also allows model discovery; other Meta API paths are blocked.
Open the dashboard
Read connector docs
What your agent can do with it
Run the risky stuff somewhere else

Builds, migrations, dependency upgrades, and whole test suites run on the Sprite. Your laptop stays clean and stays yours.
Checkpoint before it breaks things

A checkpoint snapshots the whole filesystem in about a second. If the agent wrecks the environment, restore and carry on.
Keep services running between sessions

Dev servers, workers, and databases live on as Sprite services, with a URL you can open. Closing your editor doesn't kill them.
Confirm before anything destructive

Confirmation behavior varies by plugin. Check its safety controls before destructive work; terminal backends such as Hermes do not provide Sprites-specific confirmation hooks.



Connectors
Meta (Muse Spark)
OpenAI
TypeSafe AI
Anthropic
OpenRouter
Ollama Cloud
GitHub
Slack (user)
Slack (bot)
Discourse
S3-compatible Object Store
Fly.io Machines
Custom API
Steel Browser Tools
Firecrawl
Deepgram
Fireworks
Parasail
Depot build insights
Upstash Redis
Tigris
Agentmail
Resend
Neon
ElevenLabs
SenseLab
Hugging Face
Blotato
Keenable
LemonSlice
Mnemom
Era
Composio Tools
Supabase
Usable
Link Agent Payments
Box
Codex
Apify
Browserbase
Pinecone
Temporal Cloud
Kagi
Gigapipe
DigitalOcean Inference
Echo by Fulcrum
Linkup
Vocal Bridge
Google Gemini
Nebius Serverless AI Jobs
Assembly AI
Granola
Linear
Notion
Nscale
Pushover
Sentry
Stripe
Tavily
ZeroGPU

