# Platform Utilities, Engines & Storage Primitives

Comprehensive reference for supporting platform utilities, data engines, resilience primitives, and communication services across the Krewire ecosystem.

---

## 1. Package Resolver & Plugin Hub (`pkg/hub`)

Import: `github.com/krewire/krewire/pkg/hub`

The `pkg/hub` package powers package resolution, plugin registry discovery, and dynamic capability loading used by developer CLI tools (`krew add`, `krew remove`).

- **`hub/resolver`**: Parses and resolves package paths, git repositories, and semantic version constraints (`@latest`, `@v0.2.0`).
- **`hub/plugin`**: Lifecycle interface and registry for opt-in framework plugins (such as Tailwind CSS).

```go
import "github.com/krewire/krewire/pkg/hub/resolver"

pkg, ver := resolver.ParsePackageSpec("github.com/krewire/krewire@v0.2.0")
```

---

## 2. Concurrency Resilience & Protection (`pkg/resilience`)

Import: `github.com/krewire/krewire/pkg/resilience`

Provides defensive patterns protecting services and workers against thread/goroutine exhaustion, cascading upstream outages, and network jitter:

- **Bulkhead**: Limits concurrent execution slots (`NewBulkhead(maxConcurrent, maxWait)`) to bound resource consumption.
- **Rate Limiter**: Token-bucket frequency limiter (`NewRateLimiter(rate, burst)`) supporting non-blocking `Allow()` and context-cancellable `Wait(ctx)`.
- **Retry**: Exponential backoff with random full-jitter and context cancellation traps (`Retry(ctx, cfg, fn)`).
- **Circuit Breaker**: Three-state circuit breaker (`Closed`, `Open`, `HalfOpen`) tracking failure thresholds and probe recovery timeouts.

```go
cb := resilience.NewCircuitBreaker(resilience.BreakerConfig{
    MaxFailures: 5,
    Timeout:     30 * time.Second,
})

err := cb.Execute(func() error {
    return callUpstreamAPI()
})
```

---

## 3. Markdown Parser & Link Prefixing (`pkg/markdown`)

Import: `github.com/krewire/krewire/pkg/markdown`

Fast, secure CommonMark and GFM parser based on Goldmark. Used across static sites (`site`), technical books (`book`), and documentation compilers:

- `Render(source)` — Renders Markdown to sanitized HTML.
- `RenderWithBase(source, base)` — Renders Markdown and automatically prefixes eligible relative links and asset paths with a base URL.
- `PrefixLinks(html, base)` — Rewrites relative URLs in pre-rendered HTML strings.

---

## 4. Functional Utilities & Collections (`pkg/functional`)

Import: `github.com/krewire/krewire/pkg/functional`

Type-safe generic collection and functional programming utilities operating on standard Go slices and closures:

- `Map[A, B](slice, fn)` — Transforms slice elements into a new slice of target types.
- `Filter[T](slice, predicate)` — Selects matching elements into a new slice.
- `Reduce[A, B](slice, initial, accumulator)` — Folds a slice into an accumulated value.
- `Chunk[T](slice, size)` — Splits a slice into fixed-size chunks.
- `Find[T](slice, predicate)` — Returns the first matching element and a boolean flag.
- `Memoize[K, V](fn)` / `MemoizeWithTTL[K, V](ttl, fn)` — Thread-safe closure result memoization.
- `Bind[A, B, C](fn, a)` / `Compose[A, B, C](f, g)` — Function currying and composition.

---

## 5. Internationalization & Pluralization (`pkg/i18n`)

Import: `github.com/krewire/krewire/pkg/i18n`

Lightweight, zero-dependency localization engine supporting message bundles, variable interpolation, plural forms, and HTTP locale detection:

- **Message Catalogs**: Load translation bundles from JSON or YAML files (`LoadTranslations(locale, data)`).
- **Translation (`T`)**: `bundle.T("greeting", map[string]any{"Name": "Alice"})`.
- **Pluralization**: Cardinal number plural category selection (`zero`, `one`, `other`).
- **HTTP Middleware**: `i18n.Middleware(bundle)` extracts locales from query parameters (`?lang=id`), cookies, and `Accept-Language` headers, binding the active locale to `context.Context`.

