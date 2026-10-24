# `app`

Import: `github.com/krewire/krewire/pkg/app`

## Purpose

Dependency-injection container, application assembly, and service-provider lifecycle root for fullstack and backend applications.

## Main API

- `app.New(opts ...Option) *Container`: Creates an empty composition root (`NewContainer` is an alias). Bindings are added with `Provide`/`Singleton` and resolved with `Resolve`.
- `app.NewApp(providers ...Provider) *App`: Creates an application that assembles the given providers. `Register` then `Boot` run when `Build` or `Test` is called.
- `(*App).Build() (*Container, error)`: Runs every provider's `Register` in order, then every `Boot`, and returns the locked container.
- `(*App).Start(ctx, c) error` / `(*App).Stop(ctx, c) error`: Run registered start/stop hooks with `app.DefaultStopTimeout` (10s) as the shutdown budget.
- `(*App).OnStart(fn)` / `(*App).OnStop(fn)` / `(*App).AddHook(h)`: Register lifecycle hooks.
- `app.Provide[T](c, factory)`: Registers a factory for `T`; each `Resolve` builds a fresh value.
- `app.Singleton[T](c, factory)`: Registers a factory whose result is resolved once.
- `app.Resolve[T](c) (T, error)` / `app.ResolveNamed[T](c, name) (T, bool)`: Typed resolution.
- `app.Provider`: Interface with `Register(c *Container) error` and `Boot(c *Container) error`.
- `app.ProviderFunc`, `app.HttpRunner(handler, addr)`: Provider and HTTP runner helpers.

## Example

```go
package main

import (
	"context"
	"log"

	"github.com/krewire/krewire/pkg/app"
)

type DatabaseProvider struct{}

func (d *DatabaseProvider) Register(c *app.Container) error {
	return c.Set("db", "postgres://localhost:5432/app")
}

func (d *DatabaseProvider) Boot(c *app.Container) error { return nil }

func main() {
	a := app.NewApp(&DatabaseProvider{})

	c, err := a.Build()
	if err != nil {
		log.Fatal(err)
	}
	if err := a.Start(context.Background(), c); err != nil {
		log.Fatal(err)
	}
	defer a.Stop(context.Background(), c)
}
```

## Design boundary

`app` owns assembly order and lifecycle only. Domain rules stay in `pkg/model`; runtime mechanics stay in `pkg/lifecycle`. It never imports `tools/kiw`.
