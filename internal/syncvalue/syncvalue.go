// Package syncvalue - allowing any type to be thread-safe
package syncvalue

import "sync"

// Value - allow storing and loading of values while guarding against race conditions.
type Value[T any] struct {
	mu    sync.RWMutex
	value T
}

// Load - loads current value.
func (l *Value[T]) Load() T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.value
}

// Store - stores new value.
func (l *Value[T]) Store(v T) {
	l.mu.Lock()
	l.value = v
	l.mu.Unlock()
}
