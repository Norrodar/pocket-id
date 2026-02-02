CREATE TABLE qr_login_sessions
(
    id            UUID        NOT NULL PRIMARY KEY,
    created_at    TIMESTAMPTZ,
    token         TEXT        NOT NULL UNIQUE,
    expires_at    TIMESTAMPTZ NOT NULL,
    is_authorized BOOLEAN     NOT NULL DEFAULT FALSE,
    user_id       UUID REFERENCES users ON DELETE CASCADE
);
