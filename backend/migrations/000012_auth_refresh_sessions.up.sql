CREATE TABLE operator_refresh_sessions (
    token_id             UUID PRIMARY KEY,
    family_id            UUID NOT NULL,
    operator_id          UUID NOT NULL REFERENCES operators(id) ON DELETE CASCADE,
    expires_at           TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at         TIMESTAMPTZ,
    revoked_at           TIMESTAMPTZ,
    replaced_by_token_id UUID REFERENCES operator_refresh_sessions(token_id)
);

CREATE INDEX idx_operator_refresh_sessions_family
    ON operator_refresh_sessions (family_id);

CREATE INDEX idx_operator_refresh_sessions_operator_active
    ON operator_refresh_sessions (operator_id, expires_at)
    WHERE revoked_at IS NULL;
