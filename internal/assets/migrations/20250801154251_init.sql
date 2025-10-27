-- +goose Up
-- +goose StatementBegin
create table roles (
    role_id integer primary key,
    role_name text
);

insert into roles (role_id, role_name) values (10000, 'superuser');
insert into roles (role_id, role_name) values (1000, 'admin');
insert into roles (role_id, role_name) values (10, 'operator');

create table users (
    user_id integer primary key autoincrement,
    user_role_id integer not null references roles(role_id) on delete restrict,
    username text not null unique,
    password_hash text,
    password_changed_on timestamp
);

create table auth_sessions (
    auth_session_id text primary key,
    auth_session_user_id integer not null references users(user_id) on delete cascade,
    auth_session_expires TIMESTAMP
);

create table credentials (
    credential_id integer primary key autoincrement,
    username text,
    description text,
    ssh_private_key_hash text
);

create table agents (
    agent_id integer primary key autoincrement,
    agent_credential_id integer references credentials(credential_id) on delete restrict,
    name text unique,
    hostname text,
    workspace text,
    description text
);

create table pipelines (
    pipeline_id integer primary key autoincrement,
    pipeline_agent_id integer references agents(agent_id) on delete restrict,
    name text,
    description text,
    repository text,
    script_path text,
    schedule text,
    schedule_branch text,
    schedule_job_id text unique
);

create table runs (
    run_id integer primary key autoincrement,
    run_pipeline_id integer references pipelines(pipeline_id) on delete set null,
    branch text,
    working_directory text,
    output text,
    artifacts text,
    status text, -- queued, running, cancelled, passed, failed
    created_on timestamp not null default current_timestamp,
    started_on timestamp,
    ended_on timestamp,
    archive bool not null default false
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
drop table runs;
drop table pipelines;
drop table agents;
drop table credentials;
drop table auth_sessions;
drop table users;
drop table roles;
-- +goose StatementEnd
