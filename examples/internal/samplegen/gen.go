package samplegen

// Range - generates int values from 0 to n-1
func Range(n int) <-chan int {
	out := make(chan int)
	go func() {
		for i := range n {
			out <- i
		}
		close(out)
	}()
	return out
}
