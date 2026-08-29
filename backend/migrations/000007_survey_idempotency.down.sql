DROP INDEX IF EXISTS idx_entry_conv_idempotency;
DROP INDEX IF EXISTS idx_flow_obs_idempotency;
ALTER TABLE entry_conversion_observations DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE flow_observations DROP COLUMN IF EXISTS idempotency_key;
