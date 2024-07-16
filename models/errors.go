package models

import "errors"

var (
	ErrEmailTaken       = errors.New("models: email address is already in use")
	ErrUserDoesNotExist = errors.New("models: user with provided email address does not exist")
	ErrNotFound         = errors.New("models: resource could not be found")
)
