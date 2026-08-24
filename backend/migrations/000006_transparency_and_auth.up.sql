CREATE TABLE transparency_struk (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id   UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    photo_url    TEXT NOT NULL, -- Cloudflare R2 public URL
    ai_reading   TEXT NOT NULL, -- JSON-as-text snapshot of the OCR output
    confidence   NUMERIC(4,3) NOT NULL DEFAULT 0,
    is_ambiguous BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE transparency_gerai (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id   UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    photo_url    TEXT NOT NULL,
    ai_reading   TEXT NOT NULL,
    confidence   NUMERIC(4,3) NOT NULL DEFAULT 0
);

CREATE TABLE transparency_properti (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    station_id   UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    photo_url    TEXT NOT NULL,
    ai_reading   TEXT NOT NULL,
    confidence   NUMERIC(4,3) NOT NULL DEFAULT 0
);

CREATE TABLE operators (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    role           TEXT NOT NULL CHECK (role IN ('operator','admin')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
