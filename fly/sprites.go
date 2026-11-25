package fly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	sprites "github.com/superfly/sprites-go"
)

// NotFoundError reports a missing remote resource, distinguished from
// transient failures so callers reconcile accurately instead of retrying
// blindly.
type NotFoundError struct {
	Resource string
}

// Error names the missing resource without echoing bodies.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("fly: %s not found", e.Resource)
}

// asNotFound translates the SDK's untyped not-found errors (plain
// "…not found…" strings, e.g. "sprite not found: <name>") into
// *NotFoundError. The match is coupled to sprites-go v0.2.2 message text;
// re-verify it on any SDK upgrade. Everything else passes through untouched
// so genuine failures are never misclassified as absent resources.
func asNotFound(resource string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "not found") {
		return &NotFoundError{Resource: resource}
	}
	return err
}

// SpriteAdapter implements SpriteManager over the official sprites-go SDK.
// It performs no credential handling beyond passing the constructor token to
// the SDK client, and it never logs request or response bodies.
type SpriteAdapter struct {
	client *sprites.Client
}

// NewSpriteAdapter returns an adapter over a client authenticated with token.
// An empty baseURL selects the default Sprites API endpoint. The token must
// come from server-side secret state; it is never written anywhere by this
// package. Handle-based operations (exec, filesystem, checkpoints) perform
// the SDK's bounded control-plane check; pure management calls do not.
func NewSpriteAdapter(token, baseURL string) *SpriteAdapter {
	if baseURL == "" {
		return &SpriteAdapter{client: sprites.New(token)}
	}
	return &SpriteAdapter{client: sprites.New(token, sprites.WithBaseURL(baseURL))}
}

// CreateSprite creates one sprite with labels and an optional region.
func (a *SpriteAdapter) CreateSprite(ctx context.Context, name string, opts SpriteOptions) (Sprite, error) {
	var cfg *sprites.SpriteConfig
	if opts.Region != "" {
		cfg = &sprites.SpriteConfig{Region: opts.Region}
	}
	s, err := a.client.CreateSpriteWithOrg(ctx, name, cfg, nil, opts.Labels)
	if err != nil {
		return Sprite{}, err
	}
	return a.GetSprite(ctx, s.Name())
}

// GetSprite returns the authoritative sprite record, including the
// Fly-issued URL. Callers persist URL verbatim; it is never constructed.
func (a *SpriteAdapter) GetSprite(ctx context.Context, name string) (Sprite, error) {
	s, err := a.client.GetSprite(ctx, name)
	if err != nil {
		return Sprite{}, asNotFound("sprite "+name, err)
	}
	return toOwnedSprite(s), nil
}

func toOwnedSprite(s *sprites.Sprite) Sprite {
	out := Sprite{
		ID: s.ID, Name: s.Name(), Status: s.Status,
		URL: s.URL, Labels: append([]string(nil), s.Labels...),
		CreatedAt: s.CreatedAt,
	}
	if s.URLSettings != nil {
		out.Auth = s.URLSettings.Auth
		out.PrivateAccess = s.URLSettings.PrivateAccess
	}
	return out
}

// DeleteSprite deletes one sprite.
func (a *SpriteAdapter) DeleteSprite(ctx context.Context, name string) error {
	return asNotFound("sprite "+name, a.client.DeleteSprite(ctx, name))
}

// UpdateURLSettings replaces the sprite's URL authentication settings.
func (a *SpriteAdapter) UpdateURLSettings(ctx context.Context, name string, settings URLSettings) error {
	return a.client.UpdateURLSettings(ctx, name, &sprites.URLSettings{
		Auth: settings.Auth, PrivateAccess: settings.PrivateAccess,
	})
}

// CreateService creates one service and drains its start stream to a terminal
// outcome: success only when the stream completes without an error event.
func (a *SpriteAdapter) CreateService(ctx context.Context, sprite, service string, req ServiceRequest) error {
	stream, err := a.client.CreateService(ctx, sprite, service, &sprites.ServiceRequest{
		Cmd: req.Cmd, Args: req.Args, HTTPPort: req.HTTPPort,
	})
	if err != nil {
		return err
	}
	defer stream.Close()
	return stream.ProcessAll(func(event *sprites.ServiceLogEvent) error {
		if event.Type == "error" {
			return fmt.Errorf("fly: service %q failed: %s", service, event.Data)
		}
		return nil
	})
}

// GetService returns one service definition with runtime state.
func (a *SpriteAdapter) GetService(ctx context.Context, sprite, service string) (ServiceState, error) {
	svc, err := a.client.GetService(ctx, sprite, service)
	if err != nil {
		return ServiceState{}, asNotFound("service "+service, err)
	}
	out := ServiceState{Name: svc.Name}
	if svc.State != nil {
		out.Status = svc.State.Status
		out.PID = svc.State.PID
		out.Error = svc.State.Error
	}
	return out, nil
}

