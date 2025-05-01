package kms

import (
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"net/http"
)

const (
	ErrInvalidCiphertextException core.ErrorCode = iota + 4000
	ErrKMSInternalException
)

type errorCodeMap map[core.ErrorCode]awslib.ApiError

var ErrorCodes = errorCodeMap{
	core.ErrInternalError: {
		Code:           "InternalError",
		Description:    "We encountered an internal error, please try again.",
		HTTPStatusCode: http.StatusInternalServerError,
	},
	ErrInvalidCiphertextException: {
		Code:           "InvalidCiphertextException",
		Description:    "The request was rejected because the specified ciphertext, or additional authenticated data incorporated into the ciphertext, such as the encryption context, is corrupted, missing, or otherwise invalid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrKMSInternalException: {
		Code:           "KMSInternalException",
		Description:    "The request was rejected because an internal exception occurred. The request can be retried.",
		HTTPStatusCode: http.StatusBadRequest,
	},
}

func translateToApiError(ec core.ErrorCode) awslib.ApiError {

	value, ok := ErrorCodes[ec]
	if ok {

		return value
	}

	return awslib.ErrorCodes[awslib.ErrInternalError]
}
