-- Weekly jobs (SPEC B13.5, D-95; v1.4). Add-only, so a rollback to 1.3.0 is
-- safe: 1.3.0 never reads these columns, and its inserts take the defaults,
-- so a job made while rolled back is an ordinary job. A weekly job opened on
-- 1.3.0 is an ordinary job there (it may even be tucked away into Finished
-- tasks after 14 days); upgrading again, the next Saturday's reset finds it
-- still marked weekly and brings it back (gate 7.39).
--
-- repeat is 'none' or 'weekly'. repeat_reset_on is the Saturday (YYYY-MM-DD)
-- the job was last put right: the reset runs again only once a later Saturday
-- has come, which is what stops it running twice for one Saturday.
ALTER TABLE tasks ADD COLUMN repeat TEXT NOT NULL DEFAULT 'none';
ALTER TABLE tasks ADD COLUMN repeat_reset_on TEXT;
