-- Link each transparency record back to the extraction row it was built from.
-- The pipeline callback now writes both in one transaction: the raw extraction
-- (struk_extractions et al.) and its public-facing evidence card
-- (transparency_*). The unique FK makes that write idempotent — a re-run of the
-- same batch job updates the existing card instead of stacking duplicates — and
-- lets the frontend drill from a map feature straight to the source photo
-- (methodology 3.5: "telusuri sampai foto aslinya").

ALTER TABLE transparency_struk
    ADD COLUMN struk_extraction_id UUID UNIQUE REFERENCES struk_extractions(id) ON DELETE CASCADE;

ALTER TABLE transparency_gerai
    ADD COLUMN gerai_classification_id UUID UNIQUE REFERENCES gerai_classifications(id) ON DELETE CASCADE;

ALTER TABLE transparency_properti
    ADD COLUMN properti_extraction_id UUID UNIQUE REFERENCES properti_extractions(id) ON DELETE CASCADE;
