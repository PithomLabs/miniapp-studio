-- 000022: artifact stream kinds. Verified, tested, fetched, and
-- decided receipts append conversation-stream rows so the Pulse
-- renders artifact outcomes from the same replayable cursor.
-- Pass and fail both stream; the shaped payload carries polarity.
ALTER TABLE studio.conversation_stream DROP CONSTRAINT conversation_stream_kind_check;
ALTER TABLE studio.conversation_stream ADD CHECK (kind IN ('message', 'event', 'dispatch',
  'artifact_fetched', 'artifact_verified', 'artifact_tested', 'artifact_decided'));
