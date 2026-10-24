package queue

import "errors"

// permanentError marks an error that will fail identically on every retry —
// a bad payload, a deleted record, missing config. Worker.finish checks
// IsPermanent to skip straight to StatusFailed instead of burning the job's
// remaining attempts on a delay ladder that can't help.
type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// Permanent wraps err so the worker knows not to retry it.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

func IsPermanent(err error) bool {
	var pe *permanentError
	return errors.As(err, &pe)
}
