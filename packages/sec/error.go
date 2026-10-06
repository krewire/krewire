package sec

import (
	"net/http"

	"github.com/krewire/krewire/packages/auth"
)

// Error writes the HTTPError as JSON or plain text via http.Error.
func Error(w http.ResponseWriter, err error) {
	auth.Error(w, err)
}

// Middleware is a standard http middleware.
type Middleware = auth.Middleware

// cookieVal is re-exported from auth for backward compatibility within sec.
func cookieVal(r *http.Request, name string) string {
	return auth.CookieValue(r, name)
}
