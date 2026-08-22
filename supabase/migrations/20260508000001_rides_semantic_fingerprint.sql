-- Race-safe ride dedup via semantic fingerprint, plus message↔ride reverse link.
-- Aliases live in location_contexts; the parser computes the fingerprint in Go.

-- 0. Clean up vestiges from the earlier ride_occurrences experiment.
--    Safe to run on a fresh DB (IF EXISTS). Required on local DBs that already
--    applied the now-removed canonical_rides_occurrences migration.
DO $$
BEGIN
  IF to_regclass('public.ride_occurrences') IS NOT NULL THEN
    DROP TRIGGER IF EXISTS ride_occurrence_added_trigger ON ride_occurrences;
  END IF;
END $$;
DROP TABLE IF EXISTS ride_occurrences;
DROP FUNCTION IF EXISTS notify_ride_occurrence_added();
DROP FUNCTION IF EXISTS canonical_ride_location_key(text);
DROP FUNCTION IF EXISTS clean_ride_location_text(text);

-- Restore notify_ride_added to fire on every ride insert. The earlier
-- experiment had it skip rows with a fingerprint; with ride_occurrences
-- gone, every new canonical ride should emit rides_added.
CREATE OR REPLACE FUNCTION notify_ride_added()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify('rides_added', new.id::text);
  RETURN new;
END;
$$;

-- 1. rides: semantic fingerprint column + partial unique index.
--    Partial because legacy rows have no fingerprint (NULL); they are not deduped.
ALTER TABLE rides
  ADD COLUMN IF NOT EXISTS semantic_fingerprint text,
  ADD COLUMN IF NOT EXISTS fingerprint_version int NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX IF NOT EXISTS rides_semantic_fingerprint_key
  ON rides (semantic_fingerprint)
  WHERE semantic_fingerprint IS NOT NULL;

-- 2. messages: reverse link to canonical ride (N messages → 1 ride).
--    rides.message_id stays as the originating-message denormalization;
--    messages.ride_id is the general N:1 link covering reposts.
ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS ride_id uuid REFERENCES rides (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS messages_ride_id_idx ON messages (ride_id);

-- Backfill: every existing rides.message_id becomes messages.ride_id.
UPDATE messages m
   SET ride_id = r.id
  FROM rides r
 WHERE r.message_id = m.id
   AND m.ride_id IS NULL;

-- 3. location_contexts: normalized alias column for fuzzy matching.
--    "Lions Gate", "lions-gate", "LionsGate" all hash to the same key.
--    Lookup uses (group_id, alias_normalized); insertion uses raw location_alias.
ALTER TABLE location_contexts
  ADD COLUMN IF NOT EXISTS alias_normalized text
    GENERATED ALWAYS AS (
      lower(regexp_replace(coalesce(location_alias, ''), '[^[:alnum:]]+', '', 'g'))
    ) STORED;

CREATE INDEX IF NOT EXISTS location_contexts_group_alias_normalized_idx
  ON location_contexts (group_id, alias_normalized);
