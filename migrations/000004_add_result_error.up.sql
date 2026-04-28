-- 004_add_result_error.sql
-- result stores output file paths incrementally (workers append one key at a time)
-- Always initialize result to '{}' so the || merge operator works
-- error_msg is plain TEXT (nullable)

ALTER TABLE jobs
ADD COLUMN result    JSONB NOT NULL DEFAULT '{}',
ADD COLUMN error_msg TEXT;