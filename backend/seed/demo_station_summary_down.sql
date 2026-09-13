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

-- Only drop the demo stations when nothing real hangs off them. Every child
-- table here is ON DELETE CASCADE, so an unconditional delete would take the
-- ingested field survey (flow_observations, entry_conversion_observations)
-- down with it — and the A2 workflow runs this script right before a real
-- pipeline run, which is exactly when that data must survive.
DELETE FROM stations s
WHERE s.id IN (
  'a0000000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000002'
)
AND NOT EXISTS (SELECT 1 FROM flow_observations WHERE station_id = s.id)
AND NOT EXISTS (SELECT 1 FROM entry_conversion_observations WHERE station_id = s.id)
AND NOT EXISTS (SELECT 1 FROM station_summary WHERE station_id = s.id)
AND NOT EXISTS (SELECT 1 FROM struk_extractions WHERE station_id = s.id);
COMMIT;
