-- Demo snapshot for the public rental-assets layer.
-- Manggarai rows are an exact-name snapshot from Space KAI. Sudirman rows are
-- local field-survey coordinates because the upstream endpoint has no exact
-- `namalokasi = Sudirman` records.

BEGIN;

INSERT INTO rental_assets (
    station_id, source_id, data_source, location_name, plot_name, area_name,
    latitude, longitude, land_area, building_area, rented, availability_status,
    commercial_value, commercial_value_visible, source_updated_at, note
)
SELECT s.id, v.source_id, v.data_source, v.location_name, v.plot_name, v.area_name,
       v.latitude, v.longitude, v.land_area, v.building_area, v.rented,
       v.availability_status, v.commercial_value, v.commercial_value_visible,
       v.source_updated_at, v.note
FROM (VALUES
    ('75153818818957567', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN MANGGARAI 50,09M (OUTLET CFC)', 'Daop 1 Jakarta', -6.209931205497052::double precision, 106.85023123473187::double precision, 50.09::numeric, 0::numeric, true,  'occupied', 1577342000::numeric, false, '2026-08-21 06:36:04+07'::timestamptz, 'Aset Space KAI; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818819205543', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN BESAR MANGGARAI 34,20M (FAMILY MART)', 'Daop 1 Jakarta', -6.209867210204161::double precision, 106.85000592918202::double precision, 34.2::numeric, 0::numeric, true,  'occupied', 250000000::numeric, false, '2026-08-21 04:58:14+07'::timestamptz, 'Aset Space KAI; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818818422713', 'space_kai', 'Manggarai', 'SPACE STASIUN MANGGARAI 36M (OUTLET INDOMARTE PINTU TIMUR)', 'Daop 1 Jakarta', -6.209774277055045::double precision, 106.85015876559434::double precision, 36::numeric, 0::numeric, true,  'occupied', 200000000::numeric, false, '2026-08-21 04:43:34+07'::timestamptz, 'Aset Space KAI; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818833728806', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN MANGGARAI 84,30M (TOTO EXPRESS)', 'Daop 1 Jakarta', -6.209288317907934::double precision, 106.84975628276572::double precision, 84.3::numeric, 0::numeric, true,  'occupied', 150000000::numeric, false, '2026-08-21 04:23:01+07'::timestamptz, 'Aset Space KAI; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818843382536', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN MANGGARAI NO 5 49,5M', 'Daop 1 Jakarta', -6.2093009274974165::double precision, 106.84980529761327::double precision, 49.5::numeric, 0::numeric, false, 'available', 594000000::numeric, false, '2026-04-10 02:56:16+07'::timestamptz, 'Aset Space KAI tersedia; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818843418498', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN MANGGARAI NO 6 49,5M', 'Daop 1 Jakarta', -6.209331051895262::double precision, 106.84982124612299::double precision, 49.5::numeric, 0::numeric, false, 'available', 594000000::numeric, false, '2026-04-10 02:55:44+07'::timestamptz, 'Aset Space KAI tersedia; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818843418475', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN MANGGARAI NO 7 49,5M', 'Daop 1 Jakarta', -6.209361176291391::double precision, 106.8498148667191::double precision, 49.5::numeric, 0::numeric, false, 'available', 594000000::numeric, false, '2026-04-10 02:54:50+07'::timestamptz, 'Aset Space KAI tersedia; nilai komersial tidak ditampilkan oleh sumber.')),
    ('75153818843418165', 'space_kai', 'Manggarai', 'SPACE EMPLASEMEN STASIUN MANGGARAI NO 8  49,5M', 'Daop 1 Jakarta', -6.209391300685802::double precision, 106.8498419791856::double precision, 49.5::numeric, 0::numeric, false, 'available', 594000000::numeric, false, '2026-04-10 02:28:02+07'::timestamptz, 'Aset Space KAI tersedia; nilai komersial tidak ditampilkan oleh sumber.')),
    ('local-sudirman-kios-1', 'field_survey', 'Sudirman', 'Kios lokal Sudirman 1', 'Kios 1', 'Area depan stasiun', -6.2024901::double precision, 106.8236287::double precision, NULL::numeric, NULL::numeric, false, 'needs_verification', NULL::numeric, false, NULL::timestamptz, 'Koordinat survei lokal; status, luas, dan harga sewa perlu verifikasi.')),
    ('local-sudirman-kios-2', 'field_survey', 'Sudirman', 'Kios lokal Sudirman 2', 'Kios 2', 'Area depan stasiun', -6.2024691::double precision, 106.8235871::double precision, NULL::numeric, NULL::numeric, false, 'needs_verification', NULL::numeric, false, NULL::timestamptz, 'Koordinat survei lokal; status, luas, dan harga sewa perlu verifikasi.')),
    ('local-sudirman-kios-3', 'field_survey', 'Sudirman', 'Kios lokal Sudirman 3', 'Kios 3', 'Area depan stasiun', -6.2024697::double precision, 106.8234600::double precision, NULL::numeric, NULL::numeric, false, 'needs_verification', NULL::numeric, false, NULL::timestamptz, 'Koordinat survei lokal; status, luas, dan harga sewa perlu verifikasi.')
) AS v(source_id, data_source, location_name, plot_name, area_name, latitude, longitude,
       land_area, building_area, rented, availability_status, commercial_value,
       commercial_value_visible, source_updated_at, note)
JOIN stations s ON s.code = CASE v.location_name WHEN 'Manggarai' THEN 'MRI' ELSE 'SUD' END
ON CONFLICT (data_source, source_id) DO UPDATE SET
    station_id = EXCLUDED.station_id,
    location_name = EXCLUDED.location_name,
    plot_name = EXCLUDED.plot_name,
    area_name = EXCLUDED.area_name,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    land_area = EXCLUDED.land_area,
    building_area = EXCLUDED.building_area,
    rented = EXCLUDED.rented,
    availability_status = EXCLUDED.availability_status,
    commercial_value = EXCLUDED.commercial_value,
    commercial_value_visible = EXCLUDED.commercial_value_visible,
    source_updated_at = EXCLUDED.source_updated_at,
    note = EXCLUDED.note,
    updated_at = now();

COMMIT;
