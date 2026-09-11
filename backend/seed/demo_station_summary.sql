-- Demo seed for the station-summary / Compare View endpoint.
--
-- Dev + demo only. Lets `GET /api/v1/analytics/station-summary` return the two
-- study stations (Manggarai, Sudirman) on a fresh stack so the endpoint can be
-- eyeballed with curl. Numbers mirror the frontend mock
-- (isistasiun-fe/public/mock/station-summary.json) exactly.
--
-- NOT a migration on purpose: it inserts data, and the stations rows overlap
-- with whatever the stations owner seeds for real. Guarded with fixed UUIDs +
-- ON CONFLICT so it is idempotent and yields to a real stations seed on `code`.
--
--   make be-seed        # apply
--   make be-seed-down    # remove just these demo rows
--
-- Real data path: Python pipeline -> POST /pipeline/simulations/monte-carlo
-- (station-scoped) -> station_summary. This file stands in until that lands.

BEGIN;

-- ---- stations (yield to a real seed sharing the same code) ------------------
INSERT INTO stations (id, name, code, operator, area_type, location) VALUES
  ('a0000000-0000-4000-8000-000000000001', 'Manggarai', 'MRI', 'KAI Commuter', 'mixed',
   ST_SetSRID(ST_MakePoint(106.84993, -6.21490), 4326)::geography),
  ('a0000000-0000-4000-8000-000000000002', 'Sudirman',  'SUD', 'KAI Commuter', 'office',
   ST_SetSRID(ST_MakePoint(106.82300, -6.20130), 4326)::geography)
ON CONFLICT (code) DO NOTHING;

-- Resolve ids by code so the rest works whether the rows above won or a real
-- seed already held them.
INSERT INTO station_entrances (station_id, label, location)
SELECT s.id, v.label, ST_SetSRID(ST_MakePoint(v.lon, v.lat), 4326)::geography
FROM (VALUES
  ('MRI', 'Pintu Utama',            106.84980, -6.21470),
  ('MRI', 'Koridor Transit Utara',  106.85010, -6.21440),
  ('MRI', 'Koridor Transit Selatan',106.85000, -6.21540),
  ('SUD', 'Pintu 1',                106.82280, -6.20100),
  ('SUD', 'Pintu 2',                106.82320, -6.20120),
  ('SUD', 'Pintu 3',                106.82290, -6.20160),
  ('SUD', 'Pintu 4',                106.82330, -6.20150)
) AS v(code, label, lon, lat)
JOIN stations s ON s.code = v.code
ON CONFLICT (station_id, label) DO NOTHING;

-- ---- station_summary (weekday) --------------------------------------------
INSERT INTO station_summary (
  id, job_id, station_id, day_type,
  potential_p10, potential_p50, potential_p90,
  captured_p10, captured_p50, captured_p90,
  gap_p10, gap_p50, gap_p90,
  capture_rate, confidence_min, confidence_max, struk_terbaca,
  pintu_dicacah, pintu_ditahan,
  peak_point_label, peak_time_slot, peak_f, peak_e, peak_c, peak_v,
  peak_gap_p10, peak_gap_p50, peak_gap_p90, basis
)
SELECT * FROM (VALUES
  ('b0000000-0000-4000-8000-000000000001'::uuid, 'demo-seed',
   (SELECT id FROM stations WHERE code = 'MRI'), 'weekday',
   7200000, 8200000, 10500000,
   2200000, 2500000, 3200000,
   5000000, 5700000, 7300000,
   0.3049, 0.770, 0.870, 713,
   3, 0,
   'Koridor Transit Utara', 'morning', 1574, 0.0700, 0.6600, 30000,
   860000, 1100000, 1520000, 'monte-carlo-simpul'),
  ('b0000000-0000-4000-8000-000000000002'::uuid, 'demo-seed',
   (SELECT id FROM stations WHERE code = 'SUD'), 'weekday',
   4400000, 5000000, 6400000,
   1300000, 1500000, 1900000,
   3100000, 3500000, 4500000,
   0.3000, 0.770, 0.920, 848,
   4, 1,
   'Pintu 4', 'morning', 977, 0.0620, 0.7200, 30000,
   530000, 680000, 940000, 'monte-carlo-simpul')
) AS v
ON CONFLICT (station_id, day_type) DO UPDATE SET
  potential_p10 = EXCLUDED.potential_p10, potential_p50 = EXCLUDED.potential_p50, potential_p90 = EXCLUDED.potential_p90,
  captured_p10  = EXCLUDED.captured_p10,  captured_p50  = EXCLUDED.captured_p50,  captured_p90  = EXCLUDED.captured_p90,
  gap_p10       = EXCLUDED.gap_p10,       gap_p50       = EXCLUDED.gap_p50,       gap_p90       = EXCLUDED.gap_p90,
  capture_rate  = EXCLUDED.capture_rate,
  confidence_min = EXCLUDED.confidence_min, confidence_max = EXCLUDED.confidence_max,
  struk_terbaca = EXCLUDED.struk_terbaca,
  pintu_dicacah = EXCLUDED.pintu_dicacah, pintu_ditahan = EXCLUDED.pintu_ditahan,
  peak_point_label = EXCLUDED.peak_point_label, peak_time_slot = EXCLUDED.peak_time_slot,
  peak_f = EXCLUDED.peak_f, peak_e = EXCLUDED.peak_e, peak_c = EXCLUDED.peak_c, peak_v = EXCLUDED.peak_v,
  peak_gap_p10 = EXCLUDED.peak_gap_p10, peak_gap_p50 = EXCLUDED.peak_gap_p50, peak_gap_p90 = EXCLUDED.peak_gap_p90,
  basis = EXCLUDED.basis, computed_at = now();

-- ---- station_summary_category --------------------------------------------
DELETE FROM station_summary_category
WHERE station_summary_id IN (
  'b0000000-0000-4000-8000-000000000001', 'b0000000-0000-4000-8000-000000000002'
);

INSERT INTO station_summary_category (station_summary_id, category, demand_share, gerai_count, is_missing) VALUES
  ('b0000000-0000-4000-8000-000000000001', 'makanan_minuman',  0.4666, 3, false),
  ('b0000000-0000-4000-8000-000000000001', 'ritel_kemasan',    0.1837, 2, true),
  ('b0000000-0000-4000-8000-000000000001', 'apotek_kesehatan', 0.1061, 0, true),
  ('b0000000-0000-4000-8000-000000000001', 'jasa',             0.1257, 1, true),
  ('b0000000-0000-4000-8000-000000000001', 'lainnya',          0.1179, 2, true),
  ('b0000000-0000-4000-8000-000000000002', 'makanan_minuman',  0.5087, 3, false),
  ('b0000000-0000-4000-8000-000000000002', 'ritel_kemasan',    0.1483, 2, true),
  ('b0000000-0000-4000-8000-000000000002', 'apotek_kesehatan', 0.1221, 0, true),
  ('b0000000-0000-4000-8000-000000000002', 'jasa',             0.1240, 1, true),
  ('b0000000-0000-4000-8000-000000000002', 'lainnya',          0.0969, 2, true);

COMMIT;
