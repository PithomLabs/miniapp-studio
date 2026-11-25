-- 000019: durable expiry cause on questions. Terminal-while-waiting must
-- expire the open question with a recorded reason instead of absorbing
-- silently, so the Console can render why a later answer conflicts.
ALTER TABLE studio.questions ADD COLUMN expired_reason TEXT NOT NULL DEFAULT '';
