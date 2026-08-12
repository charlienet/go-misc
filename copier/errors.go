package copier

import "errors"

var (
	ErrInvalidCopyDestination = errors.New("copier: destination must be valid and addressable")
	ErrInvalidCopyFrom        = errors.New("copier: source must be non-nil")
	ErrMapKeyNotMatch         = errors.New("copier: map key is not a string")
	ErrNotSupported           = errors.New("copier: type combination not supported")
	ErrMaxDepthExceeded       = errors.New("max copy depth exceeded")
	ErrMethodReturnError      = errors.New("copier: method returned error")
)
