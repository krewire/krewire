# `sec`

Import: `github.com/krewire/libs/sec`

## Purpose

HTTP security middleware and security-safe diagnostics: headers, CORS, CSRF, health endpoints, authorization policies, outbound URL validation, and PII masking.

## Main API

- `SecurityHeaders`, `CORS`, `CSRF`, `Health`
- `Require`, `Authenticated`, `WithRoles`
- `ValidateOutboundURL`
- `MaskPII`, `MaskPIIMap`, `MaskPIIAttrs`
- `NewPIIHandler` for automatic `log/slog` redaction
- `StripTags`

## Example

```go
handler := sec.SecurityHeaders()(
    sec.CORS(sec.WithOrigins("https://example.com"))(
        sec.Require(sec.Authenticated())(protectedHandler),
    ),
)
logger := slog.New(sec.NewPIIHandler(slog.NewJSONHandler(os.Stdout, nil)))
```

## Security boundaries

`ValidateOutboundURL` validates input but performs no network request and cannot alone prevent DNS rebinding; consumers must validate resolved addresses and restrict ports. `StripTags` extracts text and is not a context-safe HTML sanitizer. PII masking protects diagnostics, not storage encryption, authorization, or retention.

Authentication APIs re-exported by `sec` are compatibility aliases; new authentication code should use [`auth`](./auth.md).
