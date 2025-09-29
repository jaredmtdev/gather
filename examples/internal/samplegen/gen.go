package samplegen

// Range - generates int values from 0 to n-1.
func Range(n int, buffer ...int) <-chan int {
	var bufferSize int
	if len(buffer) > 0 && buffer[0] > 0 {
		bufferSize = buffer[0]
	}
	out := make(chan int, bufferSize)
	go func() {
		for i := range n {
			out <- i
		}
		close(out)
	}()
	return out
}
