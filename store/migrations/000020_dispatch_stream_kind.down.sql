-- Restore the pre-000020 conversation_stream shape exactly.
-- Rows of kind 'dispatch' cannot exist below this version; delete
-- them first so the narrowed CHECK restores cleanly.
DELETE FROM studio.conversation_stream WHERE kind = 'dispatch';
ALTER TABLE studio.conversation_stream DROP CONSTRAINT conversation_stream_kind_check;
ALTER TABLE studio.conversation_stream ADD CHECK (kind IN ('message', 'event'));
