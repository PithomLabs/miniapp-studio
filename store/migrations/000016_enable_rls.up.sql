-- 000016: enable RLS on the Phase A tables (bounded closure item 3).
-- Migrations 000011-000014 set FORCE ROW LEVEL SECURITY but never ENABLEd
-- it, leaving the tenant policies inert: FORCE without ENABLE enforces
-- nothing. Tenant isolation held at the predicate layer (every query
-- carries explicit tenant_id), but the mandated database layer was
-- absent. This migration enables enforcement on all ten tables; FORCE is
-- retained so table owners stay inside the policy boundary too.

ALTER TABLE studio.conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.builds ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.turns ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.questions ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.commands ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.events ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.conversation_stream ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.artifacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.approvals ENABLE ROW LEVEL SECURITY;
