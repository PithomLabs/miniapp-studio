-- 000020: dispatch stream kind. Acknowledged AND terminally failed
-- command settlements append conversation-stream rows, so the Pulse
-- renders dispatch outcomes from the same replayable cursor as
-- messages and gateway events. Acks are included deliberately: a
-- failure-only dispatch stream would mislead about coverage.
ALTER TABLE studio.conversation_stream DROP CONSTRAINT conversation_stream_kind_check;
ALTER TABLE studio.conversation_stream ADD CHECK (kind IN ('message', 'event', 'dispatch'));
