package core

import "errors"

// Standard application errors
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInternal      = errors.New("internal server error")
	ErrInvalidKeyId  = errors.New("the parameter key id isn't valid")
)
