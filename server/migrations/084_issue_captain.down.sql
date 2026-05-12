DROP INDEX IF EXISTS idx_issue_captain;

ALTER TABLE issue
    DROP CONSTRAINT IF EXISTS issue_captain_pair_check;

ALTER TABLE issue
    DROP COLUMN IF EXISTS captain_id,
    DROP COLUMN IF EXISTS captain_type;
