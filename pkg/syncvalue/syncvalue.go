package syncvalue

import "sync"

type Value[T any] struct {
	mu    sync.RWMutex
	value T
}

func (l *Value[T]) Load() T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.value
}

func (l *Value[T]) Store(v T) {
	l.mu.Lock()
	l.value = v
	l.mu.Unlock()
}
