-- Read-only analytics tables populated by the Python batch pipeline
-- (Monte Carlo, spatial join, weighted overlay/AHP). Go API never writes
-- computed values here except via the pipeline callback module.

CREATE TABLE spending_gap_estimates (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id              TEXT NOT NULL,
    station_id          UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    time_slot           TEXT NOT NULL CHECK (time_slot IN ('morning','midday','evening','night')),
    potential_low_p10   NUMERIC(14,2) NOT NULL,
    potential_high_p90  NUMERIC(14,2) NOT NULL,
    captured_low_p10    NUMERIC(14,2) NOT NULL,
    captured_high_p90   NUMERIC(14,2) NOT NULL,
    gap_low_p10         NUMERIC(14,2) NOT NULL,
    gap_high_p90        NUMERIC(14,2) NOT NULL,
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (station_id, time_slot)
);

CREATE TABLE category_gap_estimates (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id             UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    category               TEXT NOT NULL CHECK (category IN ('makanan_minuman','ritel_kemasan','apotek_kesehatan','jasa','lainnya')),
    demand_in_area         BOOLEAN NOT NULL,
    available_in_station   BOOLEAN NOT NULL,
    UNIQUE (station_id, category)
);

CREATE TABLE rent_flow_index (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plot_id         TEXT NOT NULL,
    station_id      UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    offered_rent    NUMERIC(14,2) NOT NULL,
    measured_flow   NUMERIC(14,2) NOT NULL,
    index_value     NUMERIC(14,4) NOT NULL,
    is_outlier      BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (station_id, plot_id)
);

CREATE TABLE event_potential_scores (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id          UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    zone_id             TEXT NOT NULL,
    activation_score    NUMERIC(6,4) NOT NULL,
    recommended_slot    TEXT,
    UNIQUE (station_id, zone_id)
);

CREATE TABLE confidence_layer (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id          UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    zone_id             TEXT NOT NULL,
    sample_count        INTEGER NOT NULL DEFAULT 0,
    is_thin_sample      BOOLEAN NOT NULL DEFAULT false,
    confidence_score    NUMERIC(4,3) NOT NULL DEFAULT 0,
    UNIQUE (station_id, zone_id)
);
