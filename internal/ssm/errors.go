package ssm

import (
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"net/http"
)

const (
	ErrParameterNotFound core.ErrorCode = iota + 1000
	ErrParameterAlreadyExists
	ErrInvalidKeyId
	ErrInvalidName
	ErrInvalidTier
	ErrInvalidDataType
	ErrInvalidFilterKey
	ErrInvalidFilterOption
	ErrInvalidFilterValue
	ErrUnsupportedParameterType
	ErrInvalidPath
)

type errorCodeMap map[core.ErrorCode]awslib.ApiError

var ErrorCodes = errorCodeMap{
	core.ErrInternalError: {
		Code:           "InternalError",
		Description:    "We encountered an internal error, please try again.",
		HTTPStatusCode: http.StatusInternalServerError,
	},
	ErrParameterNotFound: {
		Code:           "ParameterNotFound",
		Description:    "The ParameterData Name provided does not exist.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrParameterAlreadyExists: {
		Code:           "ParameterAlreadyExists",
		Description:    "The parameter already exists. You can't create duplicate parameters.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrUnsupportedParameterType: {
		Code:           "UnsupportedParameterType",
		Description:    "The parameter type isn't supported.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidKeyId: {
		Code:           "InvalidKeyId",
		Description:    "The ParameterData KeyId is not valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidName: {
		Code:           "InvalidParameterName",
		Description:    "The ParameterData Name is not valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidDataType: {
		Code:           "InvalidDataType",
		Description:    "The ParameterData DataType is not valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidTier: {
		Code:           "InvalidParameterTier",
		Description:    "The ParameterData Tier is not valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidFilterKey: {
		Code:           "InvalidParameterTier",
		Description:    "The specified key isn't valid.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidFilterOption: {
		Code:           "InvalidFilterOption",
		Description:    "The specified filter option isn't valid. Valid options are Equals and BeginsWith. For Path filter, valid options are Recursive and OneLevel.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidFilterValue: {
		Code:           "InvalidFilterValue",
		Description:    "The filter value isn't valid. Verify the value and try again.",
		HTTPStatusCode: http.StatusBadRequest,
	},
	ErrInvalidPath: {
		Code:           "ValidationException",
		Description:    "The parameter doesn't meet the parameter name requirements. The parameter name must begin with a forward slash '/'.",
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
