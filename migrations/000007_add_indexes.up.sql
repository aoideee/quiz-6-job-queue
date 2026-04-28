-- 007_add_indexes.sql

-- B-tree composite index: for worker polling (filter by status, order by created_at)
CREATE INDEX idx_jobs_status_created ON jobs (status, created_at);

-- B-tree index: for updated_at queries (SSE / stuck job detection)
CREATE INDEX idx_jobs_updated_at ON jobs (updated_at DESC);

-- GIN index: for JSONB containment queries on payload (@>)
CREATE INDEX idx_jobs_payload ON jobs USING GIN (payload);

-- GIN index: for JSONB containment queries on result (@>)
CREATE INDEX idx_jobs_result ON jobs USING GIN (result);