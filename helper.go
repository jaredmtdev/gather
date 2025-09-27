package gather

// orDone - use to close a done channel. make sure that:
// sender uses a select statement that checks for this channel signal
// receiver can loop through this safely
func orDone[T any](in <-chan T, done chan struct{}) <-chan T {
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
