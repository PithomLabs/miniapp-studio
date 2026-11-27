APPROVED — STAGE 1 EXIT

**I approve closing the Neon-first reconciliation stage. I select Option A: Demo-first, minimal Neon-compliant slice.**

I find no foundational architectural blocker in the reviewed deliverables. The reconciliation has addressed the major conflicts, preserved the mandatory Neon-only constraint, and established a credible sequence from development infrastructure to demonstration and eventually to the complete platform.

The next priority should be demonstrating a working product, not implementing every component of the eventual engineering platform.

## **Selected sequencing: Option A**

1. **T1 — Local Neon development**

   Resolve C4, establish the Neon developer-branch workflow, revise `studio dev`, and make `task dev:setup` reproducible.
2. **Minimal B2 demonstration**

   Demonstrate the existing agent execution path, genuine GitHub Connector read, Studio-enforced approval boundary, and visible execution evidence.
3. **Platform roadmap**

   T2 — Fly-native CI; T3 — per-PR previews; T4 — background coding agents; T5 — customer-miniapp database provisioning.

This is the best sequencing because it produces a meaningful demonstration without requiring the entire platform to be operational first. Both the demo and future platform remain Neon-compliant.&#x20;

stage1-exit-proposal.md

stage1-conflicts.md



## **Three corrections before Stage 2 execution**

These do not invalidate the Stage 1 exit decision, but should be reflected in the implementation plan.

**1. Make infrastructure prerequisites track-specific.**

The exit proposal currently requires a GitHub App, controller credential ceremony and N14 quota verification before any Stage 2 work. That is unnecessarily restrictive for Option A.

T1 needs its Neon non-production project, approved secret store and relevant ADRs. GitHub App installation and controller credentials belong to T2/T4; preview concurrency and quota validation belong to T3. Preserve the individual track gates.&#x20;

stage1-exit-proposal.md

stage2-tracks.md



**2. Keep T5 consistent with ADR-016.**

The track currently prescribes project-per-miniapp provisioning, while the requirement matrix leaves project-per-miniapp versus project-per-tenant versus another security boundary to ADR-016.

T5 should implement whichever allocation strategy the accepted ADR establishes, rather than deciding it prematurely.&#x20;

stage1-req-matrix.md

stage2-tracks.md



**3. Preserve evidence accuracy.**

The exit proposal should say 33 capability claims were *reviewed and classified*, not that all 33 were verified. The verification table correctly distinguishes verified, attested, assumed, disputed and unverified claims.

I also independently reread the full Fly per-PR preview article, including its previously outstanding teardown and cost sections. It reinforces the importance of independent databases, cleanup reconciliation and read-only agent diagnostics. Its GitHub Actions and Fly Managed Postgres examples must not be copied into our Neon-first, Fly-native implementation.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.fly.io\&sz=32)

Learn+1



The disputed `sprite exec` timeout claim should remain excluded. Sessions, markers and heartbeats remain the chosen supervision approach.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.fly.io\&sz=32)

fly.io



## **Approval boundaries**

My decision approves:

- Closing Stage 1.
- Selecting Option A.
- Using T1 as the first implementation track.
- Preparing the minimal B2 demonstration after T1 succeeds.
- Retaining T2–T5 as the subsequent platform roadmap.

It does not authorize infrastructure provisioning, GitHub writes, deployment or other consequential live operations.

One review limitation: ADR-004–018 themselves were not included among the uploaded files. I approve the architecture and reconciliation represented by the supplied evidence, but cannot claim to have independently reviewed every ADR's full text. Record their formal acceptance separately.

**Final verdict: APPROVED. Close Stage 1 and proceed with Option A.**

The objective now is to get a real, demonstrable Neon-backed Miniapp Studio working, without sacrificing the architecture required to grow it into the complete platform.