package core

type ErrorCode int

const (
	ErrNone ErrorCode = iota
	ErrInternalError
	ErrNotFound
)
