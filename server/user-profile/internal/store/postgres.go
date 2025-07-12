package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/omsurase/blogger_microservices/server/user-profile/internal/models"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dbURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Close() {
	s.db.Close()
}

func (s *PostgresStore) InitDB() error {
	query := `
		CREATE TABLE IF NOT EXISTS profiles (
			id UUID PRIMARY KEY,
			user_id UUID UNIQUE NOT NULL,
			bio TEXT,
			avatar_url TEXT,
			twitter_url TEXT,
			linkedin_url TEXT,
			github_url TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) GetProfileByUserID(userID uuid.UUID) (*models.Profile, error) {
	profile := &models.Profile{}
	query := `
		SELECT id, user_id, bio, avatar_url, twitter_url, linkedin_url, github_url, created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`
	err := s.db.QueryRow(query, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Bio,
		&profile.AvatarURL,
		&profile.TwitterURL,
		&profile.LinkedInURL,
		&profile.GithubURL,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting profile: %v", err)
	}
	return profile, nil
}

func (s *PostgresStore) CreateProfile(userID uuid.UUID) (*models.Profile, error) {
	profile := &models.Profile{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO profiles (id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := s.db.Exec(query,
		profile.ID,
		profile.UserID,
		profile.CreatedAt,
		profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating profile: %v", err)
	}
	return profile, nil
}

func (s *PostgresStore) UpdateProfile(userID uuid.UUID, req *models.UpdateProfileRequest) (*models.Profile, error) {
	query := `
		UPDATE profiles
		SET bio = $1, avatar_url = $2, twitter_url = $3, linkedin_url = $4, github_url = $5, updated_at = $6
		WHERE user_id = $7
		RETURNING id, user_id, bio, avatar_url, twitter_url, linkedin_url, github_url, created_at, updated_at
	`
	profile := &models.Profile{}
	err := s.db.QueryRow(
		query,
		req.Bio,
		req.AvatarURL,
		req.TwitterURL,
		req.LinkedInURL,
		req.GithubURL,
		time.Now(),
		userID,
	).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Bio,
		&profile.AvatarURL,
		&profile.TwitterURL,
		&profile.LinkedInURL,
		&profile.GithubURL,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating profile: %v", err)
	}
	return profile, nil
} 