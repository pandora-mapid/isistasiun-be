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
INSERT INTO category_gap_estimates (station_id, category, demand_in_area, available_in_station, demand_count) VALUES
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'makanan_minuman',  true,  true,  21),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'ritel_kemasan',    true,  true,  4),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'apotek_kesehatan', true,  false, 13),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'jasa',             true,  false, 55),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001', 'lainnya',          false, false, 0),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'makanan_minuman',  true,  true,  30),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'ritel_kemasan',    true,  true,  20),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'apotek_kesehatan', true,  false, 4),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'jasa',             true,  false, 71),
  ('a10a6cf2-0002-4f2b-9c1a-000000000002', 'lainnya',          false, false, 0)
ON CONFLICT (station_id, category) DO UPDATE
  SET demand_in_area = EXCLUDED.demand_in_area,
      available_in_station = EXCLUDED.available_in_station,
      demand_count = EXCLUDED.demand_count;

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

-- ---- rental_assets (Space by KAI — Manggarai in-station, real) ---------------
-- Tenant contracts: commercial_value_visible=false (KAI marks them not for
-- public display). Vacant plots No 5-8: public leasing price (visible=true).
-- Sudirman has no in-station KAI assets (verified by-coordinate) — none seeded.
INSERT INTO rental_assets
  (station_id, source_id, data_source, location_name, plot_name, area_name,
   latitude, longitude, building_area, rented, availability_status,
   commercial_value, commercial_value_visible, source_updated_at) VALUES
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818818957567','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN MANGGARAI 50,09M (OUTLET CFC)','Daop 1 Jakarta',-6.2099312,106.8502312,50.09,true,'occupied',1577342000,false,'2026-08-21 06:36:04+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818819205543','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN BESAR MANGGARAI 34,20M (FAMILY MART)','Daop 1 Jakarta',-6.2098672,106.8500059,34.20,true,'occupied',250000000,false,'2026-08-21 04:58:14+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818818422713','space_kai','Stasiun Manggarai','SPACE STASIUN MANGGARAI 36M (OUTLET INDOMARET PINTU TIMUR)','Daop 1 Jakarta',-6.2097743,106.8501588,36.00,true,'occupied',200000000,false,'2026-08-21 04:43:34+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818833728806','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN MANGGARAI 84,30M (TOTO EXPRESS)','Daop 1 Jakarta',-6.2092883,106.8497563,84.30,true,'occupied',150000000,false,'2026-08-21 04:23:01+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818843382536','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN MANGGARAI NO 5 49,5M','Daop 1 Jakarta',-6.2093009,106.8498053,49.50,false,'available',594000000,true,'2026-04-10 02:56:16+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818843418498','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN MANGGARAI NO 6 49,5M','Daop 1 Jakarta',-6.2093311,106.8498212,49.50,false,'available',594000000,true,'2026-04-10 02:55:44+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818843418475','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN MANGGARAI NO 7 49,5M','Daop 1 Jakarta',-6.2093612,106.8498149,49.50,false,'available',594000000,true,'2026-04-10 02:54:50+07'),
  ('a10a6cf2-0001-4f2b-9c1a-000000000001','75153818843418165','space_kai','Stasiun Manggarai','SPACE EMPLASEMEN STASIUN MANGGARAI NO 8 49,5M','Daop 1 Jakarta',-6.2093913,106.8498420,49.50,false,'available',594000000,true,'2026-04-10 02:28:02+07')
ON CONFLICT (data_source, source_id) DO UPDATE
  SET commercial_value = EXCLUDED.commercial_value, rented = EXCLUDED.rented,
      availability_status = EXCLUDED.availability_status,
      commercial_value_visible = EXCLUDED.commercial_value_visible,
      latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
      building_area = EXCLUDED.building_area, source_updated_at = EXCLUDED.source_updated_at,
      updated_at = now();

COMMIT;
