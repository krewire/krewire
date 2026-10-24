package i18n_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/krewire/krewire/pkg/i18n"
)

func sampleFileSystem() fstest.MapFS {
	return fstest.MapFS{
		// Indonesian
		"lang/id/auth.json": &fstest.MapFile{
			Data: []byte(`{
				"login": "Masuk ke Akun",
				"btn": {
					"submit": "Kirim"
				}
			}`),
		},
		"lang/id.json": &fstest.MapFile{
			Data: []byte(`{
				"app_title": "Aplikasi Krewire",
				"auth": {
					"forgot": "Lupa Kata Sandi",
					"login": "Masuk Fallback"
				}
			}`),
		},

		// English (Default fallback)
		"lang/en/auth.json": &fstest.MapFile{
			Data: []byte(`{
				"login": "Log in",
				"logout": "Log out"
			}`),
		},
		"lang/en.json": &fstest.MapFile{
			Data: []byte(`{
				"app_title": "Krewire App",
				"global_msg": "Hello {name}",
				"messages": {
					"zero": "No new messages",
					"one": "1 new message",
					"other": "{count} new messages"
				},
				"items": "No items | One item | {count} items"
			}`),
		},

		// French (Single file only)
		"lang/fr.json": &fstest.MapFile{
			Data: []byte(`{
				"welcome": "Bienvenue {{name}}!"
			}`),
		},

		// Direct file match
		"lang/id/welcome.json": &fstest.MapFile{
			Data: []byte(`{
				"message": "Selamat datang :name!"
			}`),
		},
	}
}

func TestBundle_NamespacedKeyWithFallback(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys,
		i18n.WithBasePath("lang"),
		i18n.WithDefaultLocale("en"),
		i18n.WithFallbackLocale("en"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating bundle: %v", err)
	}

	if bundle.DefaultLocale() != "en" {
		t.Errorf("expected default locale 'en', got '%s'", bundle.DefaultLocale())
	}
	if bundle.FallbackLocale() != "en" {
		t.Errorf("expected fallback locale 'en', got '%s'", bundle.FallbackLocale())
	}

	locales := bundle.Locales()
	if len(locales) < 3 {
		t.Errorf("expected at least 3 locales, got %v", locales)
	}
	if !bundle.HasLocale("id") || !bundle.HasLocale("en") || !bundle.HasLocale("fr") {
		t.Errorf("expected locales to contain id, en, fr; got %v", locales)
	}

	// 1. Target key exists in namespaced file lang/id/auth.json (takes priority over lang/id.json)
	got := bundle.Translate("id", "auth.login")
	if got != "Masuk ke Akun" {
		t.Errorf("expected 'Masuk ke Akun', got '%s'", got)
	}

	// 2. Nested subkey in namespaced file
	got = bundle.Translate("id", "auth.btn.submit")
	if got != "Kirim" {
		t.Errorf("expected 'Kirim', got '%s'", got)
	}

	// 3. Key missing in namespaced file, but present in fallback file lang/id.json
	got = bundle.Translate("id", "auth.forgot")
	if got != "Lupa Kata Sandi" {
		t.Errorf("expected 'Lupa Kata Sandi', got '%s'", got)
	}

	// 4. Root key in fallback file lang/id.json
	got = bundle.Translate("id", "app_title")
	if got != "Aplikasi Krewire" {
		t.Errorf("expected 'Aplikasi Krewire', got '%s'", got)
	}

	// 5. Key missing in target locale 'id', falls back to default locale 'en' (namespaced file)
	got = bundle.Translate("id", "auth.logout")
	if got != "Log out" {
		t.Errorf("expected 'Log out', got '%s'", got)
	}

	// 6. Key missing in target locale 'id', falls back to default locale 'en' (fallback file)
	got = bundle.Translate("id", "global_msg", "name", "Budi")
	if got != "Hello Budi" {
		t.Errorf("expected 'Hello Budi', got '%s'", got)
	}

	// 7. Non-existent key returns key name
	got = bundle.Translate("id", "non.existent.key")
	if got != "non.existent.key" {
		t.Errorf("expected 'non.existent.key', got '%s'", got)
	}
}

func TestBundle_SingleFileAndDirectLookup(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test single file fr.json with {{name}} template
	got := bundle.Translate("fr", "welcome", map[string]any{"name": "Pierre"})
	if got != "Bienvenue Pierre!" {
		t.Errorf("expected 'Bienvenue Pierre!', got '%s'", got)
	}

	// Test direct file welcome.json with :name template
	got = bundle.Translate("id", "welcome", "name", "Budi")
	if got != "Selamat datang Budi!" {
		t.Errorf("expected 'Selamat datang Budi!', got '%s'", got)
	}
}

