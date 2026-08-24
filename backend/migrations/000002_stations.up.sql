CREATE TABLE stations (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          TEXT NOT NULL,
    code          TEXT NOT NULL UNIQUE,
    operator      TEXT NOT NULL,
    area_type     TEXT NOT NULL CHECK (area_type IN ('residential','office','mixed')),
    location      GEOGRAPHY(POINT, 4326) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_stations_location ON stations USING GIST (location);

CREATE TABLE station_entrances (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id    UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    label         TEXT NOT NULL, -- e.g. "T1"
    location      GEOGRAPHY(POINT, 4326) NOT NULL,
    UNIQUE (station_id, label)
);
CREATE INDEX idx_entrances_location ON station_entrances USING GIST (location);
CREATE INDEX idx_entrances_station ON station_entrances (station_id);
