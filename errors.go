package rabbit

import "errors"

var ErrTimeout = errors.New("timeout")

// ErrCritical is returned when client is unable to continue working
type ErrCritical struct {
	cause error
}

func newErrCritical(cause error) *ErrCritical {
	return &ErrCritical{cause: cause}
}

func (e *ErrCritical) Unwrap() error {
	return e.cause
}

func (e *ErrCritical) Error() string {
	return e.cause.Error()
}

func asCritical(err error) (*ErrCritical, bool) {
	if err == nil {
		return nil, false
	}

	critErr := &ErrCritical{}
	if errors.As(err, &critErr) {
		return critErr, true
	}

	return nil, false
}
