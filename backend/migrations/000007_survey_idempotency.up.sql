-- Optional request-level idempotency for survey ingestion. MAPID Apps runs on
-- field connections and may retry a submission it never saw acknowledged; a
-- repeated Idempotency-Key header then returns the original row instead of
-- inserting a duplicate observation. Rows submitted without the header carry a
-- NULL key and are excluded from the unique index, so they never collide.

ALTER TABLE flow_observations ADD COLUMN idempotency_key TEXT;
ALTER TABLE entry_conversion_observations ADD COLUMN idempotency_key TEXT;

CREATE UNIQUE INDEX idx_flow_obs_idempotency
    ON flow_observations (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE UNIQUE INDEX idx_entry_conv_idempotency
    ON entry_conversion_observations (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
