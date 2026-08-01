package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloud-engineer-test/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) CreateSession(ctx context.Context, id, extension string) (*model.Session, error) {
	sess := &model.Session{ID: id, CurrentVersion: 1, Extension: extension}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (id, current_version, extension) VALUES ($1, $2, $3)`,
		id, 1, extension)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Store) GetSession(ctx context.Context, id string) (*model.Session, error) {
	var sess model.Session
	err := s.pool.QueryRow(ctx,
		`SELECT id, current_version, extension, created_at FROM sessions WHERE id = $1`, id).
		Scan(&sess.ID, &sess.CurrentVersion, &sess.Extension, &sess.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("session %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) CreateJob(ctx context.Context, sessionID string, op model.OperationType, params map[string]any, inputVersion int) (*model.Job, error) {
	job := &model.Job{
		ID:            uuid.NewString(),
		SessionID:     sessionID,
		Operation:     op,
		Params:        params,
		InputVersion:  inputVersion,
		OutputVersion: inputVersion + 1,
		Status:        model.JobStatusPending,
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO jobs (id, session_id, operation, params, input_version, output_version, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		job.ID, sessionID, string(op), job.ParamsJSON(), inputVersion, inputVersion+1, string(model.JobStatusPending))
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) LatestJob(ctx context.Context, sessionID string) (*model.Job, error) {
	job, err := s.scanJob(ctx, `
		SELECT id, session_id, operation, params, input_version, output_version, status, error, created_at
		FROM jobs WHERE session_id = $1
		ORDER BY created_at DESC, id DESC LIMIT 1`, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (s *Store) ClaimNextJob(ctx context.Context) (*model.Job, error) {
	job, err := s.scanJob(ctx, `
		UPDATE jobs
		SET status = $1
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = $2
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, session_id, operation, params, input_version, output_version, status, error, created_at`,
		string(model.JobStatusProcessing), string(model.JobStatusPending))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (s *Store) CompleteJob(ctx context.Context, jobID, sessionID string, inputVersion int, outputExt string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE sessions SET current_version = current_version + 1, extension = $1
		 WHERE id = $2 AND current_version = $3`,
		outputExt, sessionID, inputVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("session %s advanced past job %s input version %d", sessionID, jobID, inputVersion)
	}

	_, err = tx.Exec(ctx,
		`UPDATE jobs SET status = $1, error = '' WHERE id = $2`,
		string(model.JobStatusDone), jobID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) FailJob(ctx context.Context, jobID string, errMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET status = $1, error = $2 WHERE id = $3`,
		string(model.JobStatusFailed), errMsg, jobID)
	return err
}

func (s *Store) scanJob(ctx context.Context, query string, args ...any) (*model.Job, error) {
	var (
		job      model.Job
		params   []byte
		status   string
		errorMsg string
	)
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&job.ID, &job.SessionID, &job.Operation, &params, &job.InputVersion,
		&job.OutputVersion, &status, &errorMsg, &job.CreatedAt)
	if err != nil {
		return nil, err
	}
	job.Status = model.JobStatus(status)
	job.Error = errorMsg
	if len(params) > 0 {
		_ = json.Unmarshal(params, &job.Params)
	}
	if job.Params == nil {
		job.Params = map[string]any{}
	}
	return &job, nil
}
