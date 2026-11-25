-- 000018: durable sender scheduling. next_attempt_at carries backoff and
-- prerequisite waits so a Studio restart resumes honestly instead of
-- either hammering the gateway or forgetting a wait.
ALTER TABLE studio.commands ADD COLUMN next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX commands_due ON studio.commands (tenant_id, state, next_attempt_at);
