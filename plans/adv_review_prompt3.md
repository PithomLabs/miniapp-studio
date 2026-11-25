# Adversarial Code Review — Miniapp Studio P1 Specification and Validator

- Scope: `spec/` package only (P1). Read-only review; no code modified.
- Reviewer posture: principal Go architect / application-security reviewer.
- Date: 2026-11-24. Studio root: `/home/chaschel/Documents/go/miniapp-studio`.

## Executive verdict

**APPROVE P1** — with four MAJOR findings to contain and track into P2.

I attempted to defeat every foundational P1 claim with ~90 hostile inputs executed
against the real code (not inferred from comments). The core trust-boundary result
holds: **no JSON input I could construct validates (`len(errs) == 0`) while
containing an external write.** Every non-GET method, every non-read effect, every
`externalWrites` value other than `disabled`, every unknown step/trigger
discriminator, and every unknown struct field is rejected. Hashing is stable,
mutation-safe, formatting-independent, and owner-sensitive. Tenant ownership cannot
be omitted. The package is standard-library-only with zero framework coupling.

The required precision, however, is this: **writes are thoroughly *rejected by the
validator*, not *unrepresentable in the data model*.** The Go types use plain
`string` fields, `Validate` returns a populated spec alongside errors, and any
caller that ignores `errs` (or constructs `MiniappSpec` directly) holds a
write-laden value. Code comments overclaim structural unrepresentability. That gap
is real but it is a validator-rejection guarantee plus a P2 enforcement handoff,
not a P1 structural defect — no JSON-schema package can make a string field
unconstructible in Go, and demanding that of P1 would block the foundation on an
unattainable property. The four MAJOR findings below (representability/API shape,
silently ignored per-type fields, path-traversal asymmetry, name-only secret
scanning) are containable without restructuring P1 and must be carried as explicit
P2 requirements. No finding meets the BLOCKER bar (a write accepted in a valid
spec, a validation bypass, unstable hash identity, omittable ownership, or
framework coupling).

## Invariant scorecard

| Invariant | Verdict | Code evidence | Test evidence | Remaining limitation |
| --------- | ------- | ------------- | ------------- | -------------------- |
| Read-only structure | PASS (validator-rejected, see F1) | `spec/validate.go:194-199` method/effect equality; `:580-583` externalWrites; `:480-483` step discriminator; `:447-449` trigger | `invalid_capability.json`, `invalid_unapproved_send.json`, `TestDeliberateRejection` asserts method+effect | Rejected, not unrepresentable: plain strings, invalid spec returned non-nil (`validate.go:103-136`) |
| Type/schema integrity | PASS with MAJOR gaps (F2, F6) | Closed maps `validate.go:64-84`; exact version `122-125`; zero-value policy per field | `invalid_version.json`, `invalid_unknown_section.json`, `TestMalformedJSON` | Irrelevant per-type fields ignored; binding sub-paths/columns unchecked; direct construction bypasses |
| Canonical hashing | PASS with MINOR gap (F7) | `spec/hash.go:14-26` canonical form; `:32-50` hash minus contentHash | 5 hash tests incl. round-trip stamp/tamper | float64 collapses integers >2^53; key-sort rests on stdlib guarantee |
| Validator correctness | PASS with MINOR gaps (F5) | Strict decode `:116-121`; secrets-first `:114`; fixed section order `:126-134`; sorted map walks `:142-146`, `:487-491` | `TestDeterminism`, 13-fixture matrix, secret case-insensitivity | Duplicates last-win; field names case-insensitive; no Decoder EOF check (masked by pre-scan) |
| Tenant ownership | PASS (schema only) | Required `validate.go:174-176`; single `Owner` field `spec/types.go:100-107`; hashed with content | Indirect (no committed owner-variant test) | Free-form string, no normalization; persistence isolation is P2+ work |
| Dependency isolation | PASS | Imports in `spec/hash.go:3-8`, `spec/validate.go:3-10` are stdlib-only; `go.mod` has zero requires | `go list -m all` single module; `go list -deps` stdlib-only | None |
| Regression coverage | PASS with MINOR gap (F9) | 11 test funcs, 13 invalid subcases, deliberate-reject test | `go test`, `go test -race` green | Breadth gaps: no committed tests for PUT/PATCH/DELETE, null sections, dup keys, trailing docs, conn-secret |

## Required conclusion (prompt §2.A)

**A write in v1 is rejected by the validator; it is not unrepresentable in the
data model.** The precise boundary:

