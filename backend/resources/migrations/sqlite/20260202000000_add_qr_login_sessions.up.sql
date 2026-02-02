PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE qr_login_sessions
(
    id            TEXT     NOT NULL PRIMARY KEY,
    created_at    DATETIME,
    token         TEXT     NOT NULL UNIQUE,
    expires_at    DATETIME NOT NULL,
    is_authorized BOOLEAN  NOT NULL DEFAULT FALSE,
    user_id       TEXT REFERENCES users ON DELETE CASCADE
);
COMMIT;
PRAGMA foreign_keys=ON;
