-- Isi Stasiun — demo/analytics seed (real compiled data: Manggarai & Sudirman)
-- Populates the read-only analytics tables the pipeline would normally fill,
-- so the API serves real numbers before the batch pipeline is wired up.
--
-- Values are identical to the AI service fixtures (isistasiun-ai/fixtures/*),
-- so AI grounding, BE responses, and the FE map all agree. Station UUIDs are
-- fixed (…0001 Manggarai, …0002 Sudirman) to match those fixtures.
--
-- Idempotent: safe to re-run (ON CONFLICT DO UPDATE).
-- Run:  psql "$DATABASE_URL" -f backend/seed/isi_stasiun_seed.sql
--
-- NOT seeded (deliberate):
--   * event_potential_scores — no real data exists (no field survey); left empty.
--   * Manggarai station_entrances — no real entrance coordinates were captured
--     (only gerai/ruko points). Sudirman's two real entrances are seeded.
--   * rent_flow_index has no "status" column in the schema; the vacant plots
--     are the plot_id 'manggarai-petak-05..08' rows (offered_rent 594,000,000).

BEGIN;

-- ---- stations ---------------------------------------------------------------
INSERT INTO stations (id, name, code, operator, area_type, location) VALUES
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'Manggarai', 'MRI', 'KAI Commuter', 'mixed',
   ST_SetSRID(ST_MakePoint(106.8502, -6.2102), 4326)::geography),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'Sudirman', 'SDM', 'KAI Commuter', 'office',
   ST_SetSRID(ST_MakePoint(106.8235, -6.2024), 4326)::geography)
ON CONFLICT (id) DO UPDATE
  SET name = EXCLUDED.name, code = EXCLUDED.code, operator = EXCLUDED.operator,
      area_type = EXCLUDED.area_type, location = EXCLUDED.location, updated_at = now();

-- ---- station_entrances (Sudirman only — real coords) ------------------------
INSERT INTO station_entrances (station_id, label, location) VALUES
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'Pintu Atas',
   ST_SetSRID(ST_MakePoint(106.8233293, -6.2022544), 4326)::geography),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'Pintu C1',
   ST_SetSRID(ST_MakePoint(106.8246516, -6.2027007), 4326)::geography)
ON CONFLICT (station_id, label) DO UPDATE SET location = EXCLUDED.location;

-- ---- spending_gap_estimates -------------------------------------------------
-- Sudirman morning is intentionally absent (no Lewat denominator → not estimated).
INSERT INTO spending_gap_estimates
  (job_id, station_id, time_slot, potential_low_p10, potential_high_p90,
   captured_low_p10, captured_high_p90, gap_low_p10, gap_high_p90) VALUES
  ('seed-manual', 'a10a6cf2-0001-4f2b-9c1a-000000000001', 'morning',
   900000, 1600000, 500000, 900000, 400000, 700000),
  ('seed-manual', 'a10a6cf2-0001-4f2b-9c1a-000000000001', 'evening',
   1500000, 2600000, 700000, 1200000, 800000, 1400000),
  ('seed-manual', 'a10a6cf2-0002-4f2b-9c1a-000000000002', 'evening',
   1100000, 2000000, 200000, 400000, 900000, 1600000)
ON CONFLICT (station_id, time_slot) DO UPDATE
  SET potential_low_p10 = EXCLUDED.potential_low_p10,
      potential_high_p90 = EXCLUDED.potential_high_p90,
      captured_low_p10 = EXCLUDED.captured_low_p10,
      captured_high_p90 = EXCLUDED.captured_high_p90,
      gap_low_p10 = EXCLUDED.gap_low_p10,
      gap_high_p90 = EXCLUDED.gap_high_p90,
      computed_at = now();

-- ---- category_gap_estimates -------------------------------------------------
-- demand_in_area is evidence-backed: real POI counts within 800 m of each
-- station (MAPID Data Premium POI, ~13k scanned). Counts per (station,category):
--   Manggarai: makanan_minuman 21, ritel_kemasan 4,  apotek_kesehatan 13, jasa 55
--   Sudirman:  makanan_minuman 30, ritel_kemasan 20, apotek_kesehatan 4,  jasa 71
-- (No demand_count column in this schema yet — add a migration if you want to
--  expose the number; the values live in isistasiun-ai fixtures/provenance.)
-- available_in_station is from the field survey (gerai inside the station), so
-- apotek & jasa are demanded-but-absent (the "missing" categories) at both.
INSERT INTO category_gap_estimates (station_id, category, demand_in_area, available_in_station) VALUES
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'makanan_minuman',  true,  true),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'ritel_kemasan',    true,  true),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'apotek_kesehatan', true,  false),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'jasa',             true,  false),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'lainnya',          false, false),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'makanan_minuman',  true,  true),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'ritel_kemasan',    true,  true),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'apotek_kesehatan', true,  false),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'jasa',             true,  false),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'lainnya',          false, false)
ON CONFLICT (station_id, category) DO UPDATE
  SET demand_in_area = EXCLUDED.demand_in_area,
      available_in_station = EXCLUDED.available_in_station;

-- ---- confidence_layer -------------------------------------------------------
INSERT INTO confidence_layer (station_id, zone_id, sample_count, is_thin_sample, confidence_score) VALUES
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'manggarai-morning', 4, true, 0.45),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'manggarai-evening', 4, true, 0.50),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'sudirman-morning',  2, true, 0.15),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'sudirman-evening',  4, true, 0.42)
ON CONFLICT (station_id, zone_id) DO UPDATE
  SET sample_count = EXCLUDED.sample_count,
      is_thin_sample = EXCLUDED.is_thin_sample,
      confidence_score = EXCLUDED.confidence_score;

-- ---- rent_flow_index --------------------------------------------------------
-- offered_rent = Rp/year (Space by KAI). measured_flow = station gate flow proxy.
-- index_value = offered_rent / measured_flow. petak-05..08 = vacant plots.
INSERT INTO rent_flow_index (plot_id, station_id, offered_rent, measured_flow, index_value, is_outlier) VALUES
  ('manggarai-cfc',             'a10a6cf2-0001-4f2b-9c1a-000000000001', 1577000000, 15000, 105133.33, false),
  ('manggarai-familymart',      'a10a6cf2-0001-4f2b-9c1a-000000000001',  250000000, 15000,  16666.67, false),
  ('manggarai-indomaret-timur', 'a10a6cf2-0001-4f2b-9c1a-000000000001',  200000000, 15000,  13333.33, false),
  ('manggarai-toto-express',    'a10a6cf2-0001-4f2b-9c1a-000000000001',  150000000, 15000,  10000.00, false),
  ('manggarai-petak-05',        'a10a6cf2-0001-4f2b-9c1a-000000000001',  594000000, 15000,  39600.00, false),
  ('manggarai-petak-06',        'a10a6cf2-0001-4f2b-9c1a-000000000001',  594000000, 15000,  39600.00, false),
  ('manggarai-petak-07',        'a10a6cf2-0001-4f2b-9c1a-000000000001',  594000000, 15000,  39600.00, false),
  ('manggarai-petak-08',        'a10a6cf2-0001-4f2b-9c1a-000000000001',  594000000, 15000,  39600.00, false)
ON CONFLICT (station_id, plot_id) DO UPDATE
  SET offered_rent = EXCLUDED.offered_rent,
      measured_flow = EXCLUDED.measured_flow,
      index_value = EXCLUDED.index_value,
      is_outlier = EXCLUDED.is_outlier;

COMMIT;
