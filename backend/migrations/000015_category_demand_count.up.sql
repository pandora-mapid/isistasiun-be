-- Real POI-in-catchment count per (station, category), evidence for
-- category_gap.demand_in_area. Nullable-safe default 0 so existing rows are valid.
ALTER TABLE category_gap_estimates
    ADD COLUMN IF NOT EXISTS demand_count INTEGER NOT NULL DEFAULT 0;
