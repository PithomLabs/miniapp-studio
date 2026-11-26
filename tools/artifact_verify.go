package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/mod/modfile"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp-studio/spec"
	"github.com/PithomLabs/miniapp-studio/store"
)

// CheckSetVersion versions the verification check set. Any check
// added, removed, or redefined bumps this — evidence rows key on it,
// so re-verify under a new set writes new evidence instead of
// colliding with the old.
const CheckSetVersion = "v1"

// verifyCheck is one evidence entry: what ran and whether it passed.
type verifyCheck struct {
	Check   string `json:"check"`
	Pass    bool   `json:"pass"`
	Code    string `json:"code,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
}

type artifactVerifyIn struct {
	ArtifactID string `json:"artifact_id"`
}

// artifactVerify deterministically verifies one bundle artifact
// against studio-artifact/v1: identity, contract, spec, admission,
// dependencies, secrets, and ownership. Zero code execution, zero
// network. Evidence persists atomically with the verified_at flip;
// failures persist evidence and leave the row unverified.
func (d Deps) artifactVerify(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in artifactVerifyIn
	if err := decode(call.Input, &in); err != nil || in.ArtifactID == "" {
		return nil, errInvalidArguments()
	}
	return verifyArtifact(call.Context, call.Tx, tenant, in.ArtifactID)
}

// VerifyOutcome is the persisted result of one verification.
type VerifyOutcome struct {
	Passed     bool
	EvidenceID string
	Checks     json.RawMessage
}

// VerifyBundleTx runs the deterministic verification over one
// bundle's stored bytes and persists evidence, the verified_at flip,
// and the stream receipt in the caller's transaction. It is the
// shared core behind the artifact.verify tool and the sender's
// fetch-completion hook: both advance the same bytes through the
// same checks. A bundle that fails verification returns
// Passed=false with nil error — only storage failures error.
func VerifyBundleTx(ctx context.Context, tx pgx.Tx, tenant, artifactID string) (VerifyOutcome, error) {
	return verifyBundleTx(ctx, tx, tenant, artifactID)
}

func verifyArtifact(ctx context.Context, tx pgx.Tx, tenant, artifactID string) (json.RawMessage, error) {
	out, err := verifyBundleTx(ctx, tx, tenant, artifactID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("studio: unknown artifact")
		}
		return nil, err
	}
	var checks []verifyCheck
	if jerr := json.Unmarshal(out.Checks, &checks); jerr != nil {
		return nil, jerr
	}
	return marshalMap(map[string]any{
		"verified": out.Passed, "artifact_id": artifactID,
		"checks": checks, "evidence_id": out.EvidenceID,
	})
}

func verifyBundleTx(ctx context.Context, tx pgx.Tx, tenant, artifactID string) (VerifyOutcome, error) {
	v := &verifier{tenant: tenant, artifactID: artifactID}
	data, err := store.LoadBundle(ctx, tx, tenant, artifactID)
	if err != nil {
		// Missing row: no evidence can persist (FK), so fail as
		// invalid input rather than a verification.
		if err == pgx.ErrNoRows {
			return VerifyOutcome{}, pgx.ErrNoRows
		}
		v.fail("load", artifact.EContract, "$", err.Error())
		return v.finish(ctx, tx)
	}
	if data.Kind != "bundle" {
		v.fail("load", artifact.EContract, "$", "only bundle rows are verifiable")
		return v.finish(ctx, tx)
	}
	members := data.Members
	env, errs := artifact.ParseEnvelope(data.Envelope)
	if errs != nil {
		for _, e := range errs {
			v.fail("contract", e.Code, e.Path, e.Message)
		}
		return v.finish(ctx, tx)
	}
	v.pass("contract")
	v.checkOwnership(data, env)
	if !v.checkClaimEnvelopeAgreement(env, members) {
		return v.finish(ctx, tx)
	}
	root, errs := artifact.ComputeRoot(data.Envelope, members)
	if errs != nil {
		for _, e := range errs {
			v.fail("identity", e.Code, e.Path, e.Message)
		}
		return v.finish(ctx, tx)
	}
	if root != data.Digest {
		// Identity failure aborts: untrustworthy bytes invalidate
		// every downstream check. Other failures accumulate so one
		// verification reports everything wrong with the bundle.
		v.fail("identity", artifact.EDigest, "$", "recomputed identity does not match the pinned digest")
		return v.finish(ctx, tx)
	}
	v.pass("identity")
	entries := bundleEntries(env, data.Envelope, members)
	if errs := artifact.ValidateBundle(entries); len(errs) != 0 {
		for _, e := range errs {
			v.fail("structure", e.Code, e.Path, e.Message)
		}
	} else {
		v.pass("structure")
	}
	v.checkSpec(members[artifact.SpecFileName])
	v.checkManifest(members[artifact.ManifestFileName])
	v.checkGoMod(members[artifact.GoModFile])
	v.checkGoSum(members[artifact.GoSumFile])
	v.checkImports(members)
	v.checkSecrets(members)
	v.checkBinding(env, members)
	return v.finish(ctx, tx)
}

// verifier accumulates checks and persists evidence once.
type verifier struct {
	tenant     string
	artifactID string
	checks     []verifyCheck
	failed     bool
}

func (v *verifier) pass(check string) {
	v.checks = append(v.checks, verifyCheck{Check: check, Pass: true})
}

func (v *verifier) fail(check string, code artifact.Code, path, message string) {
	v.failed = true
	v.checks = append(v.checks, verifyCheck{Check: check, Pass: false,
		Code: string(code), Path: path, Message: message})
}

// checkClaimEnvelopeAgreement requires the claim member set to equal
// the envelope member set exactly: every fetched blob bound by the
// envelope, no unbound bytes, nothing missing.
func (v *verifier) checkClaimEnvelopeAgreement(env *artifact.Envelope, members map[string][]byte) bool {
	if len(members) != len(env.Files) {
		v.fail("identity", artifact.EBinding, "files", "claim member set differs from the envelope")
		return false
	}
	for _, e := range env.Files {
		if _, ok := members[e.Path]; !ok {
			v.fail("identity", artifact.EBinding, e.Path, "envelope member bytes missing")
			return false
		}
	}
	return true
}

// bundleEntries rebuilds the file-entry list for layout validation
// from stored bytes (sizes re-derived, never trusted).
func bundleEntries(env *artifact.Envelope, envRaw []byte, members map[string][]byte) []artifact.FileEntry {
	entries := make([]artifact.FileEntry, 0, len(members)+1)
	for _, e := range env.Files {
		entries = append(entries, artifact.FileEntry{Path: e.Path, Digest: e.Digest, Size: int64(len(members[e.Path]))})
	}
	entries = append(entries, artifact.FileEntry{Path: artifact.EnvelopeFileName, Size: int64(len(envRaw))})
	return entries
}

func (v *verifier) checkOwnership(data *store.BundleData, env *artifact.Envelope) {
	if env.TenantID != v.tenant {
		v.fail("ownership", artifact.EBinding, "tenant_id", "envelope tenant does not match the row owner")
		return
	}
	if env.BuildID == "" || env.BuildID != data.BuildID {
		v.fail("ownership", artifact.EBinding, "build_id", "envelope build does not match the row owner")
		return
	}
	v.pass("ownership")
}

func (v *verifier) checkSpec(raw []byte) {
	if len(raw) == 0 {
		v.fail("spec", artifact.EContract, artifact.SpecFileName, "spec bytes missing")
		return
	}
	parsed, errs := spec.Validate(raw)
	if len(errs) != 0 {
		for _, e := range errs {
			v.fail("spec", artifact.Code(e.Code), e.Path, e.Message)
		}
		return
	}
	if parsed.App.Owner != v.tenant {
		v.fail("spec", artifact.EBinding, "app.owner", "spec owner does not match the tenant")
		return
	}
	v.pass("spec")
}

func (v *verifier) checkManifest(raw []byte) {
	if len(raw) == 0 {
		v.fail("manifest", artifact.EContract, artifact.ManifestFileName, "manifest bytes missing")
		return
	}
	var mf contract.MiniappManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		v.fail("manifest", artifact.EContract, artifact.ManifestFileName, "invalid JSON: "+err.Error())
		return
	}
	if err := mf.Validate(); err != nil {
		v.fail("manifest", artifact.EContract, artifact.ManifestFileName, err.Error())
		return
	}
	for _, tool := range mf.Tools {
		if tool.Substrate != contract.SubstrateGo {
			v.fail("manifest", artifact.ESubstrate, string(tool.Name),
				fmt.Sprintf("substrate %q refused: Studio admits go only", tool.Substrate))
			return
		}
	}
	v.pass("manifest")
}

func (v *verifier) checkGoMod(raw []byte) {
	if len(raw) == 0 {
		v.fail("gomod", artifact.EContract, artifact.GoModFile, "go.mod bytes missing")
		return
	}
	f, err := modfile.Parse(artifact.GoModFile, raw, nil)
	if err != nil {
		v.fail("gomod", artifact.EDeps, artifact.GoModFile, "go.mod does not parse: "+err.Error())
		return
	}
	if f.Module == nil || !artifact.ValidModulePath(f.Module.Mod.Path) {
		v.fail("gomod", artifact.EBinding, artifact.GoModFile, "module path outside studio-artifact/<tenant>/<app>")
		return
	}
	if f.Go == nil || f.Go.Version != artifact.PinnedGoVersion {
		v.fail("gomod", artifact.EDeps, artifact.GoModFile, "go directive must equal the pinned toolchain")
		return
	}
	// Exactly one DIRECT require (the pinned miniapp). Indirect
	// requires are tidy's normal output and carry no authority: a
	// direct evil import dies at the import check, an unimported
	// entry is inert, and the offline sandbox build resolves only
	// the pre-seeded closure. (D2 correction to the plan's
	// "go.sum closure match" bullet: static closure pinning adds
	// no control beyond these three.)
	var direct []*modfile.Require
	for _, r := range f.Require {
		if !r.Indirect {
			direct = append(direct, r)
		}
	}
	if len(direct) != 1 || direct[0].Mod.Path != artifact.MiniappModulePath ||
		direct[0].Mod.Version != artifact.PinnedMiniapp {
		v.fail("gomod", artifact.EDeps, artifact.GoModFile, "require must be exactly the pinned miniapp")
		return
	}
	if len(f.Replace) != 0 || len(f.Retract) != 0 || f.Toolchain != nil || len(f.Exclude) != 0 {
		v.fail("gomod", artifact.EDeps, artifact.GoModFile, "replace/retract/toolchain/exclude forbidden")
		return
	}
	v.pass("gomod")
}

func (v *verifier) checkGoSum(raw []byte) {
	if len(raw) == 0 {
		v.fail("gosum", artifact.EContract, artifact.GoSumFile, "go.sum bytes missing")
		return
	}
	body := string(raw)
	for _, line := range []string{artifact.MiniappGoSumZip, artifact.MiniappGoSumMod} {
		if !strings.Contains(body, line+"\n") && !strings.HasSuffix(body, line) {
			v.fail("gosum", artifact.EDeps, artifact.GoSumFile, "pinned miniapp go.sum lines missing")
			return
		}
	}
	v.pass("gosum")
}

func (v *verifier) checkImports(members map[string][]byte) {
	fset := token.NewFileSet()
	for _, path := range sortedKeys(members) {
		if !strings.HasPrefix(path, artifact.AppDir) || !strings.HasSuffix(path, artifact.GoSuffix) {
			continue
		}
		f, err := parser.ParseFile(fset, path, members[path], parser.ImportsOnly)
		if err != nil {
			v.fail("imports", artifact.ELang, path, "does not parse: "+err.Error())
			return
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if artifact.IsBannedImport(p) {
				v.fail("imports", artifact.EDeps, path, "banned import "+p)
				return
			}
			if !artifact.IsStdlib(p) && !strings.HasPrefix(p, artifact.MiniappModulePath) &&
				!strings.HasPrefix(p, "studio-artifact/") {
				v.fail("imports", artifact.EDeps, path, "unapproved import "+p)
				return
			}
		}
	}
	v.pass("imports")
}

// sortedKeys orders member iteration so evidence is deterministic.
func sortedKeys(members map[string][]byte) []string {
	keys := make([]string, 0, len(members))
	for k := range members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (v *verifier) checkSecrets(members map[string][]byte) {
	for _, path := range sortedKeys(members) {
		data := members[path]
		if strings.HasSuffix(path, ".json") {
			if errs := spec.ScanSecretsJSON(data); len(errs) != 0 {
				v.fail("secrets", artifact.ESecret, path, errs[0].Message)
				return
			}
		}
		if ContainsSecretSpan(string(data)) {
			v.fail("secrets", artifact.ESecret, path, "credential-shaped content")
			return
		}
	}
	v.pass("secrets")
}

func (v *verifier) checkBinding(env *artifact.Envelope, members map[string][]byte) {
	var doc spec.MiniappSpec
	if err := json.Unmarshal(members[artifact.SpecFileName], &doc); err != nil {
		v.fail("binding", artifact.EBinding, artifact.SpecFileName, "spec does not decode")
		return
	}
	if env.AppID != doc.App.ID {
		v.fail("binding", artifact.EBinding, "app.id", "envelope app does not match the spec")
		return
	}
	var mf contract.MiniappManifest
	if err := json.Unmarshal(members[artifact.ManifestFileName], &mf); err != nil {
		v.fail("binding", artifact.EBinding, artifact.ManifestFileName, "manifest does not decode")
		return
	}
	if string(mf.ID) != doc.App.ID {
		v.fail("binding", artifact.EBinding, "app.id", "manifest id does not match the spec")
		return
	}
	if string(mf.Version) != doc.App.AppVersion {
		v.fail("binding", artifact.EBinding, "app.appVersion", "manifest version does not match the spec")
		return
	}
	declared := map[string]bool{}
	for _, tool := range mf.Tools {
		declared[string(tool.Name)] = true
	}
	for _, op := range doc.Capabilities.Operations {
		if !declared[op.ID] {
			v.fail("binding", artifact.EBinding, "capabilities", "spec operation "+op.ID+" has no declared tool")
			return
		}
	}
	v.pass("binding")
}

func (v *verifier) finish(ctx context.Context, tx pgx.Tx) (VerifyOutcome, error) {
	passed := !v.failed
	checksRaw, err := json.Marshal(v.checks)
	if err != nil {
		return VerifyOutcome{}, err
	}
	evidenceID, err := randomID()
	if err != nil {
		return VerifyOutcome{}, err
	}
	tag, err := tx.Exec(ctx,
		`INSERT INTO studio.artifact_verifications (id, tenant_id, artifact_id, check_set_version, passed, checks)
		 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (tenant_id, artifact_id, check_set_version) DO NOTHING`,
		evidenceID, v.tenant, v.artifactID, CheckSetVersion, passed, json.RawMessage(checksRaw))
	if err != nil {
		return VerifyOutcome{}, dbErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		// Re-verified identical bytes: return the original
		// evidence id instead of the discarded fresh one.
		if err := tx.QueryRow(ctx,
			`SELECT id FROM studio.artifact_verifications
			 WHERE tenant_id = $1 AND artifact_id = $2 AND check_set_version = $3`,
			v.tenant, v.artifactID, CheckSetVersion).Scan(&evidenceID); err != nil {
			return VerifyOutcome{}, dbErr(err, "")
		}
	}
	if passed {
		if _, err := tx.Exec(ctx,
			`UPDATE studio.artifacts SET verified_at = now() WHERE tenant_id = $1 AND id = $2 AND verified_at IS NULL`,
			v.tenant, v.artifactID); err != nil {
			return VerifyOutcome{}, dbErr(err, "")
		}
	}
	if err := v.appendStreamRow(ctx, tx); err != nil {
		return VerifyOutcome{}, err
	}
	return VerifyOutcome{Passed: passed, EvidenceID: evidenceID, Checks: checksRaw}, nil
}

// appendStreamRow announces the outcome on the conversation Pulse.
// Pass and fail both stream (a failure-only signal would mislead);
// the shaped payload carries the polarity.
func (v *verifier) appendStreamRow(ctx context.Context, tx pgx.Tx) error {
	var conv, build string
	if err := tx.QueryRow(ctx,
		`SELECT b.conversation_id, b.id FROM studio.artifacts AS a
		 JOIN studio.builds AS b ON b.tenant_id = a.tenant_id AND b.id = a.build_id
		 WHERE a.tenant_id = $1 AND a.id = $2`,
		v.tenant, v.artifactID).Scan(&conv, &build); err != nil {
		return dbErr(err, "")
	}
	_, err := appendStream(ctx, tx, v.tenant, conv, "artifact_verified", v.artifactID, build)
	return err
}
