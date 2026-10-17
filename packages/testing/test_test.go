package test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestEqual_Valid(t *testing.T) {
	// should not fail for equal
	Equal(t, "a", "a")
	Equal(t, 123, 123)
}

func TestNoError_HasError(t *testing.T) {
	NoError(t, nil)
	HasError(t, os.ErrNotExist)
}

func TestContains_Valid(t *testing.T) {
	Contains(t, "<html>hello</html>", "hello")
	NotContains(t, "<html>hello</html>", "bad")
}

func TestHTTP_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	req := NewRequest(t, "GET", "/", nil)
	rec := Record(handler, req)
	EqualStatus(t, rec, 200)
	Contains(t, rec.Body.String(), "ok")
}

func TestFile_Valid(t *testing.T) {
	dir := TempDir(t)
	path := filepath.Join(dir, "a/b.txt")
	WriteFile(t, path, "hello")
	AssertFile(t, path, "hello")
	ReadFile(t, path)
}

func TestGolden_Valid(t *testing.T) {
	name := "test_golden_mvp"
	path := filepath.Join("testdata", name+".golden")
	_ = os.Remove(path)
	t.Setenv("UPDATE_GOLDEN", "1")
	Golden(t, name, "hello golden")
	t.Setenv("UPDATE_GOLDEN", "")
	Golden(t, name, "hello golden")
	_ = os.Remove(path)
}

func TestScopeLevelConventions(t *testing.T) {
	// Scope levels: Workspace → Module → Domain → Service → Unit
	levels := []string{"Workspace", "Module", "Domain", "Service", "Unit"}
	Equal(t, len(levels), 5)
}
