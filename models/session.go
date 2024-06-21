package models

import (
	"database/sql"
	"fmt"
)

type Session struct {
	ID     int
	UserID int
	// Token is only set when creating a new session.
	// When looking up a session this will be left empty,
	// as we only store the hash of a session token
	// in our database and we cannot reverse it into a raw token.
	Token     string
	TokenHash string
}

type SessionService struct {
	DB           *sql.DB
	TokenManager TokenManager
}

// Create will create a new session for the user provided. The session token
// will be returned as the Token field on the Session type, but only the hashed
// session token is stored in the database.
func (ss *SessionService) Create(userID int) (*Session, error) {
	token, tokenHash, err := ss.TokenManager.New()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	session := Session{
		UserID:    userID,
		Token:     token,
		TokenHash: tokenHash,
	}

	row := ss.DB.QueryRow(`
		UPDATE sessions
		SET token_hash=$2
		WHERE user_id=$1
		RETURNING id;`, session.UserID, session.TokenHash)
	err = row.Scan(&session.ID)
	if err == sql.ErrNoRows {
		row = ss.DB.QueryRow(`
		INSERT INTO sessions (user_id, token_hash)
		VALUES ($1, $2) 
		RETURNING id;`, session.UserID, session.TokenHash)
		err = row.Scan(&session.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &session, nil
}

func (ss *SessionService) User(token string) (*User, error) {
	tokenHash := ss.TokenManager.Hash(token)
	var user User
	row := ss.DB.QueryRow(`
		SELECT users.id, users.email, users.password_hash 
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = $1;`, tokenHash)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("user session: %w", err)
	}

	return &user, nil
}

func (ss *SessionService) Delete(token string) error {
	tokenHash := ss.TokenManager.Hash(token)
	_, err := ss.DB.Exec(`
		DELETE
		FROM sessions 
		WHERE token_hash = $1;`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
