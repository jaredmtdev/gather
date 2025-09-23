package seq

import (
	"context"
	"iter"
	"sync"
)

// ToChan - iter.Seq to chan
func ToChan[T any](ctx context.Context, in iter.Seq[T], buffer int) <-chan T {
	out := make(chan T, buffer)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case <-ctx.Done():
				return
			case out <- v:
			}
		}
	}()
	return out
}

// ToChans - slice of iter.Seqs to slice of channels
func ToChans[T any](ctx context.Context, ins []iter.Seq[T], buffer int) []<-chan T {
	outs := make([]<-chan T, len(ins))
	wg := sync.WaitGroup{}
	wg.Add(len(ins))
	for i, in := range ins {
		go func() {
			out := make(chan T, buffer)
			defer close(out)
			outs[i] = out
			wg.Done()
			for v := range in {
				select {
				case <-ctx.Done():
					return
				case out <- v:
				}
			}
		}()
	}
	wg.Wait()
	return outs
}

// FromChan - chan to iter.Seq
func FromChan[T any](ctx context.Context, in <-chan T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok || !yield(v) {
					return
				}
			}
		}
	}
}

func FromChans[T any](ctx context.Context, ins []<-chan T) []iter.Seq[T] {
	outs := make([]iter.Seq[T], len(ins))
	for i, in := range ins {
		outs[i] = func(yield func(T) bool) {
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-in:
					if !ok || !yield(v) {
						return
					}
				}
			}
		}
	}
	return outs
}
