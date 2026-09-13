-- No-op by design: migration 13 owns this column and constraint on fresh
-- databases. Removing them here would roll back schema that predates 16.
SELECT 1;
