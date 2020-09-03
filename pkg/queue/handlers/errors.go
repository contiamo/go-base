package handlers

import "github.com/pkg/errors"

var (
	ErrSerializingHearbeat = errors.New("failed to serialize progress payload while sending heartbeat")
)
