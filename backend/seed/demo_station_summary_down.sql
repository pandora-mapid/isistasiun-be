-- Undo backend/seed/demo_station_summary.sql. Removes only the demo rows.
--
-- Resolved by station code/label, not the fixed demo UUIDs: the up script no
-- longer writes station_summary/station_summary_category under a fixed id
-- (it yields to a real pipeline row's id via ON CONFLICT), and entrances can
-- land under a real station's id too if the demo `stations` insert lost its
-- own ON CONFLICT race. Matching on code/label + job_id = 'demo-seed' keeps
-- this from ever touching real pipeline data.
BEGIN;

DELETE FROM station_summary_category
WHERE station_summary_id IN (
  SELECT ss.id FROM station_summary ss
  JOIN stations s ON s.id = ss.station_id
  WHERE s.code IN ('MRI', 'SUD') AND ss.job_id = 'demo-seed'
);

DELETE FROM station_summary
WHERE job_id = 'demo-seed'
  AND station_id IN (SELECT id FROM stations WHERE code IN ('MRI', 'SUD'));

DELETE FROM station_entrances
WHERE (station_id, label) IN (
  SELECT s.id, v.label
  FROM stations s
  JOIN (VALUES
    ('MRI', 'Pintu Utama'),
    ('MRI', 'Koridor Transit Utara'),
    ('MRI', 'Koridor Transit Selatan'),
    ('SUD', 'Pintu 1'),
    ('SUD', 'Pintu 2'),
    ('SUD', 'Pintu 3'),
    ('SUD', 'Pintu 4')
  ) AS v(code, label) ON s.code = v.code
);

DELETE FROM stations WHERE id IN (
  'a0000000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000002'
);
COMMIT;
