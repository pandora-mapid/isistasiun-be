ALTER TABLE operators DROP CONSTRAINT IF EXISTS operators_operator_role_needs_station;
ALTER TABLE operators DROP COLUMN IF EXISTS station_id;
