create index if not exists messages_group_content_hash_idx
  on messages (group_id, content_hash);
