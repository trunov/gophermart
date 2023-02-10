-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS withdrawal
(
    id              uuid DEFAULT uuid_generate_v4(),
    amount          NUMERIC(10,2) NOT NULL,
    order_id        VARCHAR(16) NOT NULL,
    user_id         uuid REFERENCES users(id),
    processed_at    TIMESTAMP NOT NULL DEFAULT now()
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
