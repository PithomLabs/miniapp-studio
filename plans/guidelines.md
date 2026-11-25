## prompt

just to be clear I expect the AI-generated miniapps to follow the Miniapp framework (not Python or any other language), is that clear in the specs and design as we develop this project? brief concise reply only

## answer

Yes, that is the intended architecture, but **the constraint needs to be made more explicit and enforceable in the specifications.**

The current Gateway plan already requires AI-generated applications to produce a **MiniappSpec**, which must pass the existing Miniapp validator and admission boundary before acceptance (Phase D, §14). It also identifies natural-language-to-MiniappSpec as the intended production workload.&#x20;

agent-gateway-implementation-plan.md

agent-gateway-implementation-plan.md



I am making this a binding architectural invariant:

**Every AI-generated miniapp must be built for and executed by the Miniapp framework, using its MiniappSpec and supported capabilities. Python, Node.js, or another language must never become an alternative application runtime or substitute for Miniapp.**

This must be enforced in:

- Coding-agent instructions and build assignments.
- MiniappSpec validation and artifact admission.
- Phase D acceptance tests.
- Deployment and Runtime Sprite provisioning.

Go, shell, or other tools may be used for infrastructure and development tooling where appropriate, but not to bypass the Miniapp application model.

**Miniapp is the product framework, not merely one optional output format.**


I’ll treat this as a **locked architectural invariant** for Miniapp Studio:

AI-generated applications must use the Miniapp framework and its MiniappSpec. Python, Node.js, or other languages must never replace or bypass the Miniapp application runtime. This applies to agent instructions, artifact validation, acceptance tests, and deployment.

I can enforce this in our ongoing project work, though I cannot directly force a persistent Memory update from this chat.


## prompt

just so we are aligned does writeup4.md reflect our vision of this project and if so how do we ensure that it will be properly enforced in the implementation

## answer

**Yes, broadly. But there are a few important qualifications.** The attached document reflects the central business vision: business owners describe processes in ordinary language, receive working software, clarify requirements, review evidence, retain control over consequential actions, and continuously improve their applications.&#x20;

writeup4(2).md



However, I would strengthen three aspects before treating it as the definitive product vision.

### **1. Miniapp must be the only application framework**

The document mentions generating a `MiniappSpec` and validating it through the Miniapp admission boundary. That is correct, but it does not explicitly prohibit alternative application runtimes.

Our locked invariant must be:

> Every AI-generated miniapp must be designed, represented, validated, built, tested, and executed using the Miniapp framework. MiniappSpec is the canonical application specification. Python, Node.js, Go, or another language must never become an alternative application runtime or bypass the framework.

Go, Python, shell, and other languages may be used for infrastructure and development tooling, but not as substitutes for the Miniapp application model.

### **2. Preserve tenant ownership and isolation**

One recommendation in the attachment needs correction: the suggestion that one customer's custom connector automatically becomes available to other customers.

That must not happen by default. We can eventually provide a shared catalog of vetted connector definitions and reusable templates, but customer credentials, private configurations, business data, and tenant-created integrations must remain isolated.&#x20;

writeup4(2).md



The customer should also retain ownership of their application, not merely the conversation that produced it.

### **3. Distinguish promised benefits from capabilities not yet implemented**

The attachment describes benefits such as mobile approvals, execution receipts, rollback, and accumulated reusable patterns. These are valuable aspirations, but some require additional implementation beyond the current accepted roadmap. For example, rollback of an application does not automatically undo an external action such as sending an email.&#x20;

writeup4(2).md



## **How we enforce the vision in implementation**

I recommend five enforcement layers:

| Layer                     | Required enforcement                                                                                      |
| ------------------------- | --------------------------------------------------------------------------------------------------------- |
| Coding-agent instructions | Explicitly require MiniappSpec and Miniapp framework compliance                                           |
| Artifact admission        | Reject applications that do not satisfy the Miniapp specification and supported capabilities              |
| Build and testing         | Execute generated applications through the Miniapp runner/framework, not arbitrary language runtimes      |
| Runtime provisioning      | Install and execute only approved Miniapp-compatible artifacts                                            |
| Regression tests          | Include negative tests proving Python/Node.js standalone applications and framework bypasses are rejected |

