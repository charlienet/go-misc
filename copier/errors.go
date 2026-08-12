package copier

import "errors"

var (
	ErrInvalidCopyDestination = errors.New("copy destination must be non-nil and addressable")
	ErrInvalidCopyFrom        = errors.New("copy from must be non-nil and addressable")
	ErrMapKeyNotMatch         = errors.New("map's key type doesn't match")
	ErrNotSupported           = errors.New("not supported")
	ErrMaxDepthExceeded       = errors.New("max copy depth exceeded")
)
