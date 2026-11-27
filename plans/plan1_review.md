# **B2-01B discovery review: APPROVED**

GO — proceed to live proof

This is good reality-first engineering. The investigation found actual provider mechanisms instead of inventing SDK capabilities. **The lifecycle contract does not need redesign.**&#x20;

plan1(1).md



The findings are consistent with Fly's current documentation:

- Sleep is automatic: warm pause followed eventually by cold pause.
- Any incoming request can wake a Sprite.
- The Tasks API provides genuine holds through the Unix socket.
- Deleting or expiring a hold releases the Sprite to pause.
- A hold has a maximum one-hour lifetime and needs refreshing for longer work.&#x20;

  [image](https://www.google.com/s2/favicons?domain=https://docs.fly.io\&sz=32)

  Fly.io+1



### **Three implementation cautions**

1\. **Distinguish warm and cold experiments.** The proposed 45-second idle test primarily investigates warm pause. A separate longer-idle experiment is needed to establish cold behavior. Do not claim both from one wake test.

2\. **Treat sleep as an observed transition, not a commanded operation.** Releasing the Tasks hold does not force sleep. Record the release and subsequent observed behavior separately. The vendor documentation also contains different descriptions of process survival, so verify actual behavior rather than assuming a detached TTY process is representative.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.fly.io\&sz=32)

Fly.io+1



3\. **Account for hold expiration.** Record the hold's expiry and plan renewal for long-running agent work. The one-hour expiration is a useful safety backstop, not a long-term execution strategy.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.fly.io\&sz=32)

Fly.io



### **Execution decision**

Proceed with the smallest live proof in the attached sequence. Preserve the existing contract, record actual observations, and stop on surprises.

**Live provisioning, connection, disconnection, wake and destruction still require their respective owner approvals.** This review approves the approach, not those individual operations.

No further architectural review cycle is needed before execution.