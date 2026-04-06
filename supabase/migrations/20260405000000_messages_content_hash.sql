-- supabase/migrations/20260405000000_messages_content_hash.sql
alter table messages add column if not exists content_hash text;
create index if not exists messages_content_hash_idx on messages (content_hash);
