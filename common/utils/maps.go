package utils

import (
	"sync"
)

type MapThSf[K comparable, T comparable] struct {
	storage sync.Map
}

func (m *MapThSf[K, T]) CompareAndDelete(key K, old T) (deleted bool) {
	return m.storage.CompareAndDelete(key, old)
}
func (m *MapThSf[K, T]) CompareAndSwap(key K, old T, new T) bool {
	return m.storage.CompareAndSwap(key, old, new)
}
func (m *MapThSf[K, T]) Delete(key K) { m.storage.Delete(key) }
func (m *MapThSf[K, T]) Load(key K) (T, bool) {
	var zero T
	value, ok := m.storage.Load(key)
	if !ok {
		return zero, ok
	}
	return value.(T), ok
}
func (m *MapThSf[K, T]) LoadAndDelete(key K) (T, bool) {
	value, loaded := m.storage.LoadAndDelete(key)
	if !loaded {
		var zero T
		return zero, loaded
	}
	return value.(T), loaded
}
func (m *MapThSf[K, T]) LoadOrStore(key K, value T) (T, bool) {
	actual, loaded := m.storage.LoadOrStore(key, value)
	if !loaded {
		var zero T
		return zero, loaded
	}
	return actual.(T), loaded
}
func (m *MapThSf[K, T]) Range(f func(key K, value T) bool) {
	m.storage.Range(func(key, value any) bool {
		return f(key.(K), value.(T))
	})
}
func (m *MapThSf[K, T]) Store(key K, value T) { m.storage.Store(key, value) }
func (m *MapThSf[K, T]) Swap(key K, value T) (T, bool) {
	previous, loaded := m.storage.Swap(key, value)
	if !loaded {
		var zero T
		return zero, loaded
	}
	return previous.(T), loaded
}
