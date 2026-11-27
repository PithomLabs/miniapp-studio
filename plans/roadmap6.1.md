# **Miniapp Studio roadmap — current position**

**We are past the architectural foundation and into B2: building the remote, Sprite-resident AI software engineer.**

The important distinction is that much of the engineering foundation is complete, but the central Hacksgiving demo experience has not yet been demonstrated end to end.

We are on the right architectural track. Our immediate priority should be turning the completed foundations into a working, observable product.

## **1. Roadmap status**

**B0 — Studio manages one development task**

Accepted

Studio's core task management, execution records, authorization and evidence foundations are established.

**B1 — Studio manages its local development loop**

Accepted

Local execution, checkpoints, recovery, Git evidence, Console Run and related verification are complete.

**B2 — Remote development and resident engineering agent**

In progress

This is our current milestone and the Hacksgiving centerpiece.

- Sprite-per-app topology and core integration interfaces established.
- B2-01A lifecycle contract implemented and accepted.
- B2-01B local lifecycle implementation prepared.
- Fly's actual lifecycle semantics investigated.
- PostgreSQL test runner and host execution broker implemented and adversarially verified.
- Broker installation, owner-side sandbox attestation and live infrastructure proof remain outstanding.
- Genuine in-Sprite agent execution, GitHub Connector operations, authorized writes and Workspace Preview still need end-to-end demonstration.

**B3 — Formal Application Preview**

Not started

Immutable OCI artifact deployed to a Fly Machine with an isolated Neon branch.

**B4 — Studio troubleshoots itself**

Not started

Real runtime logs, diagnostics, evidence-driven repair and verification.

**B5 — Release management**

Not started

Staging, Production, release approvals, migrations, health verification and rollback.

## **2. What the new architecture documents clarify**

The most important contribution is that we now have three distinct decisions for every miniapp tool.

| Dimension          | Question                         | Decision                       |
| ------------------ | -------------------------------- | ------------------------------ |
| Execution category | How must it execute and recover? | A, B, C or D                   |
| Compute placement  | Where should the code run?       | Sprite, Machine or Hybrid      |
| Authorization      | What is it allowed to do?        | Dry-run, confirm or autonomous |

These decisions must remain independent. The category guidelines explicitly establish this separation.&#x20;

app_category_guidelines.md



Our placement rules are now:

- **Sprites:** persistent development workspaces, agent-authored code, generated code and isolated previews.
- **Machines:** trusted, reviewed application workloads of every execution category.
- **Hybrid:** a trusted DBOS workflow on a Machine delegates a bounded untrusted task to a Sprite.

And critically, Sprite checkpoints never replace DBOS business-workflow checkpoints.&#x20;

app_category_guidelines.md

machines_vs_sprites.md



This validates our existing separation between Workspace Preview and formal Application Preview.

## **3. How far are we from the demo?**

I would measure progress by completed demo experiences, not source-code volume.

Here is the current acceptance picture:

| Hacksgiving demo requirement                     | Current position                                                |
| ------------------------------------------------ | --------------------------------------------------------------- |
| Local Studio cockpit                             | Foundation implemented                                          |
| Actual coding agent inside persistent Fly Sprite | Not yet demonstrated                                            |
| Studio-to-Sprite task handoff                    | Integration groundwork exists; live proof pending               |
| Genuine Fly GitHub Connector call from Sprite    | Not yet demonstrated                                            |
| Owner-authorized GitHub Issue/Discussion writes  | Designed; implementation and proof pending                      |
| Sprite checkpoints and restoration               | Local checkpoint foundation exists; remote Sprite proof pending |
| Evidence returned to Studio                      | Local foundation exists; remote end-to-end proof pending        |
| Sprite Workspace Preview URL                     | Not yet demonstrated                                            |

These are evidence-based status classifications, not percentage estimates.

My assessment: **we have mature foundations, but the visible product experience that differentiates Miniapp Studio still needs to be built and demonstrated.**

The biggest missing piece is not another architecture document. It is the first successful end-to-end remote engineering task.

## **4. What we should prioritize now**

I recommend the following sequence.

Priority 1

**Activate the host execution broker**

Complete the one-time owner-side installation attestation, install and verify the broker, and allow the agent to operate the approved Docker runner without repeatedly asking you to execute commands.

This is enabling infrastructure, not a demo feature.

Priority 2

**Finish B2-01B live lifecycle proof**

Verify actual Sprite provisioning, Tasks API holds, automatic warm/cold pause, wake behavior, receipt persistence, reconciliation and separately authorized destruction.

Stop relying on inferred SDK behavior. Observe the real provider.

