-- Down reverses 000001. The application role is operator-owned and is
-- never dropped here.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'studio_lookup') THEN
    DROP ROLE studio_lookup;
  END IF;
END $$;
