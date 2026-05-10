-- Strip remaining cost/seat suffixes from historical canonical ride locations.

CREATE OR REPLACE FUNCTION clean_ride_location_text(input text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
  SELECT nullif(trim(regexp_replace(coalesce(input, ''),
    '( at[[:space:]]*[0-9].*| now.*| tomorrow.*| tmr.*| today.*| tonight.*| on[[:space:]].*| for[[:space:]]+([0-9]+|one|two|three|four|five|six|seven|eight).*$|[[:space:]]*\(?[0-9.]+[[:space:]]*(mils?|miles?|mile|mi|km).*$|[[:space:]]+[0-9]+[[:space:]]*[$₹£€&].*$)$',
    '', 'i')), '');
$$;

UPDATE rides
SET from_location_text = clean_ride_location_text(from_location_text)
WHERE from_location_text IS DISTINCT FROM clean_ride_location_text(from_location_text);

UPDATE rides
SET to_location_text = clean_ride_location_text(to_location_text)
WHERE to_location_text IS DISTINCT FROM clean_ride_location_text(to_location_text);
