package kms

import "errors"

var (
	ErrInvalidCiphertextException = errors.New("invalid ciphertext exception")
	ErrKMSInternalException       = errors.New("kms internal exception")
	ErrInvalidKeyId               = errors.New("invalid key id")
)
