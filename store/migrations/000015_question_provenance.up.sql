-- 000015: question provenance and options (Phase A follow-up).
-- Options arrive inside the gateway event payload; denormalizing them onto
-- the question row keeps answer rendering a single-row read, while
-- source_event_id preserves the link to the full ingested payload.

ALTER TABLE studio.questions
  ADD COLUMN source_event_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN options JSONB NOT NULL DEFAULT '[]';
