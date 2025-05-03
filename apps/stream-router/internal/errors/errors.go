package errors

import "errors"

var (
	ErrStreamNotFound    = errors.New("stream not found")
	ErrUnauthorized      = errors.New("unauthorized access to stream")
	ErrInvalidInput      = errors.New("invalid input")
	ErrStreamAlreadyLive = errors.New("stream is already live")
	ErrStreamNotLive     = errors.New("stream is not live")
)
