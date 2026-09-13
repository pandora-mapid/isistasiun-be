-- Station-level rollup for the "ringkasan & perbandingan antarsimpul" view
-- (free tier — Compare View, 2 simpul). One tier up from spending_gap_estimates:
-- that table is per (station, time_slot); this one is one row per station and
-- carries a full P10/P50/P90 range plus the axes Persona 2 compares on
-- (capture rate, entry ratio at the peak point, category composition).
--
-- Written by the Python batch pipeline via POST /pipeline/simulations/monte-carlo
-- once that callback is extended to emit a station-scoped run (see
-- ../Context/02-BACKEND-SPEC.md 3.6). Go API is read-only here.
--
-- `basis` records where the range came from: 'monte-carlo-simpul' (a real
-- station-scoped simulation) vs 'agregat-titik' (a stopgap rollup of point
-- figures). The frontend surfaces this so a summed number can't masquerade as
-- a simulated one.

CREATE TABLE station_summary (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id              TEXT NOT NULL,
    station_id          UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    day_type            TEXT NOT NULL DEFAULT 'weekday' CHECK (day_type IN ('weekday','weekend')),

    potential_p10       NUMERIC(14,2) NOT NULL,
    potential_p50       NUMERIC(14,2) NOT NULL,
    potential_p90       NUMERIC(14,2) NOT NULL,
    captured_p10        NUMERIC(14,2) NOT NULL,
    captured_p50        NUMERIC(14,2) NOT NULL,
    captured_p90        NUMERIC(14,2) NOT NULL,
    gap_p10             NUMERIC(14,2) NOT NULL,
    gap_p50             NUMERIC(14,2) NOT NULL,
    gap_p90             NUMERIC(14,2) NOT NULL,

    capture_rate        NUMERIC(6,4),                 -- captured_p50 / potential_p50
    confidence_min      NUMERIC(4,3),
    confidence_max      NUMERIC(4,3),
    struk_terbaca       INTEGER NOT NULL DEFAULT 0,
    pintu_dicacah       INTEGER NOT NULL DEFAULT 0,
    pintu_ditahan       INTEGER NOT NULL DEFAULT 0,

    -- Peak observation point — carries the station's F/E/C/V character.
    peak_point_label    TEXT,
    peak_time_slot      TEXT CHECK (peak_time_slot IN ('morning','midday','evening','night')),
    peak_f              NUMERIC(12,2),
    peak_e              NUMERIC(6,4),
    peak_c              NUMERIC(6,4),
    peak_v              NUMERIC(14,2),
    peak_gap_p10        NUMERIC(14,2),
    peak_gap_p50        NUMERIC(14,2),
    peak_gap_p90        NUMERIC(14,2),

    basis               TEXT NOT NULL DEFAULT 'monte-carlo-simpul'
                        CHECK (basis IN ('monte-carlo-simpul','agregat-titik')),
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (station_id, day_type)
);

CREATE TABLE station_summary_category (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_summary_id  UUID NOT NULL REFERENCES station_summary(id) ON DELETE CASCADE,
    category            TEXT NOT NULL CHECK (category IN ('makanan_minuman','ritel_kemasan','apotek_kesehatan','jasa','lainnya')),
    demand_share        NUMERIC(6,4) NOT NULL,
    gerai_count         INTEGER NOT NULL DEFAULT 0,
    is_missing          BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (station_summary_id, category)
);
CREATE INDEX idx_station_summary_category_summary ON station_summary_category (station_summary_id);
