-- Inventory of physical/commercial rental assets shown on the public map.
-- This is deliberately separate from rent_flow_index: the latter is an
-- analytical score, while this table is the source-backed asset inventory.

CREATE TABLE rental_assets (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id               UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    source_id                TEXT NOT NULL,
    data_source              TEXT NOT NULL CHECK (data_source IN ('space_kai','field_survey')),
    location_name            TEXT NOT NULL,
    plot_name                TEXT NOT NULL,
    area_name                TEXT,
    latitude                 DOUBLE PRECISION NOT NULL,
    longitude                DOUBLE PRECISION NOT NULL,
    land_area                NUMERIC(12,2),
    building_area            NUMERIC(12,2),
    rented                   BOOLEAN NOT NULL DEFAULT false,
    availability_status      TEXT NOT NULL CHECK (availability_status IN ('occupied','available','needs_verification')),
    commercial_value         NUMERIC(14,2),
    commercial_value_visible BOOLEAN NOT NULL DEFAULT false,
    source_updated_at        TIMESTAMPTZ,
    note                     TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (data_source, source_id)
);

CREATE INDEX idx_rental_assets_station ON rental_assets (station_id);
CREATE INDEX idx_rental_assets_status ON rental_assets (station_id, availability_status);
CREATE INDEX idx_rental_assets_location ON rental_assets USING GIST (
    ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography
);
