package gather

import (
	"errors"
	"fmt"
	"time"
)

var (
	errInvalidWorkerSize     = errors.New("must use at least 1 worker")
	errInvalidMinWorkerSize  = errors.New("must use at least 0 min workers")
	errInvalidBufferSize     = errors.New("buffer must be at least 0")
	errInvalidTTL            = errors.New("ttl must use at least 0 ns")
	errMinWorkerSizeTooLarge = errors.New("minWorkerSize cannot be larger than workerSize")
)

func newInvalidWorkerSizeError(workerSize int) error {
	return fmt.Errorf("%w. workerSize: %v", errInvalidWorkerSize, workerSize)
}

func newInvalidMinWorkerSizeError(minWorkerSize int) error {
	return fmt.Errorf("%w. minWorkerSize: %v", errInvalidMinWorkerSize, minWorkerSize)
}

func newInvalidBufferSizeError(bufferSize int) error {
	return fmt.Errorf("%w. bufferSize: %v", errInvalidBufferSize, bufferSize)
}

func newInvalidTTLError(ttl time.Duration) error {
	return fmt.Errorf("%w. ttlElastic: %v ns", errInvalidTTL, ttl.Nanoseconds())
}

func newMinWorkerSizeTooLargeError(minWorkerSize, maxWorkerSize int) error {
	return fmt.Errorf("%w. minWorkerSize: %v workerSize: %v", errMinWorkerSizeTooLarge, minWorkerSize, maxWorkerSize)
}
