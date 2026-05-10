-- Clean historical canonical ride display fields after duplicate collapse.

CREATE OR REPLACE FUNCTION clean_ride_location_text(input text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
  SELECT nullif(trim(regexp_replace(coalesce(input, ''),
    '( at[[:space:]]*[0-9].*| now.*| tomorrow.*| tmr.*| today.*| tonight.*| on[[:space:]].*| for[[:space:]]+[0-9].*|[[:space:]]*\(?[0-9.]+[[:space:]]*(mils?|miles?|mile|mi|km).*)$',
    '', 'i')), '');
$$;

UPDATE rides
SET from_location_text = clean_ride_location_text(from_location_text)
WHERE from_location_text IS DISTINCT FROM clean_ride_location_text(from_location_text);

UPDATE rides
SET to_location_text = clean_ride_location_text(to_location_text)
WHERE to_location_text IS DISTINCT FROM clean_ride_location_text(to_location_text);
