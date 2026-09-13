-- An operator account is scoped to the single station it represents (KAI /
-- KAI Commuter / kawasan operator — section 4.1); admin accounts stay
-- unscoped and see every station. Nullable because the CHECK below only
-- fires on INSERT/UPDATE, not on rows already present, so an existing admin
-- row with no station never breaks.
ALTER TABLE operators
    ADD COLUMN station_id UUID REFERENCES stations(id);

-- NOT VALID: enforces the rule for every future INSERT/UPDATE without
-- validating rows that already exist, so this migration can't fail on an
-- environment that already has an operator account with no station_id.
ALTER TABLE operators
    ADD CONSTRAINT operators_operator_role_needs_station
    CHECK (role <> 'operator' OR station_id IS NOT NULL) NOT VALID;
