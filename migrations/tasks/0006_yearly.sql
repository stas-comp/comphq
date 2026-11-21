-- Yearly jobs (SPEC B14.4, D-108; v1.6). Add-only, so a rollback to 1.5.0 is
-- safe: 1.5.0 never reads these columns and its inserts leave them NULL.
--
-- tasks.repeat is a TEXT column with no CHECK constraint (0003_weekly.sql), so
-- the new value 'yearly' needs no table rebuild. 1.5.0 reads anything that is
-- not 'weekly' as an ordinary job, so a rolled-back app shows a waiting yearly
-- job in To do, which CHANGES.md says; upgrading again finds it waiting again.
--
-- notice_days is the weeks of notice times 7. repeat_anchor is the month and day
-- (MM-DD) the job is due each year, so 29 February comes back in leap years.
-- shows_from is the due date minus the notice, as a local date. A yearly job is
-- WAITING while shows_from is still ahead; that is computed from the date, never
-- stored, so there is no clock job and test mode's fixed date works.
ALTER TABLE tasks ADD COLUMN notice_days INTEGER;
ALTER TABLE tasks ADD COLUMN repeat_anchor TEXT;
ALTER TABLE tasks ADD COLUMN shows_from TEXT;
