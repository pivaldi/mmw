-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth.event (
    id           BIGSERIAL PRIMARY KEY,
    event_type   VARCHAR(100) NOT NULL,
    payload      JSONB NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_auth_unpublished ON auth.event(occurred_at ASC) WHERE published_at IS NULL;
CREATE INDEX idx_auth_published   ON auth.event(published_at)    WHERE published_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_auth_published;
DROP INDEX IF EXISTS idx_auth_unpublished;
DROP TABLE IF EXISTS auth.event;
-- +goose StatementEnd
