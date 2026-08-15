-- matchStatusChanged subscriptions never fired for newly created matches:
-- match_updated_trigger (20260404000000_notify_triggers.sql) was UPDATE-only,
-- so a match INSERTed as PENDING emitted nothing until its first status change.
-- notify_match_updated() only reads NEW, so the function is reused unchanged.
drop trigger if exists match_updated_trigger on matches;

create trigger match_updated_trigger
  after insert or update on matches
  for each row execute function notify_match_updated();