- *JSON boundary (enforced):* any document containing `method != "GET"`,
  `effect != "read"`, `externalWrites != "disabled"`, an unknown step type, or
  an unknown trigger fails validation. Verified empirically across ~90 hostile
  variants (see Verification record). A P2 runner that refuses to execute unless
  `len(errs) == 0` cannot receive a write through this boundary.
- *Go boundary (not enforced, cannot be at reasonable cost):*
  `CapabilityOp{Method: "POST", Effect: "write"}` compiles; `Validate` returns
  it populated next to 2 errors; `ContentHash` will happily hash an unvalidated
  struct. Any in-process construction path that skips `Validate` is outside the
  P1 guarantee by design.
- *Runtime boundary (does not exist yet):* no interpreter exists to confuse, so
  no runtime write is possible today; P2 must re-validate at admission and at
  execution time and must dispatch strictly on step type.

## Findings

### F1 — Writes are representable in Go; invalid specs are returned usable (MAJOR, non-blocking)

- Severity: MAJOR. Does not block P2 (containable via handoff + comment/API correction).
- Location: `spec/types.go:38-40` (comment claims structural enforcement),
  `spec/types.go:224-232` (`Method`/`Effect` are plain strings),
  `spec/validate.go:103-136` (returns `&spec` with `errs != nil`),
  `spec/validate.go:582` (message text "writes are unrepresentable until G-01").
- Repro (executed): `Validate` of an op with `method POST` / `effect write`
  returns `spec != nil` with `spec.Capabilities.Operations[0].Method == "POST"`.
  `doc=null` and `doc={}` return non-nil specs with 9 errors. Direct
  construction `MiniappSpec{...Method: "POST", Effect: "write"...}` compiles and
  `ContentHash` hashes it. Only the strict-decode path (`validate.go:118-121`)
  returns nil.
- Root cause: the "closed vocabulary" is a validator map lookup over exported
  string fields, not a type-level constraint; and the API shape invites
  `spec, _ := Validate(raw)`.
- Durable correction: (1) fix the overclaim comments (`types.go:38-40`,
  `validate.go:582`, `CapabilityOp` doc) to say "rejected by the validator";
  document on `Validate` that a non-empty error slice forbids any use of the
  returned spec; (2) P2 must gate every admission and execution on
  `len(errs) == 0` (see P2 handoff); (3, optional, owner decision) change
  `Validate` to return `nil` spec whenever `errs` is non-empty, or add a
  struct-level `ValidateSpec(*MiniappSpec)` entry for constructed/transformed
  specs. Do not add a third-party validation framework.
- Regression test: `TestInvalidSpecMustNotBeUsed` — write-op JSON yields errors
  and (after correction 3) a nil spec; plus a test that a programmatically
  built write-op struct fails struct-level validation. Fails before, passes after.
- Blocks P2: no, provided the P2 handoff gate is honored.

### F2 — Fields irrelevant to a step/component type are silently accepted (MAJOR, non-blocking)

- Severity: MAJOR. Does not block P2 (no accepted input gains an *effective*
  write, since the referenced write capability itself cannot validate).
- Location: `spec/validate.go:476-538` (`checkStep` never rejects inapplicable
  fields), `spec/validate.go:366-398` (`checkComponent` same),
  `spec/types.go:204-214` (shared `Step` shape), `spec/types.go:178-186`
  (shared `Component` shape).
- Repro (executed, all `nerr=0`): `transform` carrying
  `operation:"mail.send"`, `path:"/send"`, `condition`, `message`,
  `maxIterations`; `branch` and `bounded-loop` carrying `operation:"mail.send"`;
  `notify` carrying `operation:"mail.send"`; `form` carrying
  `binding:"ghost_query"` plus `columns` (undeclared reference accepted because
  forms only check `action`); `table` carrying `action` (ignored).
- Root cause: per-type checks assert required fields but never forbid
  inapplicable ones, so `DisallowUnknownFields` (which only sees the shared
  struct) cannot help. This is the "contradictory declarations" class from the
  prompt: undeclared operations and ghost bindings ride inside known-but-ignored
  fields of otherwise valid specs.
