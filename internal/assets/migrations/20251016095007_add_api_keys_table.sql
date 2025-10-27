-- +goose Up
-- +goose StatementBegin
create table api_keys (
    id integer primary key autoincrement,
    value text unique not null,
    created_on timestamp not null default current_timestamp
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table api_keys;
-- +goose StatementEnd
