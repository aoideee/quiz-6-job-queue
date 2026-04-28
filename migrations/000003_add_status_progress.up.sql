-- 003_add_status_progress.sql
-- Status tracks where the job is in its lifecycle
-- Progress tracks percentage complete (0-100)
-- These are NOT stored in the JSONB payload — they are first-class columns

ALTER TABLE jobs
ADD COLUMN status   TEXT    NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'processing', 'done', 'failed')),
ADD COLUMN progress INTEGER NOT NULL DEFAULT 0
    CHECK (progress BETWEEN 0 AND 100);