func TestBundle_AddTranslation(t *testing.T) {
	bundle, err := i18n.New(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bundle.AddTranslation("es", "auth", map[string]any{
		"login": "Iniciar sesión",
	})
	bundle.AddTranslation("es", "", map[string]any{
		"title": "Bienvenido",
	})

	if !bundle.HasLocale("es") {
		t.Error("expected locale es to be registered")
	}
	if got := bundle.Translate("es", "auth.login"); got != "Iniciar sesión" {
		t.Errorf("expected 'Iniciar sesión', got '%s'", got)
	}
	if got := bundle.Translate("es", "title"); got != "Bienvenido" {
		t.Errorf("expected 'Bienvenido', got '%s'", got)
	}
}

func TestBundle_Pluralization(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Map-based pluralization
	if got := bundle.Translate("en", "messages", "count", 0); got != "No new messages" {
		t.Errorf("count=0: expected 'No new messages', got '%s'", got)
	}
	if got := bundle.Translate("en", "messages", "count", int64(1)); got != "1 new message" {
		t.Errorf("count=1: expected '1 new message', got '%s'", got)
	}
	if got := bundle.Translate("en", "messages", "count", float64(5)); got != "5 new messages" {
		t.Errorf("count=5: expected '5 new messages', got '%s'", got)
	}
	if got := bundle.Translate("en", "messages", "count", "10"); got != "10 new messages" {
		t.Errorf("count='10': expected '10 new messages', got '%s'", got)
	}

	// Pipe-separated pluralization
	if got := bundle.Translate("en", "items", "count", 0); got != "No items" {
		t.Errorf("count=0: expected 'No items', got '%s'", got)
	}
	if got := bundle.Translate("en", "items", "count", 1); got != "One item" {
		t.Errorf("count=1: expected 'One item', got '%s'", got)
	}
	if got := bundle.Translate("en", "items", "count", 3); got != "3 items" {
		t.Errorf("count=3: expected '3 items', got '%s'", got)
	}
}

func TestBundle_ContextAndTranslator(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Context helpers
	if i18n.Locale(nil) != "" {
		t.Errorf("expected empty string for nil context, got '%s'", i18n.Locale(nil))
	}
	ctx := i18n.WithLocale(nil, "id")
	if i18n.LocaleFromContext(ctx) != "id" {
		t.Errorf("expected 'id', got '%s'", i18n.LocaleFromContext(ctx))
	}
	if got := bundle.T(ctx, "auth.login"); got != "Masuk ke Akun" {
		t.Errorf("expected 'Masuk ke Akun', got '%s'", got)
	}

	// Translator instance
	tr := bundle.ForLocale("id")
	if tr.Locale() != "id" {
		t.Errorf("expected 'id', got '%s'", tr.Locale())
	}
	if tr.Bundle() != bundle {
		t.Errorf("expected tr.Bundle() to match original bundle")
	}
	if got := tr.Translate("auth.login"); got != "Masuk ke Akun" {
		t.Errorf("expected 'Masuk ke Akun', got '%s'", got)
	}
	if !tr.Has("auth.login") {
		t.Errorf("expected Has('auth.login') to be true")
	}

	tr.SetLocale("en")
	if got := tr.T("auth.login"); got != "Log in" {
		t.Errorf("expected 'Log in', got '%s'", got)
	}
}

func TestMiddleware(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc := i18n.Locale(r.Context())
		w.Write([]byte(loc))
	})

	mw := i18n.Middleware(bundle,
		i18n.WithQueryParam("locale"),
		i18n.WithCookieName("user_lang"),
		i18n.WithHeaderName("Accept-Language"),
	)
	server := mw(handler)

	// 1. Query parameter
	req1 := httptest.NewRequest("GET", "/?locale=id", nil)
	rr1 := httptest.NewRecorder()
	server.ServeHTTP(rr1, req1)
	if rr1.Body.String() != "id" {
		t.Errorf("query param: expected 'id', got '%s'", rr1.Body.String())
	}

	// 2. Cookie
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(&http.Cookie{Name: "user_lang", Value: "fr"})
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)
	if rr2.Body.String() != "fr" {
		t.Errorf("cookie: expected 'fr', got '%s'", rr2.Body.String())
	}

	// 3. Accept-Language header
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8")
	rr3 := httptest.NewRecorder()
	server.ServeHTTP(rr3, req3)
	if rr3.Body.String() != "id" {
		t.Errorf("header: expected 'id', got '%s'", rr3.Body.String())
	}

	// 4. Default fallback
	req4 := httptest.NewRequest("GET", "/", nil)
	rr4 := httptest.NewRecorder()
	server.ServeHTTP(rr4, req4)
	if rr4.Body.String() != "en" {
		t.Errorf("default: expected 'en', got '%s'", rr4.Body.String())
	}
}

type testConfigSource struct {
	mu   sync.Mutex
	data map[string]string
	ch   chan string
}

func newTestConfigSource() *testConfigSource {
	return &testConfigSource{
		data: make(map[string]string),
		ch:   make(chan string, 10),
	}
}

func (s *testConfigSource) Get(ctx context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return v, nil
}

func (s *testConfigSource) Set(key, val string) {
	s.mu.Lock()
	s.data[key] = val
	s.mu.Unlock()
	select {
	case s.ch <- val:
	default:
	}
}

