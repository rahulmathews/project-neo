-- Required for ON CONFLICT (name) DO UPDATE on the groups table.
ALTER TABLE groups ADD CONSTRAINT groups_name_key UNIQUE (name);

-- Required for ON CONFLICT (source_type, source_identifier) DO UPDATE on group_sources.
ALTER TABLE group_sources ADD CONSTRAINT group_sources_source_key UNIQUE (source_type, source_identifier);
