package errs

import (
	"errors"
	"fmt"
)

// ErrInvalidNumber - error when invalid number is detected.
var ErrInvalidNumber = errors.New("invalid number")

// NewErrInvalidNumber - creates a new ErrInvalidNumber.
func NewErrInvalidNumber(in int) error {
	return fmt.Errorf("%v: %w", in, ErrInvalidNumber)
}
