// Package functional provides small, generic function-composition utilities.
// The utilities return closures that capture configuration or state without
// requiring callers to define a dedicated struct.
package functional

import (
	"errors"
	"sync"
	"time"
)

// ErrNilFunction indicates that a required function argument was nil.
var ErrNilFunction = errors.New("functional: nil function")

// Memoize returns a function that caches successful results by key. The
// function is safe for concurrent use. Errors are returned to the caller and
// are not cached.
func Memoize[K comparable, V any](fn func(K) (V, error)) func(K) (V, error) {
	var mu sync.Mutex
	cache := make(map[K]V)
	if fn == nil {
		return func(K) (V, error) { var zero V; return zero, ErrNilFunction }
	}
	return func(key K) (V, error) {
		mu.Lock()
		value, ok := cache[key]
		mu.Unlock()
		if ok {
			return value, nil
		}
		value, err := fn(key)
		if err != nil {
			var zero V
			return zero, err
		}
		mu.Lock()
		if existing, exists := cache[key]; exists {
			value = existing
		} else {
			cache[key] = value
		}
		mu.Unlock()
		return value, nil
	}
}

// Bind fixes the first argument of a two-argument function and returns a
// single-argument closure.
func Bind[A, B, C any](fn func(A, B) C, first A) func(B) C {
	return func(second B) C { return fn(first, second) }
}

// Compose returns a closure that applies first and then second.
func Compose[A, B, C any](first func(A) B, second func(B) C) func(A) C {
	return func(value A) C { return second(first(value)) }
}

// Counter returns a stateful closure that increments and returns its value.
func Counter(start int) func() int {
	value := start
	return func() int { value++; return value }
}

// MemoizeWithTTL caches successful results until ttl expires. A non-positive
// ttl disables caching and invokes fn for every call.
func MemoizeWithTTL[K comparable, V any](ttl time.Duration, fn func(K) (V, error)) func(K) (V, error) {
	if fn == nil {
		return func(K) (V, error) { var zero V; return zero, ErrNilFunction }
	}
	if ttl <= 0 {
		return fn
	}
	var mu sync.Mutex
	type entry struct {
		value   V
		expires time.Time
	}
	cache := make(map[K]entry)
	if fn == nil {
		return func(K) (V, error) { var zero V; return zero, ErrNilFunction }
	}
	return func(key K) (V, error) {
		now := time.Now()
		mu.Lock()
		cached, ok := cache[key]
		if ok && now.Before(cached.expires) {
			mu.Unlock()
			return cached.value, nil
		}
		mu.Unlock()
		value, err := fn(key)
		if err != nil {
			var zero V
			return zero, err
		}
		mu.Lock()
		cache[key] = entry{value: value, expires: time.Now().Add(ttl)}
		mu.Unlock()
		return value, nil
	}
}

// Map transforms every item and returns a new slice.
func Map[A, B any](items []A, fn func(A) B) []B {
	if fn == nil {
		return nil
	}
	out := make([]B, len(items))
	for i, item := range items {
		out[i] = fn(item)
	}
	return out
}

// Filter returns a new slice containing items accepted by predicate.
func Filter[T any](items []T, predicate func(T) bool) []T {
	if predicate == nil {
		return nil
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		if predicate(item) {
			out = append(out, item)
		}
	}
	return out
}

// Reduce folds items from left to right into an accumulator.
func Reduce[A, B any](items []A, initial B, fn func(B, A) B) B {
	if fn == nil {
		return initial
	}
	result := initial
	for _, item := range items {
		result = fn(result, item)
	}
	return result
}

// Pipe composes same-type functions from left to right.
func Pipe[T any](functions ...func(T) T) func(T) T {
	return func(value T) T {
		result := value
		for _, fn := range functions {
			if fn != nil {
				result = fn(result)
			}
		}
		return result
	}
}
