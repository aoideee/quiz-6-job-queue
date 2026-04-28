package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Job mirrors the jobs table.
type Job struct {
	ID         string          `json:"id"`
	PublicID   string          `json:"public_id"`
	Payload    json.RawMessage `json:"payload"`
	Status     string          `json:"status"`
	Progress   int             `json:"progress"`
	Result     json.RawMessage `json:"result"`
	ErrorMsg   *string         `json:"error_msg,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// UpdateParams holds the fields a worker can update.
type UpdateParams struct {
	Status   *string
	Progress *int
	Result   json.RawMessage
	ErrorMsg *string
}

// Store wraps the database connection.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateJob inserts a new job and returns it.
func (s *Store) CreateJob(ctx context.Context, payload json.RawMessage) (*Job, error) {
	const q = `
		INSERT INTO jobs (payload)
		VALUES ($1)
		RETURNING id, public_id, payload, status, progress, result, error_msg, created_at, updated_at
	`
	return scanJob(s.db.QueryRowContext(ctx, q, string(payload)))
}

// GetJob fetches a single job by its internal UUID.
func (s *Store) GetJob(ctx context.Context, id string) (*Job, error) {
	const q = `
		SELECT id, public_id, payload, status, progress, result, error_msg, created_at, updated_at
		FROM jobs WHERE id = $1
	`
	return scanJob(s.db.QueryRowContext(ctx, q, id))
}

// ClaimNextJob atomically claims the oldest pending job for a worker.
func (s *Store) ClaimNextJob(ctx context.Context) (*Job, error) {
	const q = `
		UPDATE jobs
		SET status = 'processing', progress = 0
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'pending'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, public_id, payload, status, progress, result, error_msg, created_at, updated_at
	`
	return scanJob(s.db.QueryRowContext(ctx, q))
}

// UpdateJob applies partial updates to a job.
func (s *Store) UpdateJob(ctx context.Context, id string, p UpdateParams) (*Job, error) {
	// Build query dynamically to only touch the columns provided.
	// updated_at is handled automatically by the DB trigger.
	const q = `
		UPDATE jobs SET
			status    = COALESCE($2, status),
			progress  = COALESCE($3, progress),
			result    = COALESCE($4::jsonb, result),
			error_msg = COALESCE($5, error_msg)
		WHERE id = $1
		RETURNING id, public_id, payload, status, progress, result, error_msg, created_at, updated_at
	`

	var resultStr *string
	if p.Result != nil {
		s := string(p.Result)
		resultStr = &s
	}

	return scanJob(s.db.QueryRowContext(ctx, q, id, p.Status, p.Progress, resultStr, p.ErrorMsg))
}

// JobsSince returns all jobs updated after the given time, ordered by updated_at.
func (s *Store) JobsSince(ctx context.Context, since time.Time) ([]*Job, error) {
	const q = `
		SELECT id, public_id, payload, status, progress, result, error_msg, created_at, updated_at
		FROM jobs
		WHERE updated_at > $1
		ORDER BY updated_at ASC
	`
	rows, err := s.db.QueryContext(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		j, err := scanJobRow(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// scanJob scans a *sql.Row into a Job.
func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	var result, payload string
	err := row.Scan(
		&j.ID, &j.PublicID, &payload,
		&j.Status, &j.Progress,
		&result, &j.ErrorMsg,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	j.Payload = json.RawMessage(payload)
	j.Result = json.RawMessage(result)
	return &j, nil
}

// scanJobRow scans a *sql.Rows into a Job.
func scanJobRow(rows *sql.Rows) (*Job, error) {
	var j Job
	var result, payload string
	err := rows.Scan(
		&j.ID, &j.PublicID, &payload,
		&j.Status, &j.Progress,
		&result, &j.ErrorMsg,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	j.Payload = json.RawMessage(payload)
	j.Result = json.RawMessage(result)
	return &j, nil
}
