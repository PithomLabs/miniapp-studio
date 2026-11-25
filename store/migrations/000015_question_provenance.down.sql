ALTER TABLE studio.questions
  DROP COLUMN IF EXISTS options,
  DROP COLUMN IF EXISTS source_event_id;
