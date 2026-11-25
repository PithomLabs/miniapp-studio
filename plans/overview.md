I've incorporated all four new requirements into **Revision 3.0 of the product direction** and updated the coding-agent prompt to Version 2.

## **Updated deliverables**

Product specification · v3.0

**Miniapp Studio — Consolidated Product Direction**

Download Markdown

Contains the updated product vision, architectural requirements, deployment topology, Fly integration and acceptance gates.

Coding-agent prompt · v2

**Miniapp Studio — Implementation Plan Prompt**

Download Markdown

Instructs the coding agent to inspect the actual repository and produce an implementation plan only. No coding or deployment.

## **What I added**

| Requirement                | Binding decision                                                                 |
| -------------------------- | -------------------------------------------------------------------------------- |
| Studio built using Miniapp | First-party Miniapp; dogfood the framework                                       |
| Studio hosting             | Fly App/Machines, serverless-style operation                                     |
| Generated app URLs         | Unique, stable Fly-issued URL derived from user input                            |
| Fly integration            | APIs, `flyctl`, Sprite CLI, SDK and appropriate MCP integration                  |
| Infrastructure             | Prefer Fly-native services, including Managed Postgres, subject to compatibility |
| MVP quality                | Complete, polished, deployable product—not a disposable hackathon prototype      |

### **Two architectural details worth emphasizing**

**1. Serverless deployment**

Fly Machines aren't conventional function-as-a-service. Fly Proxy supports autostop/autostart, waking existing Machines on incoming requests and stopping idle ones. It doesn't provision or destroy the Machine pool. This distinction is explicitly incorporated into the prompt.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://flyio-landing.fly.dev\&sz=32)

Fly Docs+1



**2. Individual generated application URLs**

I recommend evaluating a dedicated Sprite per deployed Miniapp first. Sprites provide independent HTTPS URLs, and incoming requests can wake the Sprite. URLs are private by default, with public access requiring an explicit decision.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



The coding agent must compare that approach with shared Fly routing and separate Fly Apps rather than assuming one option is automatically superior.

I also added an evaluation of Fly Managed Postgres v2 instead of blindly retaining Neon from the older architecture. Fly now exposes Managed Postgres management through its Machines API.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.fly.io\&sz=32)

Fly Docs+1



## **Instructions for your coding agent**

Give it the two new documents, the original three plans, and access to the Miniapp repository.

The expected output is one reviewable implementation plan covering the actual framework gaps, self-hosting Studio, Miniapp generation, Fly deployment, unique URLs, connector security, MVP acceptance tests and the DBOS roadmap.

**My strongest architectural recommendation:** make Miniapp Studio the first application built with Miniapp itself, and make every generated application follow that same architecture. That dogfooding requirement will prevent us from accidentally building a separate application platform that merely happens to sit on top of Miniapp.
