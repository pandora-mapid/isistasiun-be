-- Down migration assumes no 'user'/'premium' rows exist yet — dropping to the
-- narrower constraint would otherwise fail with rows violating it.
ALTER TABLE operators DROP CONSTRAINT operators_role_check;
ALTER TABLE operators
    ADD CONSTRAINT operators_role_check
    CHECK (role IN ('operator', 'admin'));
