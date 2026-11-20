# Package Reference: `functional`

Import: `github.com/krewire/krewire/pkg/functional`

## 1. Purpose & Architecture

The `pkg/functional` package provides type-safe, generic higher-order functions and closure compositions in pure Go. It avoids boilerplate loops and stateful structs when transforming collections, memoizing expensive queries, or building functional middleware pipelines.

---

## 2. Main API Reference

### Generic Collection Operations

- **`functional.Map[A, B](items []A, fn func(A) B) []B`**: Transforms every slice element into a new type.
- **`functional.Filter[T](items []T, predicate func(T) bool) []T`**: Filters elements matching a predicate.
- **`functional.Reduce[A, B](items []A, initial B, fn func(B, A) B) B`**: Folds items from left to right into an accumulator.

### Function Composition & Closures

- **`functional.Pipe[T](functions ...func(T) T) func(T) T`**: Chains same-type functions left to right.
- **`functional.Compose[A, B, C](first func(A) B, second func(B) C) func(A) C`**: Composes two functions.
- **`functional.Bind[A, B, C](fn func(A, B) C, first A) func(B) C`**: Curries the first argument of a two-argument function.
- **`functional.Memoize[K comparable, V any](fn func(K) (V, error)) func(K) (V, error)`**: Thread-safe caching of function results by key.
- **`functional.MemoizeWithTTL[K comparable, V any](ttl time.Duration, fn func(K) (V, error))`**: Caches results with automatic TTL eviction.
- **`functional.Counter(start int) func() int`**: Stateful thread-safe sequence generator.

---

## 3. Usage Examples

### Collection Transformations

```go
package main

import (
    "fmt"
    "strings"
    "github.com/krewire/krewire/pkg/functional"
)

func main() {
    names := []string{"  alice ", "bob", "   charlie  "}

    // Clean names
    trimmed := functional.Map(names, strings.TrimSpace)

    // Filter names longer than 3 characters
    filtered := functional.Filter(trimmed, func(s string) bool {
        return len(s) > 3
    })

    fmt.Println(filtered) // ["alice", "charlie"]
}
```

### Memoizing Database or API Lookups

```go
package main

import (
    "fmt"
    "time"
    "github.com/krewire/krewire/pkg/functional"
)

func FetchUser(id string) (string, error) {
    fmt.Printf("Fetching user %s from remote DB...\n", id)
    return "User: " + id, nil
}

func main() {
    // Memoize with 5-minute TTL
    cachedFetch := functional.MemoizeWithTTL(5*time.Minute, FetchUser)

    user1, _ := cachedFetch("usr_100") // Calls remote DB
    user2, _ := cachedFetch("usr_100") // Returns cached value immediately
    _ = user1
    _ = user2
}
```

---

## 4. Alignment with the Three Pillars (3S)

- **Secure (S1)**: Thread-safe cache synchronization protects against concurrent map read/write panics.
- **Sustainable (S2)**: Zero third-party dependencies; uses standard Go generics.
- **Scalable (S3)**: Pure functional transforms without global side effects; easily testable.
