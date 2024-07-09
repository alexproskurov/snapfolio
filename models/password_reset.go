package models

import (
	"database/sql"
	"fmt"
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
	return nil,fmt.Errorf("not implemented")
}

func (p *PasswordResetService) Consume(token string) (*User, error) {
	return nil, fmt.Errorf("not implemented")
}