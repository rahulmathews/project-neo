-- Canonicalize rides globally while preserving every source message as history.

ALTER TABLE rides
  ADD COLUMN IF NOT EXISTS semantic_fingerprint text,
  ADD COLUMN IF NOT EXISTS fingerprint_version int NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS ride_occurrences (
  id                uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  ride_id           uuid NOT NULL REFERENCES rides (id) ON DELETE CASCADE,
  message_id        uuid NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
  group_id          uuid NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
  group_source_id   uuid REFERENCES group_sources (id) ON DELETE SET NULL,
  sender_identifier text,
  content_hash      text,
  message_timestamp timestamptz NOT NULL,
  created_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (message_id),
  UNIQUE (ride_id, message_id)
);

CREATE INDEX IF NOT EXISTS ride_occurrences_ride_id_idx
  ON ride_occurrences (ride_id);

CREATE INDEX IF NOT EXISTS ride_occurrences_group_id_idx
  ON ride_occurrences (group_id);

CREATE INDEX IF NOT EXISTS ride_occurrences_content_hash_idx
  ON ride_occurrences (content_hash);

CREATE INDEX IF NOT EXISTS ride_occurrences_group_message_timestamp_idx
  ON ride_occurrences (group_id, message_timestamp DESC);

CREATE OR REPLACE FUNCTION canonical_ride_location_key(input text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
  WITH cleaned AS (
    SELECT trim(regexp_replace(lower(regexp_replace(coalesce(input, ''),
      '( at[[:space:]]*[0-9].*| now.*| tomorrow.*| tmr.*| today.*| tonight.*| on[[:space:]].*| for[[:space:]]+[0-9].*|[[:space:]]*\(?[0-9.]+[[:space:]]*(mils?|miles?|mile|km).*)$',
      '', 'i')), '[^[:alnum:]]+', ' ', 'g')) AS key
  )
  SELECT CASE key
    WHEN 'lions gate' THEN 'lionsgate'
    WHEN 'liones gate' THEN 'lionsgate'
    WHEN 'liongate' THEN 'lionsgate'
    WHEN 'pointee' THEN 'pointe'
    WHEN 'pointie' THEN 'pointe'
    WHEN 'point royal' THEN 'pointe'
    WHEN 'pointe royal' THEN 'pointe'
    WHEN 'pointeroyal' THEN 'pointe'
    WHEN 'pointroyal' THEN 'pointe'
    WHEN 'pointe royale' THEN 'pointe'
    WHEN 'oakpark mall' THEN 'oak park mall'
    WHEN 'state line' THEN 'stateline'
    WHEN 'state line road' THEN 'stateline'
    WHEN 'st line road' THEN 'stateline'
    WHEN 'stateline rd' THEN 'stateline'
    WHEN 'stateline road' THEN 'stateline'
    WHEN 'overland park' THEN 'op'
    WHEN 'overlandpark' THEN 'op'
    WHEN 'overlandpark kansas' THEN 'op'
    ELSE key
  END
  FROM cleaned;
$$;

WITH ride_fingerprints AS (
  SELECT
    r.id,
    concat_ws('|',
      'v1',
      r.type::text,
      canonical_ride_location_key(r.from_location_text),
      canonical_ride_location_key(r.to_location_text),
      CASE
        WHEN r.departure_time IS NOT NULL THEN 'at:' || to_char(r.departure_time AT TIME ZONE 'UTC', 'YYYYMMDDHH24MI')
        WHEN r.is_immediate THEN 'now:' || to_char(m.timestamp AT TIME ZONE 'UTC', 'YYYYMMDDHH24')
        WHEN substring(lower(m.content) FROM '([0-9]{1,2}[[:space:]]*[:.][[:space:]]*[0-9]{2}|[0-9]{1,2}[[:space:]]+[0-9]{2}|[0-9]{1,2})[[:space:]]*(am|pm|night)') IS NOT NULL
          THEN 'raw:' || to_char(m.timestamp AT TIME ZONE 'UTC', 'YYYYMMDD') || ':' || regexp_replace(substring(lower(m.content) FROM '([0-9]{1,2}[[:space:]]*[:.][[:space:]]*[0-9]{2}|[0-9]{1,2}[[:space:]]+[0-9]{2}|[0-9]{1,2})[[:space:]]*(am|pm|night)'), '[^a-z0-9]+', '', 'g')
        ELSE 'day:' || to_char(m.timestamp AT TIME ZONE 'UTC', 'YYYYMMDD')
      END,
      coalesce(round(r.cost * 100)::text, ''),
      coalesce(round(r.distance * 10)::text, ''),
      coalesce(r.seats_available::text, '')
    ) AS fingerprint
  FROM rides r
  JOIN messages m ON m.id = r.message_id
  WHERE r.message_id IS NOT NULL
)
UPDATE rides r
SET semantic_fingerprint = rf.fingerprint,
    fingerprint_version = 1
FROM ride_fingerprints rf
WHERE r.id = rf.id
  AND r.semantic_fingerprint IS NULL;

INSERT INTO ride_occurrences (
  ride_id,
  message_id,
  group_id,
  group_source_id,
  sender_identifier,
  content_hash,
  message_timestamp
)
SELECT
  r.id,
  m.id,
  m.group_id,
  m.group_source_id,
  m.sender_identifier,
  m.content_hash,
  m.timestamp
FROM rides r
JOIN messages m ON m.id = r.message_id
WHERE r.message_id IS NOT NULL
ON CONFLICT (message_id) DO NOTHING;

WITH canonical AS (
  SELECT DISTINCT ON (semantic_fingerprint)
    semantic_fingerprint,
    id AS canonical_id
  FROM rides
  WHERE semantic_fingerprint IS NOT NULL
  ORDER BY semantic_fingerprint, created_at ASC, id ASC
)
UPDATE ride_occurrences ro
SET ride_id = c.canonical_id
FROM rides r
JOIN canonical c ON c.semantic_fingerprint = r.semantic_fingerprint
WHERE ro.ride_id = r.id
  AND ro.ride_id <> c.canonical_id;

WITH canonical AS (
  SELECT DISTINCT ON (semantic_fingerprint)
    semantic_fingerprint,
    id AS canonical_id
  FROM rides
  WHERE semantic_fingerprint IS NOT NULL
  ORDER BY semantic_fingerprint, created_at ASC, id ASC
)
UPDATE matches ma
SET ride_id = c.canonical_id
FROM rides r
JOIN canonical c ON c.semantic_fingerprint = r.semantic_fingerprint
WHERE ma.ride_id = r.id
  AND ma.ride_id <> c.canonical_id;

WITH canonical AS (
  SELECT DISTINCT ON (semantic_fingerprint)
    semantic_fingerprint,
    id AS canonical_id
  FROM rides
  WHERE semantic_fingerprint IS NOT NULL
  ORDER BY semantic_fingerprint, created_at ASC, id ASC
)
DELETE FROM rides r
USING canonical c
WHERE r.semantic_fingerprint = c.semantic_fingerprint
  AND r.id <> c.canonical_id;

CREATE UNIQUE INDEX IF NOT EXISTS rides_semantic_fingerprint_key
  ON rides (semantic_fingerprint)
  WHERE semantic_fingerprint IS NOT NULL;

CREATE OR REPLACE FUNCTION notify_ride_added()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF new.semantic_fingerprint IS NULL THEN
    PERFORM pg_notify('rides_added', new.id::text);
  END IF;
  RETURN new;
END;
$$;

CREATE OR REPLACE FUNCTION notify_ride_occurrence_added()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify('ride_occurrences_added', json_build_object(
    'ride_id', new.ride_id,
    'group_id', new.group_id
  )::text);
  RETURN new;
END;
$$;

DROP TRIGGER IF EXISTS ride_occurrence_added_trigger ON ride_occurrences;
CREATE TRIGGER ride_occurrence_added_trigger
  AFTER INSERT ON ride_occurrences
  FOR EACH ROW EXECUTE FUNCTION notify_ride_occurrence_added();