// DeleteService deletes one service.
func (a *SpriteAdapter) DeleteService(ctx context.Context, sprite, service string) error {
	return asNotFound("service "+service, a.client.DeleteService(ctx, sprite, service))
}

// Exec runs one command and captures its output. An empty command is refused
// without a network call. Exit codes come from the SDK's typed error.
func (a *SpriteAdapter) Exec(ctx context.Context, sprite string, cmd []string) (ExecResult, error) {
	if len(cmd) == 0 {
		return ExecResult{}, fmt.Errorf("fly: empty command")
	}
	s := a.client.Sprite(sprite)
	c := s.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		var exitErr *sprites.ExitError
		if errors.As(err, &exitErr) {
			return ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitErr.ExitCode()}, nil
		}
		return ExecResult{}, err
	}
	return ExecResult{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

// WriteFile writes data to one absolute path, creating parent directories.
func (a *SpriteAdapter) WriteFile(ctx context.Context, sprite, path string, data []byte) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("fly: path must be absolute")
	}
	_ = ctx
	s := a.client.Sprite(sprite)
	fs := s.Filesystem()
	if dir := parentDir(path); dir != "" {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return fs.WriteFile(path, data, 0o644)
}

func parentDir(path string) string {
	for i := len(path) - 1; i > 0; i-- {
		if path[i] == '/' {
			if i == 1 {
				return ""
			}
			return path[:i]
		}
	}
	return ""
}

// CreateCheckpoint checkpoints one sprite and returns the new checkpoint ID,
// identified as the newest checkpoint after the creation stream completes.
// Callers serialize checkpoint flows per sprite; concurrent creators could
// otherwise misattribute the newest row.
func (a *SpriteAdapter) CreateCheckpoint(ctx context.Context, sprite, comment string) (string, error) {
	s := a.client.Sprite(sprite)
	var create func(context.Context) (*sprites.CheckpointStream, error)
	if comment == "" {
		create = s.CreateCheckpoint
	} else {
		create = func(ctx context.Context) (*sprites.CheckpointStream, error) {
			return s.CreateCheckpointWithComment(ctx, comment)
		}
	}
	stream, err := create(ctx)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	if err := stream.ProcessAll(func(msg *sprites.StreamMessage) error {
		if msg.Type == "error" {
			return fmt.Errorf("fly: checkpoint failed: %s", msg.Error)
		}
		return nil
	}); err != nil {
		return "", err
	}
	list, err := a.client.ListCheckpoints(ctx, sprite, "")
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("fly: no checkpoints found after creation")
	}
	newest := list[0]
	for _, c := range list[1:] {
		if c.CreateTime.After(newest.CreateTime) {
			newest = c
		}
	}
	return newest.ID, nil
}

// RestoreCheckpoint restores one checkpoint, draining to a terminal outcome.
func (a *SpriteAdapter) RestoreCheckpoint(ctx context.Context, sprite, checkpointID string) error {
	stream, err := a.client.RestoreCheckpoint(ctx, sprite, checkpointID)
	if err != nil {
		return err
	}
	defer stream.Close()
	return stream.ProcessAll(func(msg *sprites.StreamMessage) error {
		if msg.Type == "error" {
			return fmt.Errorf("fly: restore failed: %s", msg.Error)
		}
		return nil
	})
}

// GetNetworkPolicy returns the sprite egress policy.
func (a *SpriteAdapter) GetNetworkPolicy(ctx context.Context, sprite string) (NetworkPolicy, error) {
	pol, err := a.client.GetNetworkPolicy(ctx, sprite)
	if err != nil {
		return NetworkPolicy{}, err
	}
	return toOwnedPolicy(pol), nil
}

func toOwnedPolicy(pol *sprites.NetworkPolicy) NetworkPolicy {
	out := NetworkPolicy{}
	for _, r := range pol.Rules {
		out.Rules = append(out.Rules, NetworkPolicyRule{Domain: r.Domain, Action: r.Action})
	}
	return out
}

// UpdateNetworkPolicy replaces the sprite egress policy.
func (a *SpriteAdapter) UpdateNetworkPolicy(ctx context.Context, sprite string, policy NetworkPolicy) error {
	sdk := &sprites.NetworkPolicy{}
	for _, r := range policy.Rules {
		sdk.Rules = append(sdk.Rules, sprites.NetworkPolicyRule{Domain: r.Domain, Action: r.Action})
	}
	return a.client.UpdateNetworkPolicy(ctx, sprite, sdk)
}
