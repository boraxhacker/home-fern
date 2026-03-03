package route53

import "errors"

var (
	ErrHostedZoneAlreadyExists = errors.New("hosted zone already exists")
	ErrNoSuchHostedZone        = errors.New("no such hosted zone")
	ErrHostedZoneNotEmpty      = errors.New("hosted zone not empty")
	ErrInvalidChangeBatch      = errors.New("invalid change batch")
	ErrInvalidInput            = errors.New("invalid input")
)
