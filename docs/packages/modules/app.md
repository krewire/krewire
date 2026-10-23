# `app`

Import: `github.com/krewire/krewire/packages/app`

## Purpose

Dependency injection container, application assembly, and service provider lifecycle root for fullstack and backend applications.

## Main API

- `app.New()`: Initializes the application composition root.
- `app.Register(provider)`: Registers a service provider.
- `app.Boot(ctx)`: Boots all registered providers in dependency order.
- `app.Resolve[T](app)`: Resolves a typed service singleton from the container.
- `app.Provider`: Interface defining `Register()` and `Boot()` lifecycle hooks.

## Example

```go
package main

import (
	"context"
	"github.com/krewire/krewire/packages/app"
)

type DatabaseProvider struct{}

func (d *DatabaseProvider) Register(a *app.App) error {
	return a.Bind("db", "postgres://localhost:5432/app")
}

func (d *DatabaseProvider) Boot(ctx context.Context, a *app.App) error {
	return nil
}

func main() {
	application := app.New()
	application.Register(&DatabaseProvider{})
	_ = application.Boot(context.Background())
}
```

## Design boundary

`app` orchestrates domain services at the application level. It never imports `kiw`.
