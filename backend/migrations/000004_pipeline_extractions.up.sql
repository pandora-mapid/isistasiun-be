-- Raw AI extraction outputs (OCR struk/properti, gerai classification).
-- job_id + source_ref composite makes callback ingestion idempotent.

CREATE TABLE struk_extractions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id          TEXT NOT NULL,
    source_ref      TEXT NOT NULL, -- R2 object key
    station_id      UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    category        TEXT NOT NULL CHECK (category IN ('makanan_minuman','ritel_kemasan','apotek_kesehatan','jasa','lainnya')),
    final_amount    NUMERIC(14,2) NOT NULL CHECK (final_amount >= 0),
    payment_method  TEXT,
    transacted_at   TIMESTAMPTZ NOT NULL,
    confidence      NUMERIC(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    is_ambiguous    BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_id, source_ref)
);
CREATE INDEX idx_struk_station ON struk_extractions (station_id);

CREATE TABLE properti_extractions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id          TEXT NOT NULL,
    source_ref      TEXT NOT NULL,
    station_id      UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    plot_id         TEXT NOT NULL,
    offered_rent    NUMERIC(14,2) NOT NULL CHECK (offered_rent >= 0),
    area_sqm        NUMERIC(10,2) NOT NULL CHECK (area_sqm >= 0),
    confidence      NUMERIC(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_id, source_ref)
);
CREATE INDEX idx_properti_station ON properti_extractions (station_id);

CREATE TABLE gerai_classifications (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id          TEXT NOT NULL,
    source_ref      TEXT NOT NULL,
    station_id      UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    gerai_id        UUID NOT NULL,
    category        TEXT NOT NULL CHECK (category IN ('makanan_minuman','ritel_kemasan','apotek_kesehatan','jasa','lainnya')),
    visibility      TEXT CHECK (visibility IN ('high','medium','low')),
    confidence      NUMERIC(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_id, source_ref)
);
CREATE INDEX idx_gerai_class_station ON gerai_classifications (station_id);
