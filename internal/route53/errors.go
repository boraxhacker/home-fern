package route53

import (
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"log"
	"net/http"
)

const (
	ErrInvalidInput core.ErrorCode = iota + 5300
	ErrHostedZoneAlreadyExists
	ErrNoSuchHostedZone
	ErrInvalidChangeBatch
)

type errorCodeMap map[core.ErrorCode]awslib.ApiError

var ErrorCodes = errorCodeMap{
	core.ErrInternalError: {
		Code:           "InternalError",
		Description:    "We encountered an internal error, please try again.",
		HTTPStatusCode: http.StatusInternalServerError,
	},
	ErrHostedZoneAlreadyExists: {
		Code:           "HostedZoneAlreadyExists",
		Description:    "The hosted zone you're trying to create already exists.",
		HTTPStatusCode: http.StatusConflict,
	},
	ErrNoSuchHostedZone: {
		Code:           "NoSuchHostedZone",
		Description:    "No hosted zone exists with the ID that you specified.",
		HTTPStatusCode: http.StatusNotFound,
	},
	ErrInvalidInput: {
		Code:           "InvalidInput",
		Description:    "The input is not valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidChangeBatch: {
		Code:           "InvalidChangeBatch",
		Description:    "The input is not valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
}

func translateToApiError(ec core.ErrorCode) awslib.ApiError {

	value, ok := ErrorCodes[ec]
	if !ok {
		value = awslib.ErrorCodes[awslib.ErrInternalError]
	}

	log.Println("Error:", value)

	return value
}
