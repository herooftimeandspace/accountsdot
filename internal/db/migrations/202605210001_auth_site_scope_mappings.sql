create table if not exists auth_site_scope_mappings (
    id bigserial primary key,
    source_type text not null check (source_type in ('group', 'attribute')),
    source_value text not null,
    attribute_values jsonb not null default '[]'::jsonb,
    site_codes jsonb not null default '[]'::jsonb,
    actor_id text not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (source_type, source_value)
);

create index if not exists auth_site_scope_mappings_source_idx
    on auth_site_scope_mappings (source_type, source_value);
