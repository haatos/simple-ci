-- +goose Up
-- +goose StatementBegin
alter table agents add column os_type text default "unix";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table agents drop column os_type;
-- +goose StatementEnd
