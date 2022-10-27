-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
--
create table categories (
id bigserial primary key,
user_id bigint,
name text,
created_at timestamp,
updated_at timestamp
);
--
create table currencies (
user_id bigint unique,
char_code text,
created_at timestamp,
updated_at timestamp
);
--
create table spendings (
id bigserial primary key,
user_id bigint,
category_id bigint,
amount decimal(20, 8),
date timestamp,
created_at timestamp,
updated_at timestamp
);
--
create table limits (
user_id bigint unique,
amount text,
created_at timestamp,
updated_at timestamp
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
--
DROP TABLE categories;
--
DROP TABLE currencies;
--
DROP TABLE spendings;
--
DROP TABLE limits;
-- +goose StatementEnd