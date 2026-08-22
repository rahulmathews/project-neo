-- Link message history rows to their source connector and make dedupe source-aware.
ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS group_source_id uuid REFERENCES group_sources (id) ON DELETE SET NULL;

UPDATE messages m
SET group_source_id = gs.id
FROM group_sources gs
WHERE m.group_source_id IS NULL
  AND m.group_id = gs.group_id
  AND NOT EXISTS (
    SELECT 1
    FROM group_sources gs2
    WHERE gs2.group_id = m.group_id
      AND gs2.id <> gs.id
  );

ALTER TABLE messages
  DROP CONSTRAINT IF EXISTS messages_group_id_source_message_id_key;

CREATE INDEX IF NOT EXISTS messages_group_source_id_idx
  ON messages (group_source_id);

CREATE UNIQUE INDEX IF NOT EXISTS messages_group_source_message_id_key
  ON messages (group_source_id, source_message_id)
  WHERE group_source_id IS NOT NULL AND source_message_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS messages_group_source_message_id_legacy_key
  ON messages (group_id, source_message_id)
  WHERE group_source_id IS NULL AND source_message_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS messages_group_hash_timestamp_source_null_key
  ON messages (group_id, content_hash, timestamp)
  WHERE source_message_id IS NULL AND content_hash IS NOT NULL;
