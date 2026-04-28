-- we use uuids instead of serial (modern and more secure)
-- each image will have slightly different metadata, so we use JSONB
-- to factor in this variation.
-- each job must have a payload/metadata (NOT NULL)

CREATE TABLE IF NOT EXISTS jobs (
    id         UUID        PRIMARY KEY DEFAULT uuidv7(),
    payload    JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);