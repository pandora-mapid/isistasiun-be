DELETE FROM rental_assets
WHERE data_source = 'space_kai'
  AND source_id IN (
    '75153818818957567', '75153818819205543', '75153818818422713',
    '75153818833728806', '75153818843382536', '75153818843418498',
    '75153818843418475', '75153818843418165'
  );

DELETE FROM rental_assets
WHERE data_source = 'field_survey'
  AND source_id LIKE 'local-sudirman-kios-%';
