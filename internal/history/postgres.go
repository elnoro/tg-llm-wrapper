package history

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type MessageType string

const (
	UserMessageType  MessageType = "user"
	ModelMessageType MessageType = "model"
)

type PostgresMessageStorage struct {
	db *sql.DB
}

func NewPostgresMessageStorage(ctx context.Context, connectionString string) (*PostgresMessageStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	storage := &PostgresMessageStorage{db: db}

	if err := storage.AutoMigrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return storage, nil
}

func (s *PostgresMessageStorage) AutoMigrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			type TEXT NOT NULL CHECK (type IN ('user', 'model')),
			message TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	return nil
}

func (s *PostgresMessageStorage) Record(ctx context.Context, userID int64, userMessage, modelMessage string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO messages (user_id, type, message)
		VALUES ($1, $2, $3)
	`

	// Insert user message
	_, err = tx.ExecContext(ctx, query, userID, UserMessageType, userMessage)
	if err != nil {
		return fmt.Errorf("failed to record user message: %w", err)
	}

	// Insert model message
	_, err = tx.ExecContext(ctx, query, userID, ModelMessageType, modelMessage)
	if err != nil {
		return fmt.Errorf("failed to record model message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
