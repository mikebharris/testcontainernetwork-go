set timezone = 'UTC';

create schema if not exists "database";
set schema 'database';

create table database.messages
(
    id      serial primary key,
    message text   not null
);