package gather

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkerOpts(t *testing.T) {
	testCases := []struct {
		name        string
		opts        []Opt
		expectedErr error
	}{
		{
			name:        "negative buffer",
			opts:        []Opt{WithBufferSize(-1)},
			expectedErr: errInvalidBufferSize,
		},
		{
			name:        "zero worker size",
			opts:        []Opt{WithWorkerSize(0)},
			expectedErr: errInvalidWorkerSize,
		},
		{
			name:        "min workers larger than max workers",
			opts:        []Opt{WithWorkerSize(2), WithElasticWorkers(3, time.Millisecond)},
			expectedErr: errMinWorkerSizeTooLarge,
		},
		{
			name:        "negative min workers",
			opts:        []Opt{WithElasticWorkers(-1, time.Millisecond)},
			expectedErr: errInvalidMinWorkerSize,
		},
		{
			name:        "negative ttl",
			opts:        []Opt{WithElasticWorkers(1, -time.Millisecond)},
			expectedErr: errInvalidTTL,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := newWorkerOpts(tc.opts).validate()
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
