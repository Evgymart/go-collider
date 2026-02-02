create table users (
    user_id uuid not null default gen_random_uuid() primary key,
    name varchar(50) not null,
    created_at timestamp(0) not null default (now() at time zone 'Europe/Moscow')
);

create table event_types (
    type_id uuid not null default gen_random_uuid() primary key,
    name varchar(255) not null unique
);

create table events (
    event_id uuid not null default gen_random_uuid() primary key,
    user_id uuid not null,
    type_id uuid not null,
    "timestamp" timestamp(0) not null default (now() at time zone 'Europe/Moscow'),
    metadata jsonb not null,
    constraint events_type_id_fkey foreign key (type_id)
        references event_types (type_id) match simple
        on update no action
        on delete no action,
    constraint events_user_id_fkey foreign key (user_id)
        references users (user_id) match simple
        on update no action
        on delete no action
);