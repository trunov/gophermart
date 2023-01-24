-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS orders
(
    id              uuid DEFAULT uuid_generate_v4(),
    number          VARCHAR(16) NOT NULL,
    status          SMALLINT NOT NULL,
    user_id         uuid REFERENCES users(id),
    accrual         NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    created_at      TIMESTAMP NOT NULL DEFAULT now(),
    updated_at      TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT number UNIQUE (number)
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
