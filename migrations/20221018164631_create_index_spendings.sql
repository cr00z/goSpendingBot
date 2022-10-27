-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
--
CREATE INDEX ON spendings(user_id,date);
--
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
--
DROP INDEX spendings_user_id_date_idx;
-- +goose StatementEnd