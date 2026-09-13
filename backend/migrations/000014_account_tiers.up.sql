-- Adds the two self-service tiers ("user" = free registered account,
-- "premium" = user after the pretend-upgrade) alongside the existing
-- admin-provisioned "operator"/"admin". Same table, same CHECK constraint,
-- just a wider allow-list.
ALTER TABLE operators DROP CONSTRAINT operators_role_check;
ALTER TABLE operators
    ADD CONSTRAINT operators_role_check
    CHECK (role IN ('user', 'premium', 'operator', 'admin'));
