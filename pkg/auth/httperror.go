package auth

import "net/http"

// HTTPError is a structured HTTP error with status, code, and message.
type HTTPError struct {
	Status  int
	Code    string
	Message string
}

// Error returns the caller-facing message, falling back to the standard status
// text so an HTTPError never surfaces an empty string to a user.
func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return http.StatusText(e.Status)
}

// Error writes err as plain text via http.Error, preserving the status of an
// *HTTPError and defaulting to 500 for every other error.
func Error(w http.ResponseWriter, err error) {
	if he, ok := err.(*HTTPError); ok {
		http.Error(w, he.Message, he.Status)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
