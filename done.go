package gather

import "sync"

// or - combines multiple done signals.
func or(done ...<-chan struct{}) <-chan struct{} {
	orCh := make(chan struct{})
	switch len(done) {
	case 0:
		close(orCh)
		return orCh
	case 1:
		return done[0]
	}
	once := sync.Once{}

	for _, dn := range done {
		go func() {
			select {
			case <-dn:
				once.Do(func() { close(orCh) })
			case <-orCh:
			}
		}()
	}

	return orCh
}

// orDone - use for easy for loops while honoring done signals.
//
// Make sure that:
// sender uses a select statement that checks for this channel signal
// receiver can loop through this safely.
func orDone[T any](in <-chan T, doneSignals ...<-chan struct{}) <-chan T {
	done := or(doneSignals...)
	out := make(chan T, cap(in))
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-done:
					return
				}
			}
		}
	}()
	return out
}
