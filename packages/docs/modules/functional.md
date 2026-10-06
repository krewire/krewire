# `functional`

Import: `github.com/krewire/libs/functional`

## Purpose

`functional` provides small, generic utilities built around closures. It focuses on captured configuration, captured state, function composition, and memoization without forcing callers to define helper structs.

## Main API

- `Memoize[K, V]` — caches successful function results by comparable key.
- `MemoizeWithTTL[K, V]` — caches successful results until a duration expires.
- `Bind[A, B, C]` — fixes the first argument of a two-argument function.
- `Compose[A, B, C]` — chains two functions with different intermediate types.
- `Map[A, B]` — transforms a slice into a new slice.
- `Filter[T]` — selects items into a new slice.
- `Reduce[A, B]` — folds a slice into an accumulator.
- `Pipe[T]` — composes same-type functions from left to right.
- `Counter` — returns a stateful incrementing closure.

## Example

```go
lookup := functional.Memoize(func(key string) (Record, error) {
    return repository.Find(key)
})

normalize := functional.Compose(strings.TrimSpace, strings.ToLower)
nextID := functional.Counter(100)
```

## Design boundary

This package does not replace language-level closures or standard-library primitives such as `sync.OnceValue`. It provides reusable patterns where captured state or function composition improves clarity.
