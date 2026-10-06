# Service Mesh — Key Rules

- Opt-in cost: `framework/app` without `service` shows no binary or startup change.
- Fakes for every interface; `go test` without containers is the gate.
- Gateway missing upstream returns `502` `Problem` JSON, not HTML; route reload is atomic without dropping in-flight requests.
- Circuit breaker state machine with `OnStateChange` callbacks; retries honor context deadlines.
- Tracing propagates W3C `traceparent` automatically via middleware.
- Messaging at-least-once; handler errors auto-nack/redeliver; poison routes to DLQ.
- Backoff is mandatory on Watch errors; never busy-loop or panic on transient backend unavailability.
