create table users (
    user_id bigserial not null primary key,
    name varchar(50) not null,
    created_at timestamp(0) not null default (now() at time zone 'Europe/Moscow')
);

create table event_types (
    type_id bigserial not null primary key,
    name varchar(255) not null unique
);

create table events (
    event_id bigserial not null primary key,
    user_id bigint not null,
    type_id bigint not null,
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

create index idx_events_user_timestamp on events (user_id, "timestamp" desc);
create index idx_events_timestamp_desc on events ("timestamp" desc);
create index idx_events_type_timestamp on events (type_id, "timestamp" desc);
create index idx_events_stats on events (user_id, (metadata->>'page'), type_id);
create index idx_events_covering on events (user_id, type_id, "timestamp" desc) include (event_id, metadata);
create index idx_events_metadata_gin on events using gin (metadata);