Most importantly, **agent instructions are not sufficient enforcement**. The validator, admission boundary, runner, and deployment pipeline must independently enforce the rule.

The existing Phase D requirement to re-admit generated MiniappSpecs through the existing validator is the natural enforcement point.&#x20;

agent-gateway-implementation-plan.md



**My conclusion:** The attached document reflects our intended customer value. The missing piece is making Miniapp framework exclusivity an explicit product and technical acceptance requirement, rather than leaving it implicit.

Phase B should remain focused on the Gateway. Phase D must enforce this invariant before any AI-generated miniapp can be accepted, and Phase E must make bypassing it impossible through the normal deployment path.



## prompt

rectify your reply above with this requirement, 1. Miniapp must be the only application framework and must use Go language only in the miniapp implementation


## answer

**Yes. With this correction, the architectural requirement is stricter than my previous formulation.**

The attached document reflects our business vision, but its technical requirements must explicitly enforce the following invariant.

## **1. Miniapp is the exclusive application framework, and Go is its exclusive implementation language**

> Every AI-generated application must be implemented exclusively using the Miniapp framework, written in Go, and represented through the canonical MiniappSpec. No other application framework, programming language, or runtime may be used to implement or execute a Miniapp.

This means:

- **Framework:** Miniapp only.
- **Implementation language:** Go only.
- **Application specification:** MiniappSpec.
- **Validation:** Existing Miniapp validator and admission boundary.
- **Execution:** Miniapp framework/runtime.
- **Deployment:** Only approved, Go-based Miniapp-compatible artifacts.

Python, JavaScript, TypeScript, Node.js, Django, React, or other technologies must not become alternative ways to implement a Miniapp.

This restriction applies to the generated application itself. It does not prohibit using other languages or tools for infrastructure, the Agent Gateway, coding-agent software, or development tooling unless separately required.

## **2. How we enforce this**

| Layer              | Mandatory enforcement                                                                         |
| ------------------ | --------------------------------------------------------------------------------------------- |
| Agent instructions | Require Go and Miniapp exclusively for application implementation                             |
| MiniappSpec        | Canonical representation of application behavior                                              |
| Artifact admission | Reject non-Miniapp or non-Go application implementations                                      |
| Build and testing  | Compile and validate using the Go/Miniapp toolchain                                           |
| Runtime            | Execute through Miniapp, never an alternative application runtime                             |
| Deployment         | Accept only verified, approved Miniapp artifacts                                              |
| Regression tests   | Prove that Python/Node.js applications and alternative-framework implementations are rejected |

Most importantly, instructions to the coding agent are not sufficient. Enforcement must happen programmatically at artifact validation, admission, build, and deployment boundaries.

## **3. What this means for the customer vision**

The customer is not asking an AI to generate arbitrary software in whichever framework it prefers.

The customer is asking Miniapp Studio to transform business requirements into **Go-based applications built on the Miniapp framework**.

The AI agent is an implementation assistant, not a framework selector.

The customer owns the resulting application, its business behavior, its data, and its governance history.

The attached document's benefits—plain-language creation, clarification, verifiable review, controlled approvals, operation, and continuous evolution—remain valid.&#x20;

writeup4(2).md



**Final architectural invariant: Miniapp is the only application framework. Go is the only implementation language for Miniapp applications. MiniappSpec is the canonical application specification. No generated application may bypass these requirements.**

Phase B can continue building the Gateway without changing its scope. This invariant must become an explicit mandatory acceptance criterion in Phase D and Phase E.

**Locked as a binding architectural invariant for Miniapp Studio.**

1. **One application framework:** Miniapp exclusively.
2. **One implementation language:** Go exclusively for Miniapp applications.
3. **One canonical specification:** MiniappSpec.
4. **One execution model:** Miniapp framework/runtime.
5. **No bypasses:** Alternative frameworks, languages, or application runtimes cannot be used to implement Miniapps.

This applies to coding-agent instructions, generated artifacts, validation, admission, testing, and deployment.

Infrastructure tools and the Agent Gateway may use other languages where appropriate, but generated Miniapp applications may not.

**I will treat violations as architectural defects, not optional design choices, in all subsequent implementation plans, coding prompts, and acceptance reviews.**

