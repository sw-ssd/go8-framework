-- +up
CREATE TABLE IF NOT EXISTS books (
    id BIGSERIAL PRIMARY KEY,
    TEXT title NOT NULL DEFAULT '',
    TIMESTAMPTZ published_date NOT NULL,
    TEXT image_url NULL,
    TEXT description NOT NULL DEFAULT '',
    BIGINT author_id NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- +down
DROP TABLE IF EXISTS books;
