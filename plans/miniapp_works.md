## prompt

brief concise reply only, does our Miniapp Studio development so far use the Miniapp framework, is there any indication that it is doing so

## answer

**Yes, absolutely.** Miniapp Studio is actively using the Miniapp framework, not merely planning to use it.

Evidence so far:

- **Dependency:** Studio pins `github.com/PithomLabs/miniapp` at `73351a8`.
- **Execution:** P2 runner invokes operations through `runtime.Service.Invoke`.
- **Registration:** Studio tools use Miniapp's static `registry.Register`.
- **Execution model:** The generic runner uses the framework's Class A executor.
- **Infrastructure:** P0 successfully demonstrated framework invocation and approval resumption.
- **P4 server:** Its wiring uses the framework's policy, registry, executors, and runtime service.

The `spec/` package being standard-library-only is intentional. It keeps specification validation independent while the actual application execution relies on Miniapp.

**Conclusion: Miniapp Studio is a real application built on the Miniapp framework.**