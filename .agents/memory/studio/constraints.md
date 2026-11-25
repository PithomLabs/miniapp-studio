---
type: project
description: Framework may be improved during Studio build when warranted
---

# Studio build constraint (2026-11-24)

While building Miniapp Studio on top of Miniapp, improving Miniapp itself is imperative where absolutely warranted. This relaxes the earlier frozen-framework posture: framework changes are allowed when Studio needs prove them necessary, still with evidence discipline (pin + tests) and without duplicating or conflicting with the Prompt32 foundational program (F-0/F-1/F-2 track, see studio docs/miniapp/).

# Working principle (2026-11-24, user instruction)

Always address the underlying problem, never a band-aid fix. Solve root causes with durable corrections; do not paper over symptoms with local workarounds.

# Miniapp access clarification (2026-11-24, user instruction)

Read /home/chaschel/Documents/go/miniapp whenever needed to fix the Miniapp codebase where warranted and authorized. Warrant follows the §14 ownership test (joint-evolution protocol); each fix still requires explicit owner authorization — read access does not itself authorize modification.

# Test environment (2026-11-24, user instruction)

Postgres on Docker is installed and is the designated path for PG-backed testing (P0 gate, import spike, integration tests). Note: from the agent sandbox shell the docker socket currently returns permission-denied, so PG steps either need daemon access enabled for the agent or must run from the owner's shell.

# Security rule (2026-11-24, user instruction, absolute)

Never expose any secret whatsoever: no credential values in commands, prompts, logs, files committed to repos, reports, or chat output. This includes throwaway/test credentials. Generate secrets directly into 600-permission files, pass them via env-files/secret mounts, reference them by file path — never emit or display a value.
