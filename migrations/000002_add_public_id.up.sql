-- 002_add_public_id.sql
-- uuidv7() embeds a timestamp in the first 48 bits — leaks info to clients
-- So we add a random public_id using uuidv4() for external use

ALTER TABLE jobs
ADD COLUMN public_id UUID NOT NULL UNIQUE DEFAULT uuidv4();