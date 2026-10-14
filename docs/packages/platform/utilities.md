# Package: Utilities (`hub`, `resilience`, `markdown`, `functional`, `i18n`)

A catalog of focused, single-purpose utility packages supporting the Krewire ecosystem.

---

## 1. Package Resolver & Hub (`packages/hub`)
- **`hub/resolver`**: Resolves packages, version requirements, and semantic version constraints (`@latest`, `@v1.2.0`).
- **`hub/plugin`**: Plugin interface and dynamic capability loader.

---

## 2. Resilience (`packages/resilience`)
- **Retry**: Exponential backoff with jitter.
- **Circuit Breaker**: Three-state circuit breaker (`Closed`, `Open`, `HalfOpen`) preventing cascading service outages.

---

## 3. Markdown Parser (`packages/markdown`)
- Fast, secure CommonMark parser and HTML renderer with table support, syntax highlighting hooks, and relative link rewriting.

---

## 4. Functional Utilities (`packages/functional`)
- Type-safe generic collections: `Map`, `Filter`, `Reduce`, `Chunk`, and `Find`.

---

## 5. Internationalization (`packages/i18n`)
- Message catalog loader, pluralization rules, and localized string formatting.
