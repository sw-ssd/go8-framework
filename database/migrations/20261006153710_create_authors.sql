-- +up
CREATE TABLE IF NOT EXISTS authors (
    id BIGSERIAL PRIMARY KEY,
    TEXT first_name NOT NULL DEFAULT '',
    TEXT middle_name NULL,
    TEXT last_name NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- +down
DROP TABLE IF EXISTS authors;
