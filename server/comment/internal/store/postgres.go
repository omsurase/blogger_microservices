package store

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/comment/internal/models"
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dbURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) InitDB() error {
	query := `
		CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY,
			post_id UUID NOT NULL,
			user_id UUID NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateComment(comment *models.Comment) error {
	query := `
		INSERT INTO comments (id, post_id, user_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := s.db.Exec(query,
		comment.ID,
		comment.PostID,
		comment.UserID,
		comment.Content,
		comment.CreatedAt,
	)
	return err
}

func (s *PostgresStore) GetCommentsByPostID(postID uuid.UUID) ([]models.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, created_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

func (s *PostgresStore) GetCommentByID(id uuid.UUID) (*models.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, created_at
		FROM comments
		WHERE id = $1`

	var comment models.Comment
	err := s.db.QueryRow(query, id).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.UserID,
		&comment.Content,
		&comment.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (s *PostgresStore) DeleteComment(id uuid.UUID) error {
	query := `DELETE FROM comments WHERE id = $1`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
} 