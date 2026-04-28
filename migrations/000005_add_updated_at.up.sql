-- 005_add_updated_at.sql
-- updated_at lets us detect: changed jobs (SSE), stuck jobs (worker crashed)
-- The DB is responsible for this — not application code

ALTER TABLE jobs
ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Make it consistent: set updated_at = created_at for existing rows
UPDATE jobs SET updated_at = created_at;