Priority 3

**Make the resident coding agent real**

Run an actual coding agent inside the persistent Builder Sprite. Prove that it can investigate a task, use its repository and tools, survive interruption and resume.

Priority 4

**Prove genuine GitHub Connector integration**

Have the agent inside the Sprite read repository evidence through the actual Fly Connector. Keep mutations behind Studio-side authorization enforcement. Demonstrate the refusal of unauthorized writes.

Priority 5

**Demonstrate recovery and Workspace Preview**

Create a checkpoint, deliberately break the workspace, restore it under explicit approval, and show the running application through its Sprite URL.

Stretch

**B3 Application Preview**

Only after the main demo works: build a reviewed OCI image, deploy to a Fly Machine and isolated Neon branch. Valuable proof, but not necessary to make the Sprite-centered demo compelling.

## **5. What we should deliberately postpone**

Do not let the new category guidelines become another large implementation project.

Adopt the classification rules as we build actual miniapps:

| Category                     | Adoption priority                          |
| ---------------------------- | ------------------------------------------ |
| A — ordinary handlers        | First                                      |
| B — River background jobs    | When needed                                |
| C — bounded batch operations | When a real batch requires it              |
| D — DBOS durable workflows   | When real business requirements justify it |

The shared effect ledger, approval mechanism, capability proxy, idempotency and reconciliation should become framework-owned capabilities. But we should not build speculative versions of everything before delivering a useful miniapp. The guideline itself says to choose the lowest sufficient category.&#x20;

app_category_guidelines.md

app_category_guidelines.md



## **6. The demo story we should aim to deliver**

[Fly.io (@flydotio) on X](https://images.openai.com/static-rsc-4/b08dfpqs0uKmXYPaF_DXQpujE8wz8Mm7cU2gexWIbMV1W__yJu_JuIjgRoT-l0VnylVeloNzgF01BFkTmn3Inh4mSPeTh2L_ynpEOfhd1qAyjeL39T1kaZOZ-O3gbZSlTM2Z7a92ItnhSN0zIJtbc9GP7KTLUChgIFFz4NsHxJk?purpose=inline)

[Claude Code Agent Monitor - real-time dashboard for Claude Code](https://images.openai.com/static-rsc-4/mESWr7-AfQgWLCpblQoHXaCegtBlc7KPkqfgopimv3KjFGqcz-gKgIdagOSKEOPP1Aczhz4WqKfIG69SKydmO6uizrqXycHbt8kZ0DCj7pEqv0FuIaTDMHsO_N9F5dQleoOzfLQL7dkjn-Tz_NlPfe4MqqKSD7LomD7V0p8zFd4?purpose=inline)

[CodeScaleBench: Testing coding agents on large codebases and Multi-repo software engineering tasks | Dan Adler](https://images.openai.com/static-rsc-4/VuzWmBhQXhAkqJuPWyjBQ-CXptQpuzk30a3_qMqTn38rbSI91M2rxGt7ggKIVchoJ7xXy9Ilt9XVemgqHoz5rityFebtE80ERkD96jB0sNg2VDPKFZ8vliUyFsQPOFuh2Q-_00jnpw044o7Jt-GrTVim7uY5M29xpO3CsXxQDV8?purpose=inline)

11

The final demonstration should tell one coherent story:

1. Open Miniapp Studio and create an engineering task.
2. Wake the persistent Builder Sprite.
3. Dispatch the coding agent into its own workspace.
4. Have it inspect a real GitHub problem through the Fly Connector.
5. Let it prepare a proposed bug Issue and request your approval.
6. Authorize the relevant write and separately authorize code repair.
7. Have the agent reproduce the defect, repair it, run tests and create a PR.
8. Demonstrate checkpoint recovery.
9. Open the Workspace Preview URL.
10. Inspect the resulting evidence and execution history in Miniapp Studio.

This demonstrates something much more meaningful than a chatbot that generates code: **a personal AI software engineer that can take responsibility for a development task inside its own persistent, recoverable computer, while Studio controls its authority and records its work.**

## **My overall assessment**

We have not lost our direction. The new material strengthens it.

The roadmap remains B0–B5. B0 and B1 are accepted. B2 is the active milestone. B3–B5 remain future capabilities, with B3 a valuable stretch for the demo.

**The next major milestone should be one complete, real task executed by the resident agent inside the Builder Sprite and managed by Miniapp Studio.**

That is the point at which our architecture stops being primarily a collection of well-engineered components and becomes a demonstrated product.

Our operating principle remains:

**Plan enough. Build the smallest complete experience. Test it against reality. Learn. Iterate.**