- Durable correction: define a per-type field matrix (in `types.go` comments and
  enforced in `checkStep`/`checkComponent`): `operation`/`path` only on
  `connector-read`; `condition` only on `branch`; `maxIterations` only on
  `bounded-loop`; `message` only on `notify`/`finish`/`confirm`; keep `output`
  allowed on all steps (reference source) and `input` validated wherever
  present; components: `form`/`action-button` reject `binding`/`columns`/`fields`,
  `table` rejects `action`, binding-types reject `action`/`columns`/`fields`
  except their consumed set. P2 must additionally dispatch strictly on type and
  ignore nothing silently.
- Regression tests: `TestStepIrrelevantFieldsRejected` (each matrix violation
  yields `E_WORKFLOW`), `TestComponentIrrelevantFieldsRejected` (each yields
  `E_COMPONENT`). Fail before, pass after.
- Blocks P2: no, but P2 must not interpret ignored fields (handoff item).

### F3 — Step request paths and component targets admit `..` traversal (MAJOR, non-blocking)

- Severity: MAJOR. Does not block P2 (read-only GET surface; gateway/P2 owns
  final resolution — but the asymmetry is exactly an allowlist-bypass shape).
- Location: `spec/validate.go:502-506` (step path: only absolute-prefix plus
  `pathAllowed`), `spec/validate.go:395-397` (target: only absolute-prefix),
  contrast `spec/validate.go:219-228` (`validPathPattern` rejects `..` and
  whitespace for *patterns*).
- Repro (executed): op allowlist `["/*"]` plus step path
  `/files/../../admin/purge` validates with `nerr=0`; `target:"/../evil"`
  validates with `nerr=0`. Control: exact allowlist correctly rejects
  non-matching traversal with "not in the declared allowlist". `page.path`
  correctly rejects `..` (`validate.go:353-356`), proving the omission is local
  to step paths and targets.
- Root cause: two different path predicates; the stricter one was applied to
  patterns, pages, and connections but not to step request paths or targets.
- Durable correction: apply one shared predicate (absolute, no `..`, no
  whitespace/controls) to step paths and targets; P2/gateway must additionally
  normalize (dot-segment removal) and re-check membership *after*
  normalization, never match-then-normalize. Decide and document the policy for
  `?`/`#` (currently allowed everywhere paths appear).
- Regression tests: `TestStepPathTraversalRejected` (`/*` + `/x/../admin`
  yields `E_CAPABILITY`), `TestTargetTraversalRejected`. Fail before, pass after.
- Blocks P2: no, provided P2 normalizes and re-checks (handoff item).

### F4 — `connection` accepts credential-shaped values; secret scan is name-only (MAJOR, non-blocking)

