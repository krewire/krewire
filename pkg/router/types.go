package router

import "net/http"

// Middleware wraps an http.Handler with cross-cutting functionality.
type Middleware func(next http.Handler) http.Handler

// Params holds the path variables extracted from a route pattern.
type Params map[string]string

// HandlerFunc handles an HTTP request with route parameters.
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

// Controller is a cohesive set of routes.
type Controller interface {
	Register(r *Router)
}

// Registrar is the manual-registry counterpart to Controller.
type Registrar interface {
	RegisterRoutes(r *Router)
}

type registrarFunc func(*Router)

func (fn registrarFunc) RegisterRoutes(r *Router) { fn(r) }

// AsRegistrar wraps fn as a Registrar.
func AsRegistrar(fn func(*Router)) Registrar { return registrarFunc(fn) }

// PathParams supplies values for {param} segments in reverse URL generation.
type PathParams map[string]string
