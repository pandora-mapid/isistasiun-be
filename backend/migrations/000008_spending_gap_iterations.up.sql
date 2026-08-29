-- Persist the Monte Carlo iteration count behind each estimate so the
-- methodology page can show every published P10-P90 range came from the full
-- 10,000-iteration run (section 3.3). Existing rows default to 0 until the
-- next pipeline re-run overwrites them.

ALTER TABLE spending_gap_estimates ADD COLUMN iterations INTEGER NOT NULL DEFAULT 0;
