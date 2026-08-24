CREATE TABLE flow_observations (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id        UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    entrance_id       UUID NOT NULL REFERENCES station_entrances(id) ON DELETE CASCADE,
    time_slot         TEXT NOT NULL CHECK (time_slot IN ('morning','midday','evening','night')),
    observed_at       TIMESTAMPTZ NOT NULL,
    block_number      SMALLINT NOT NULL CHECK (block_number IN (1,2)),
    pedestrian_count  INTEGER NOT NULL CHECK (pedestrian_count >= 0),
    direction         TEXT NOT NULL CHECK (direction IN ('in','out')),
    weather_note      TEXT,
    surveyor_id       TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_flow_obs_station_slot ON flow_observations (station_id, time_slot);
CREATE INDEX idx_flow_obs_entrance ON flow_observations (entrance_id);

CREATE TABLE entry_conversion_observations (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id                UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    gerai_id                  UUID NOT NULL,
    category                  TEXT NOT NULL CHECK (category IN ('makanan_minuman','ritel_kemasan','apotek_kesehatan','jasa','lainnya')),
    time_slot                 TEXT NOT NULL CHECK (time_slot IN ('morning','midday','evening','night')),
    observed_at               TIMESTAMPTZ NOT NULL,
    block_number              SMALLINT NOT NULL CHECK (block_number IN (1,2)),
    passers_by                INTEGER NOT NULL CHECK (passers_by >= 0),
    entered_count             INTEGER NOT NULL CHECK (entered_count >= 0),
    completed_purchase_count  INTEGER NOT NULL CHECK (completed_purchase_count >= 0),
    surveyor_id                TEXT NOT NULL,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (entered_count <= passers_by),
    CHECK (completed_purchase_count <= entered_count)
);
CREATE INDEX idx_entry_conv_station_slot ON entry_conversion_observations (station_id, time_slot);
CREATE INDEX idx_entry_conv_gerai ON entry_conversion_observations (gerai_id);
