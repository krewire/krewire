package web

import (
	"net/http"
	"testing"

	ftest "github.com/krewire/krewire/pkg/testing"
)

func TestHTTP_UsingFtest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("krewire"))
	})
	req := ftest.NewRequest(t, "GET", "/", nil)
	rec := ftest.Record(handler, req)
	ftest.EqualStatus(t, rec, http.StatusOK)
	ftest.Contains(t, rec.Body.String(), "krewire")
	ftest.Equal(t, rec.Body.String(), "krewire")
}