- Severity: MAJOR. Does not block P2 (real fix is server-side resolution).
- Location: `spec/types.go:221-224` (comment: "must never carry a secret, URL,
  or credential — the secret scanner rejects such fields"),
  `spec/validate.go:190-193` (format check only: non-empty, ≤128, no
  whitespace, no `://`), `spec/validate.go:139-159` (scans field *names* only),
  `spec/validate.go:58-62` (name blocklist).
- Repro (executed): `"connection": "sk-live-ABCDEF1234567890abcdef"` validates
  with `nerr=0`. Controls: URL and whitespace connections correctly rejected;
  secret *keys* inside free-form maps (`input.password`, `fixture.input.api_key`)
  correctly flagged `E_SECRET`.
- Root cause: the comment promises value protection the code never implements;
  value-shape detection is inherently heuristic and was scoped out.
- Durable correction: (1) correct the comment to "secret-name scanner; values
  are not inspected"; (2) add a cheap, testable tripwire blocklist of known
  credential prefixes (`sk-`, `AKIA`, `xoxo-`, `ghp-`, `gsk-`, `Bearer ` is
  already excluded as whitespace/URL-adjacent) — documented as tripwire, not
  proof; (3) P2/server resolves every `connection` against a server-side
  registry and rejects unknown IDs (the actual fix — handoff item). Do not
  build entropy-based secret detection into P1.
- Regression test: `TestConnectionKnownSecretPrefixesRejected` (each prefix
  yields `E_CAPABILITY`). Fails before, passes after. Plus a doc-locking test
  that an opaque well-formed ID validates.
- Blocks P2: no, provided server-side resolution is a P2 requirement.

### F5 — Decoder strictness gaps: duplicates, case-insensitivity, missing EOF check (MINOR)

- Severity: MINOR. No bypass: the surviving value is always itself validated,
  and trailing documents are already rejected by the pre-scan.
- Location: `spec/validate.go:103-121` (two-parse design; `json.Decoder` with
  `DisallowUnknownFields` but no second `Decode` expecting `io.EOF`).
- Repro (executed): duplicate `"method":"GET","method":"POST"` → rejected as
  POST (last wins); reversed duplicates → accepted as GET (last wins, valid);
  duplicate `"slug"` → accepted; `"Method"`, `"APP"`, `"SchemaVersion"` field
  names decode fine (Go case-insensitive matching) — and hostile values under
  variant names (`"Method":"POST"`, `"Effect":"write"`) are still rejected on
  value; trailing document/garbage/NDJSON → `E_SCHEMA invalid JSON` (from the
  `json.Unmarshal` pre-scan at `:111`, which masks the missing EOF check).
- Root cause: Go `encoding/json` semantics (last-win duplicates,
  case-insensitive fields) plus a one-`Decode` call. The pre-scan accidentally
  but reliably covers trailing data today.
- Durable correction: document the exact decoding contract on `Validate`
  (case-insensitive field names, last-wins duplicates, winning value always
  validated); add the defense-in-depth second `Decode` expecting `io.EOF` so
  the strict path does not depend on the pre-scan for trailing-data rejection.
  Do not hand-roll a duplicate-detecting JSON parser — cost exceeds benefit
  while values are validated.
- Regression tests: `TestTrailingDocumentRejected`, `TestDuplicateKeysLastWin`,
  `TestCaseInsensitiveFieldNames` (lock documented behavior). Pass before and
  after (EOF check is defense-in-depth).
- Blocks P2: no.

### F6 — Binding sub-paths, table columns, and duplicate step outputs unchecked (MINOR)

- Severity: MINOR. Fail-late correctness, no trust-boundary bypass.
- Location: `spec/validate.go:402-418` (`checkBinding` resolves only the head
  segment), `spec/validate.go:372-380` (table requires ≥1 column, never resolves
  any), `spec/validate.go:462-464` (outputs recorded without uniqueness).
- Repro (executed against `valid_invoice_review.json`): table
  `columns: ["nope_not_a_field"]` → `nerr=0`; binding
  `"pending.nope_not_a_field"` → `nerr=0`; two steps sharing `output:"dup"` →
  `nerr=0`. Control: duplicate step *IDs* correctly rejected.
- Root cause: binding validation stops at the query/entity head; column/field
  identifiers have no declared output shape to check against in v1.
- Durable correction: resolve binding head to its entity (query→entity or
  entity), then check the first sub-path segment and every table `columns` /
  record `fields` entry against that entity's declared fields; reject duplicate
  step `output` names within a workflow. Keep the grammar (`rxBinding`,
  `checkRef`) unchanged.
- Regression tests: `TestBindingSubPathResolved`, `TestTableColumnsResolved`,
  `TestDuplicateStepOutputRejected`. Fail before, pass after.
- Blocks P2: no (P2 renderer would fail late instead of at admission).

### F7 — Hash number model: float64 collapses integers beyond 2^53 (MINOR)

- Severity: MINOR. Validator and hash agree (both see float64), so no
  accept/hash disagreement; affects only very large integers in free-form
  values.
- Location: `spec/hash.go:14-26` (`any` decode → float64 → re-encode),
  `spec/hash.go:32-50` (`ContentHash` via struct round-trip).
- Repro (executed): fixture inputs `9007199254740993` vs `9007199254740992`
  produce identical canonical JSON (`{"n":9007199254740992}`) and identical
  content hashes. Controls (all passing): repeated hashing stable with
  `reflect.DeepEqual`-verified non-mutation; pretty-print/key-permutation
  invariant; empty-vs-null equivalence; HTML characters (`a<b>&"c"`) stable;
  owner change flips the hash; key sorting recursive
  (`{"b":1,"a":{"d":4,"c":3}}` → `{"a":{"c":3,"d":4},"b":1}`).
- Root cause: `encoding/json` default number decoding; documented Go behavior,
  but unstated in this package. Key-sorted canonicalization likewise rests on
  the stdlib's documented map-key sorting rather than explicit code.
- Durable correction: document the number model and the stdlib-sorting
  reliance in `hash.go`; do not migrate to `json.Number` (it would change the
  canonical form and require a normalization spec for no P1 benefit).
- Regression tests: `TestCanonicalJSONNumberNormalization` (`1e3` vs `1000`
  hash equal), `TestContentHashDoesNotMutate` (`DeepEqual` before/after).
- Blocks P2: no. Note for P2: `ContentHash` (struct-normalized,
  `omitempty`-aware, contentHash-excluded) is the authoritative identity — not
  `CanonicalJSON` of raw input, which intentionally differs (includes the hash
  field, preserves explicit empties).

### F8 — Owner is free-form: padding and case variants accepted (MINOR)

- Severity: MINOR for P1 (canonical tenant form depends on the auth system P1
  cannot know); P2 must not treat the string as an identity proof.
- Location: `spec/validate.go:174-176` (non-blank, ≤256 only),
  `spec/types.go:100-107` (single `Owner` field — unambiguous representation).
- Repro (executed): `"owner": " acme"` and `"owner": "ACME"` validate clean;
  absent/null/whitespace-only owners correctly yield `E_APP`; owner change
  flips the content hash (probed). No conflicting second owner representation
  exists anywhere in the types.
- Root cause: v1 correctly requires *presence* but defines no canonical form.
- Durable correction: document `Owner` as an opaque tenant label whose
  canonical form is defined by the server; P2 derives the tenant from auth
  context and compares/overwrites (handoff item). Do not invent P1
  normalization rules (lowercasing/trimming) that could disagree with the
  future auth system.
- Regression test: `TestOwnerPresenceVariants` (absent/null/blank → `E_APP`;
  present → accepted). Passes before and after (locks the contract).
- Blocks P2: no.

### F9 — Committed-test breadth gaps (MINOR)

- Severity: MINOR. The critical invariant *is* tested (POST+write fixtures and
  `TestDeliberateRejection` asserting both method and effect violations); these
  gaps are breadth, not absence.
- Location: `spec/validate_test.go`, `spec/hash_test.go`.
- Missing committed coverage (all verified manually via probes in this review,
  none covered by `go test`): PUT/PATCH/DELETE/lowercase/empty/null methods and
  effects; `externalWrites` variants beyond `approval`; null vs omitted for all
  nine sections; degenerate documents beyond `{bad json` (`""`, `null`, `[]`,
  `123`, `"str"`, `{}`, `[{}]` — all behave, unpinned); duplicate keys;
  trailing documents; case-variant field names; multi-error determinism beyond
  the two current fixtures; `ContentHash` non-mutation (`DeepEqual`);
  credential-shaped `connection` (documents F4's current acceptance or its
  prefix-tripwire fix).
- Durable correction: add a table-driven `TestWriteShapedInputsRejected`
  (method × effect × policy matrix), `TestNullAndOmittedSections`,
  `TestDegenerateDocuments`, and extend `TestDeterminism` to a secret-bearing
  multi-error input. No test should merely exercise a path — each must assert
  the code *and* path (the `TestDeliberateRejection` style).
- Blocks P2: no.

## What I verified as genuinely sound (PASS details)

- **Method/effect/policy/step/trigger rejection.** 30+ hostile variants, all
  rejected with precise codes and paths; no false acceptance found.
- **Unknown-field rejection at every struct level** (top-level, app, trigger,
  step, operation) via `DisallowUnknownFields` — probed, not assumed.
- **Null/omitted equivalence is total and consistent**: all nine sections behave
  identically under `null` and omission (required ones error, optional ones
  default). No `null`-smuggled bypass.
- **No-panic robustness**: empty, scalar, array, null, truncated, and
  type-mismatched documents all yield `E_SCHEMA`/semantic errors; `checkRef`
  slicing is guarded by regex match; `mustMarshal`'s `panic` is unreachable for
  JSON-derived values.
- **Determinism**: fixed section order, sorted secret-scan and input-ref walks,
  no validators range over maps for output; multi-error ordering stable across
  runs (probed with `reflect.DeepEqual`).
- **13-code taxonomy is closed and consistent**: all 13 constants
  (`validate.go:17-31`) are used; `Error()` embeds code+path+message; error
  strings echo spec content (contains no secrets by policy) — P2 should still
  treat them as tenant content, not cross-tenant-safe logs.
- **Hash**: excludes exactly `metadata.contentHash` (`hash.go:41-43`);
  `omitempty` normalization is semantically correct (explicit `0`/`""`/`[]`
  hash identically to omitted); validation and hashing agree (checkHash runs
  last over the same struct).
- **Tenant foundation**: exactly one owner representation, required, hashed;
  within-spec uniqueness checks are correctly spec-scoped (capabilities,
  entities, queries, workflows, steps, pages, fields, fixtures, roles) —
  nothing imposes or omits a global-uniqueness claim P1 cannot keep.
- **Dependency isolation**: `go list -m all` reports the single module;
  `go list -deps ./spec` reports stdlib plus the spec package itself; no
  Miniapp runtime, Fly, Sprite, provider, or DB imports. (The string
  "miniapp" appears only in `miniappspec/v1` literals.)
- **No input mutation**: `Validate` reads bytes; `ContentHash` verified
  non-mutating via `DeepEqual`; the validator accumulates into its own struct.

## Verification record

Environment: sandbox shell with Go toolchain; default `GOCACHE` was read-only,
so Go commands ran with `GOCACHE=/tmp/gocache-adv
GOPATH=/tmp/gopath-adv GOTOOLCHAIN=local`. All probes ran from `/tmp/advprobe`
(a scratch module with a `replace` directive); **no repository file was created,
modified, or deleted except this review**, per instruction.

| Command | Result |
| ------- | ------ |
| `gofmt -l .` | clean (empty output, exit 0) |
| `go vet ./...` | pass (exit 0; initial run failed only on sandbox read-only default cache) |
| `go build ./...` | pass (exit 0) |
| `go test ./...` | pass (`spec` ok; root has no test files) |
| `go test -race -count=1 -v ./spec/` | pass: 11 test funcs, 13 `TestInvalidFixtures` subtests, 0 failures |
| `go list -m all` | single module `github.com/PithomLabs/miniapp-studio` |
| `go list -deps ./spec` | stdlib packages + spec package only |
| CI gates in `.github/workflows/ci.yml` (`gofmt`, `vet`, `build`, `test`) | all four executed locally and passing |
| Adversarial probe suite (`/tmp/advprobe`, 3 runs, ~90 inputs) | all findings above reproduced from live output; first run's hash section was re-executed after a probe-side index bug (repo untouched) |
| `spec/testdata` file count | **14 files** (13 invalid + 1 valid), not 15 as the prompt states — description discrepancy, not a code defect |
| Test-function count | 11 (`validate_test.go:6`, `hash_test.go:5`) — matches the claim |
| Not run | PostgreSQL/P0 Docker steps — out of scope for this P1 review (stdlib-only suite needs no DB) and unavailable from the sandbox |

## P2 handoff (must-preserve requirements)

1. **Gate everything on validation.** Refuse to admit, render, or execute unless
   `len(errs) == 0`; never use a spec returned alongside errors (F1).
   Re-validate at execution time — admission-time validity is not
   execution-time validity — and bind each run to the `ContentHash` it started with.
2. **Preserve the v1 read-only boundary at runtime.** The runner issues GET
   requests only, through declared capabilities, to allowlisted paths; the
   renderer adds no write affordance. Any future write needs a schema-version
   bump and the G-01 lifecycle — never a quiet interpreter extension.
3. **Dispatch strictly on step type; resolve connections server-side.**
   Ignore nothing silently (F2); resolve every `connection` against a
   server-side registry and reject unknown IDs and inline credentials (F4).
4. **Normalize and re-check paths after normalization** for step paths and
   targets: reject `..`/whitespace/controls, decide `?`/`#` policy, and never
   match-then-normalize (F3). Treat `/*` as the operator's explicit
   broad-allowlist choice, and surface its breadth at admission.
5. **Derive tenancy from auth context.** Compare (or overwrite) `app.owner`
   against the server-derived tenant; define the canonical owner form there
   (F8). Future persistence scopes every record by (tenant, app ID); P1
   provides the pair, not the isolation.
6. **Bound input before validating.** `Validate` double-parses with no size
   cap; P2 must enforce a max request size ahead of it.
7. **Use `ContentHash` as the identity**, never `CanonicalJSON` of raw input
   (different normalization, includes the hash field). Stay clear of integers
   beyond 2^53 in free-form values (F7).
8. **Track F1–F4 corrections** (comment accuracy, per-type field matrix, path
   predicate unification, connection tripwire + registry) as P2-adjacent work;
   each ships with its named regression test. F5–F9 are document-and-pin.

## Final decision

**Proceed: P1 is accepted as the foundation for P2.** No foundational or
structural blocker was found after adversarial testing of the hostile-input
boundary, the hash identity, ownership, and dependency isolation. Start P2 after
the separate P0 dependency gate (Docker Postgres access) passes. No further
general review cycle is requested; the MAJOR findings above are tracked work
with durable corrections and named regression tests, not re-review triggers.
