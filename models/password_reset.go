package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultResetDuration = 1 * time.Hour
)

type PasswordReset struct {
	ID     int
	UserID int
	// Token is only set when a PasswordReset is being created.
	Token     string
	TokenHash string
	ExpiresAt time.Time
}

type PasswordResetService struct {
	DB           *sql.DB
	TokenManager TokenManager
	// Duration is the amount of time that a PasswordReset is valid for.
	// Defaults to DefaultResetDuration
	Duration time.Duration
}

func (p *PasswordResetService) Create(email string) (*PasswordReset, error) {
	// Verify we have a valid email address for a user, and get that user's ID.
	email = strings.ToLower(email)
	var userID int
	row := p.DB.QueryRow(`
		SELECT id FROM users 
		WHERE email = $1;`, email)

	err := row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserDoesNotExist
		}
		return nil, fmt.Errorf("create password: %w", err)
	}

	// Build the PasswordReset.
	token, tokenHash, err := p.TokenManager.New()
	if err != nil {
		return nil, fmt.Errorf("create password: %w", err)
	}
	duration := p.Duration
	if duration == 0 {
		duration = DefaultResetDuration
	}
	pwReset := PasswordReset{
		UserID:    userID,
		Token:     token,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(duration),
	}

	// Insert the PasswordReset into the DB.
	row = p.DB.QueryRow(`
		INSERT INTO password_resets(user_id, token_hash, expires_at)
		VALUES ($1, $2, $3) ON CONFLICT (user_id) DO
		UPDATE 
		SET token_hash = $2, expires_at = $3
		RETURNING id;`, pwReset.UserID, pwReset.TokenHash, pwReset.ExpiresAt)
	err = row.Scan(&pwReset.ID)
	if err != nil {
		return nil, fmt.Errorf("create password: %w", err)
	}

	return &pwReset, nil
}

func (p *PasswordResetService) Consume(token string) (*User, error) {
	tokenHash := p.TokenManager.Hash(token)
	var user User
	var pwReset PasswordReset

	row := p.DB.QueryRow(`
		SELECT password_resets.id, password_resets.expires_at,
			users.id, users.email, users.password_hash
		FROM password_resets
			JOIN users ON users.id = password_resets.user_id
		WHERE password_resets.token_hash = $1;`, tokenHash)
	err := row.Scan(
		&pwReset.ID,
		&pwReset.ExpiresAt,
		&user.ID,
		&user.Email,
		&user.PasswordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("consume password: %w", err)
	}

	if time.Now().After(pwReset.ExpiresAt) {
		return nil, fmt.Errorf("token expired: %v", token)
	}

	err = p.delete(pwReset.ID)
	if err != nil {
		return nil, fmt.Errorf("consume password: %w", err)
	}

	return &user, nil
}

func (p *PasswordResetService) delete(id int) error {
	_, err := p.DB.Exec(`
		DELETE FROM password_resets
		WHERE id = $1;`, id)
	if err != nil {
		return fmt.Errorf("delete password: %w", err)
	}

	return nil
}
