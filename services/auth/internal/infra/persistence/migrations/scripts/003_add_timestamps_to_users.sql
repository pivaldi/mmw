-- +goose Up
ALTER TABLE auth.users
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down
ALTER TABLE auth.users
    DROP COLUMN created_at,
    DROP COLUMN updated_at;