---

## 6. Terminal Presentation & ANSI Colors (`pkg/term`)

Import: `github.com/krewire/tui/term`

Terminal detection, ANSI colors, text styles, and console formatting for CLI applications:

- `IsTerminal(file)` — Accurately detects interactive TTY output streams.
- `Paint(text, color, styles)` — Wraps text with ANSI escape codes; automatically stripped when output is redirected to a non-TTY pipe or file.
- `NewTerminal()` — Manages standard input and output streams.

---

## 7. Multi-Driver Caching (`pkg/cache`)

Import: `github.com/krewire/krewire/pkg/cache`

High-performance caching abstraction with driver reversibility and TTL-based eviction:

- **Supported Drivers**:
  - `NewMemoryCache(cleanupInterval)` — Pure-Go in-memory cache with concurrent TTL cleanup.
  - `NewFileCache(cfg)` — Persistent disk-backed cache.
  - `NewRedisCache(cfg)` — Network cache compatible with Redis, Valkey, Dragonfly, and KeyDB.
  - `NewMemcachedCache(cfg)` — Memcached protocol client.
- **Unified Interface (`Cache`)**: `Get`, `GetString`, `Set`, `SetString`, `Delete`, `Has`, `Clear`, `Remember`, and `Close`.

```go
val, err := c.Remember(ctx, "stats", 5*time.Minute, func() ([]byte, error) {
    return computeExpensiveStats()
})
```

---

## 8. Pure-Go Redis Client (`pkg/redis`)

Import: `github.com/krewire/krewire/pkg/redis`

Zero-CGO RESP (REdis Serialization Protocol) client supporting standalone servers and connection pools:

- **Data Structures**: Strings (`Get`, `Set`, `Incr`), Hashes (`HGet`, `HSet`, `HGetAll`), Lists (`LPush`, `RPush`, `LPop`), Sets (`SAdd`, `SMembers`, `SIsMember`).
- **Pub/Sub**: `Publish`, `Subscribe`, `PSubscribe` channels.
- **Connection Management**: Bounded connection pooling, health pings, and timeout management.

---

## 9. Multi-Driver ORM & SQL Query Builder (`pkg/orm`)

Import: `github.com/krewire/krewire/pkg/orm`

Pure-Go multi-driver ORM, SQL query builder, and migration engine supporting SQLite, PostgreSQL, and MySQL:

- **Query Builder**: Fluent SQL generation (`Table("users").Where("active = ?", true).Limit(10)`).
- **Bounded Pools**: Safe connection pooling defaults (max open, max idle, lifetime bounds) adhering to **Pillar 2 (Sustainable / S2)**.
- **Drivers**:
  - `orm/sqlite`: Opt-in pure-Go SQLite driver (`modernc.org/sqlite`).
  - `orm/postgres`: Pure-Go PostgreSQL wire protocol driver.
  - `orm/mysql`: Pure-Go MySQL driver.
- **Migrations & Seeds**: Integrated migration engine (`orm/migrate`) and seed factories.

---

## 10. Multi-Driver Transactional Mail (`pkg/mail`)

Import: `github.com/krewire/krewire/pkg/mail`

Unified transactional email delivery engine with RFC 5322 MIME multipart generation (HTML, plain text, and attachments):

- **Fluent Message Builder**: `NewMessage().SetFrom(...).AddTo(...).SetSubject(...).SetHTML(...).Attach(...)`.
- **Supported Drivers**:
  - **SMTP**: Direct TLS/STARTTLS delivery.
  - **Resend**: Resend REST API integration.
  - **SendGrid**: SendGrid v3 API integration.
  - **Postmark**: Postmark transactional email integration.
  - **Memory**: In-memory mock mailer for test verification.

---

## 11. Payment Gateway Commerce (`pkg/commerce`)

Import: `github.com/krewire/krewire/pkg/commerce`

Payment gateway integration focusing on secure checkout, tokenization, and constant-time webhook verification:

- **Midtrans Client**: Snap checkout tokenization, charge requests, and transaction status queries.
- **Webhook Security**: Constant-time SHA-512 signature verification (`crypto/subtle.ConstantTimeCompare`) blocking timing attacks.
