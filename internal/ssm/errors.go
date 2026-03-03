package ssm

import "errors"

var (
	ErrParameterNotFound        = errors.New("parameter not found")
	ErrParameterAlreadyExists   = errors.New("parameter already exists")
	ErrInvalidKeyId             = errors.New("the parameter key id isn't valid")
	ErrInvalidName              = errors.New("the parameter name isn't valid")
	ErrInvalidTier              = errors.New("the parameter tier isn't valid")
	ErrInvalidDataType          = errors.New("the parameter data type isn't valid")
	ErrInvalidFilterKey         = errors.New("the specified filter key isn't valid")
	ErrInvalidFilterOption      = errors.New("the specified filter option isn't valid")
	ErrInvalidFilterValue       = errors.New("the filter value isn't valid")
	ErrUnsupportedParameterType = errors.New("the parameter type isn't supported")
	ErrInvalidPath              = errors.New("the path isn't valid")
)
