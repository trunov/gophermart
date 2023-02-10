-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users
(
    id              uuid DEFAULT uuid_generate_v4(),
    login           VARCHAR(64) NOT NULL,
    password        VARCHAR(64) NOT NULL,
    balance         NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    created_at      TIMESTAMP NOT NULL DEFAULT now(),
    updated_at      TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    CONSTRAINT login_unique UNIQUE (login)
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
