alter table auth_site_scope_mappings
    drop constraint if exists auth_site_scope_mappings_source_type_check;

alter table auth_site_scope_mappings
    add constraint auth_site_scope_mappings_source_type_check
    check (source_type in ('group', 'ou', 'attribute'));

create table if not exists auth_role_mappings (
    id bigserial primary key,
    source_type text not null check (source_type in ('group', 'ou', 'attribute')),
    source_value text not null,
    attribute_values jsonb not null default '[]'::jsonb,
    role_keys jsonb not null default '[]'::jsonb,
    actor_id text not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (source_type, source_value)
);

create index if not exists auth_role_mappings_source_idx
    on auth_role_mappings (source_type, source_value);

create table if not exists external_data_sources (
    provider_key text primary key,
    provider_label text not null,
    sync_enabled boolean not null default false,
    last_test_status text,
    last_test_summary text,
    last_test_at timestamptz,
    actor_id text not null default 'system',
    reason text not null default 'registry_default_off',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists external_provider_credentials (
    id bigserial primary key,
    provider_key text not null references external_data_sources(provider_key) on delete cascade,
    field_key text not null,
    encrypted_value text not null,
    key_id text not null,
    fingerprint text not null,
    label text not null,
    actor_id text not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (provider_key, field_key)
);

create index if not exists external_provider_credentials_provider_idx
    on external_provider_credentials (provider_key);
