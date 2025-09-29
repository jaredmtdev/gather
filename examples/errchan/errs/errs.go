package errs

import (
	"errors"
	"fmt"
)

var ErrInvalidNumber = errors.New("invalid number")

func NewErrInvalidNumber(in int) error {
	return fmt.Errorf("%v: %w", in, ErrInvalidNumber)
}
