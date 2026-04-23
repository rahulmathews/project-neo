-- supabase/migrations/20260419000000_messages_retry_count.sql
ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS retry_count int NOT NULL DEFAULT 0;
