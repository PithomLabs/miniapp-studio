package server

import (
	"context"
	"time"
)

// slowThreshold: provisioning that finishes within this window returns its
// outcome directly; slower runs continue in the background behind a
// status-polling interstitial. The interstitial reflects genuine async
// work — never an artificial delay.
const slowThreshold = 800 * time.Millisecond

// workerTimeout bounds background provisioning.
const workerTimeout = 120 * time.Second

type provisionOutcome struct {
	result map[string]any
	err    error
}

// provisionAsync runs one deploy.provision invocation, racing a timer: fast
// completions return inline, slow ones continue in the background while the
// caller renders the interstitial. The singleflight entry is always
// released; background failures log server-side and persist via the tool's
// own failure records.
func (s *Server) provisionAsync(tenant, appID, key string) <-chan provisionOutcome {
	out := make(chan provisionOutcome, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), workerTimeout)
		defer cancel()
		defer s.unmarkDeploy(tenant, appID)
		res, err := s.invokeToolTenant(ctx, tenant, "deploy.provision",
			map[string]any{"app_id": appID}, key)
		out <- provisionOutcome{result: res, err: err}
	}()
	return out
}

func (s *Server) markDeploy(tenant, appID string) bool {
	key := tenant + "\x00" + appID
	s.deployMu.Lock()
	defer s.deployMu.Unlock()
	if s.deployRunning[key] {
		return false
	}
	if s.deployRunning == nil {
		s.deployRunning = map[string]bool{}
	}
	s.deployRunning[key] = true
	return true
}

func (s *Server) unmarkDeploy(tenant, appID string) {
	key := tenant + "\x00" + appID
	s.deployMu.Lock()
	defer s.deployMu.Unlock()
	delete(s.deployRunning, key)
}
