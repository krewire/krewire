# Package: Utilities (`hub`, `resilience`, `markdown`, `functional`, `i18n`)

A catalog of focused, single-purpose utility packages supporting the Krewire ecosystem.

---

## 1. Package Resolver & Hub (`pkg/hub`)
- **`hub/resolver`**: Resolves packages, version requirements, and semantic version constraints (`@latest`, `@v1.2.0`).
- **`hub/plugin`**: Plugin interface and dynamic capability loader.

---

## 2. Resilience (`pkg/resilience`)
- **Bulkhead**: Bounds concurrent execution slots (`NewBulkhead`) to prevent thread and goroutine exhaustion.
- **Rate Limiter**: Token-bucket frequency limiter (`NewRateLimiter`) with burst capacity and blocking wait (`Wait(ctx)`).
- **Retry**: Exponential backoff with jitter and context cancellation.
- **Circuit Breaker**: Three-state circuit breaker (`Closed`, `Open`, `HalfOpen`) preventing cascading service outages.

---

## 3. Markdown Parser (`pkg/markdown`)
- Fast, secure CommonMark parser and HTML renderer with table support, syntax highlighting hooks, and relative link rewriting.

---

## 4. Functional Utilities (`pkg/functional`)
- Type-safe generic collections: `Map`, `Filter`, `Reduce`, `Chunk`, and `Find`.

---

## 5. Internationalization (`pkg/i18n`)
- Message catalog loader, pluralization rules, and localized string formatting.