func (s *testConfigSource) Watch(ctx context.Context, key string) (<-chan string, i18n.CancelFunc, error) {
	return s.ch, func() {}, nil
}

func TestBundle_Concurrency(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	ctx := i18n.WithLocale(context.Background(), "id")

	for i := 0; i < 50; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			_ = bundle.Translate("id", "auth.login")
			_ = bundle.Has("id", "auth.login")
		}()

		go func() {
			defer wg.Done()
			_ = bundle.T(ctx, "auth.login")
			_ = bundle.Translate("en", "messages", "count", 2)
		}()

		go func() {
			defer wg.Done()
			tr := bundle.ForLocale("fr")
			_ = tr.T("welcome", "name", "User")
		}()
	}

	wg.Wait()
}

func TestConfigAndCenter(t *testing.T) {
	fsys := sampleFileSystem()
	cfg := i18n.DefaultConfig()
	cfg.DefaultLocale = "id"
	cfg.FallbackLocale = "en"
	cfg.Locales = []string{"id", "en", "es"}

	bundle, err := i18n.NewFromConfig(fsys, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bundle.DefaultLocale() != "id" {
		t.Errorf("expected default locale 'id', got '%s'", bundle.DefaultLocale())
	}
	if !bundle.HasLocale("es") {
		t.Error("expected locale 'es' to be registered from config")
	}

	// Test dynamic config source integration
	src := newTestConfigSource()
	ctx := context.Background()

	// Store YAML config in source
	yamlData := `
default_locale: fr
fallback_locale: en
locales: [fr, en, de]
`
	src.Set("i18n.settings", yamlData)

	loadedCfg, err := i18n.LoadConfigFromSource(ctx, src, "i18n.settings")
	if err != nil {
		t.Fatalf("failed to load config from source: %v", err)
	}
	if loadedCfg.DefaultLocale != "fr" {
		t.Errorf("expected loaded default locale 'fr', got '%s'", loadedCfg.DefaultLocale)
	}

	// Watch source for hot-reload
	cancel, err := i18n.WatchSource(ctx, src, "i18n.settings", bundle)
	if err != nil {
		t.Fatalf("failed to watch source: %v", err)
	}
	defer cancel()

	// Update in source
	updatedData := `{"default_locale": "de", "locales": ["de", "en"]}`
	src.Set("i18n.settings", updatedData)

	// Give watcher a moment to process change
	for i := 0; i < 20; i++ {
		if bundle.DefaultLocale() == "de" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if bundle.DefaultLocale() != "de" {
		t.Errorf("expected hot-reloaded locale 'de', got '%s'", bundle.DefaultLocale())
	}
}

func TestBundle_LocaleResolutionAndFallback(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys,
		i18n.WithBasePath("lang"),
		i18n.WithDefaultLocale("en"),
		i18n.WithFallbackLocale("en"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating bundle: %v", err)
	}

	if got := bundle.Translate("id", "auth.login"); got != "Masuk ke Akun" {
		t.Errorf("id locale: expected %q, got %q", "Masuk ke Akun", got)
	}

	if got := bundle.Translate("id", "auth.logout"); got != "Log out" {
		t.Errorf("fallback to default en: expected %q, got %q", "Log out", got)
	}

	if got := bundle.Translate("zz", "auth.login"); got != "Log in" {
		t.Errorf("unknown locale fallback: expected %q, got %q", "Log in", got)
	}

	if got := bundle.Translate("en", "auth.login"); got != "Log in" {
		t.Errorf("default locale en: expected %q, got %q", "Log in", got)
	}
}

func TestBundle_HasLocaleAndLocalesList(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys, i18n.WithBasePath("lang"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	locales := bundle.Locales()
	found := map[string]bool{}
	for _, l := range locales {
		found[l] = true
	}
	for _, want := range []string{"en", "id", "fr"} {
		if !found[want] {
			t.Errorf("expected locale %q in Locales(), got %v", want, locales)
		}
	}
	if !bundle.HasLocale("en") || !bundle.HasLocale("id") || !bundle.HasLocale("fr") {
		t.Errorf("HasLocale should report en/id/fr as present")
	}
	if bundle.HasLocale("xx") {
		t.Error("HasLocale should report xx as absent")
	}
}

func TestBundle_LinkRewritingPreservesLocale(t *testing.T) {
	// Locale-specific keys resolve within their own locale; missing keys fall back.
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys,
		i18n.WithBasePath("lang"),
		i18n.WithDefaultLocale("en"),
		i18n.WithFallbackLocale("en"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same key resolves to different values per locale (locale isolation)
	enTitle := bundle.Translate("en", "app_title")
	idTitle := bundle.Translate("id", "app_title")
	if enTitle == idTitle {
		t.Errorf("expected en and id app_title to differ; both got %q", enTitle)
	}

	// Missing key in id falls back to default en (fallback notice behavior)
	got := bundle.Translate("id", "global_msg", "name", "Budi")
	if got != "Hello Budi" {
		t.Errorf("expected fallback Hello Budi, got %q", got)
	}
}
