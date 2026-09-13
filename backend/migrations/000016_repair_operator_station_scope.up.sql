-- Recovery for environments that briefly ran demand_count as migration 13.
-- Their schema_migrations table says 13 although operator_station_scope never
-- ran. Fresh databases already have this shape from migration 13, so every
-- statement below is deliberately idempotent.
ALTER TABLE operators
    ADD COLUMN IF NOT EXISTS station_id UUID REFERENCES stations(id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'operators_operator_role_needs_station'
          AND conrelid = 'operators'::regclass
    ) THEN
        ALTER TABLE operators
            ADD CONSTRAINT operators_operator_role_needs_station
            CHECK (role <> 'operator' OR station_id IS NOT NULL) NOT VALID;
    END IF;
END
$$;
