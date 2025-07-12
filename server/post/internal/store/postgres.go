package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/omsurase/blogger_microservices/server/post/internal/models"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func createTables(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS posts (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			tags TEXT[] NOT NULL DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL
		)`

	_, err := db.Exec(query)
	return err
}

func (s *PostgresStore) CreatePost(post *models.Post) error {
	query := `
		INSERT INTO posts (id, user_id, title, content, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := s.db.Exec(query,
		post.ID,
		post.UserID,
		post.Title,
		post.Content,
		pq.Array(post.Tags),
		post.CreatedAt,
		post.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) GetPost(id uuid.UUID) (*models.Post, error) {
	query := `
		SELECT id, user_id, title, content, tags, created_at, updated_at
		FROM posts
		WHERE id = $1`

	post := &models.Post{}
	err := s.db.QueryRow(query, id).Scan(
		&post.ID,
		&post.UserID,
		&post.Title,
		&post.Content,
		pq.Array(&post.Tags),
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return post, err
}

func (s *PostgresStore) UpdatePost(post *models.Post) error {
	query := `
		UPDATE posts
		SET title = $1, content = $2, tags = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6`

	result, err := s.db.Exec(query,
		post.Title,
		post.Content,
		pq.Array(post.Tags),
		post.UpdatedAt,
		post.ID,
		post.UserID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("post not found or user not authorized")
	}
	return nil
}

func (s *PostgresStore) DeletePost(id, userID uuid.UUID) error {
	query := `DELETE FROM posts WHERE id = $1 AND user_id = $2`

	result, err := s.db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("post not found or user not authorized")
	}
	return nil
}

func (s *PostgresStore) GetPostsByUser(userID uuid.UUID, page, pageSize int) ([]models.Post, int, error) {
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM posts WHERE user_id = $1`
	var totalCount int
	err := s.db.QueryRow(countQuery, userID).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, title, content, tags, created_at, updated_at
		FROM posts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.Query(query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			pq.Array(&post.Tags),
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	return posts, totalCount, rows.Err()
}

func (s *PostgresStore) GetPostsByTag(tag string, page, pageSize int) ([]models.Post, int, error) {
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM posts WHERE $1 = ANY(tags)`
	var totalCount int
	err := s.db.QueryRow(countQuery, tag).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, title, content, tags, created_at, updated_at
		FROM posts
		WHERE $1 = ANY(tags)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.Query(query, tag, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			pq.Array(&post.Tags),
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, post)
	}

	return posts, totalCount, rows.Err()
} 