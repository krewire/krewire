package auth

import "net/http"

// Middleware is a standard http middleware. It is declared here rather than in
// httperror.go so the package has one obvious home for the middleware shape that
// BasicAuth and JWTAuth return.
type Middleware func(http.Handler) http.Handler
