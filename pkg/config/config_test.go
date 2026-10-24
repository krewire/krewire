package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type AppConfig struct {
	Title   string     `yaml:"title" env:"TITLE"`
	Port    int        `yaml:"port"`
	Debug   bool       `yaml:"debug"`
	Tags    []string   `yaml:"tags"`
	Server  ServerCfg  `yaml:"server"`
	Ignored *Threshold `yaml:"threshold"`
}

type ServerCfg struct {
	Addr string `yaml:"addr"`
	Host string `env:"HOST"`
}

type Threshold struct {
	Rate float64 `yaml:"rate"`
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestLoadFile(t *testing.T) {
	path := writeFile(t, "config.yaml", "title: Hello\nport: 8080\ndebug: true\ntags: [a, b]\nserver:\n  addr: :9000\n")
	var got AppConfig
	if err := Load(path, &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Title != "Hello" {
		t.Errorf("Title = %q, want Hello", got.Title)
	}
	if got.Port != 8080 {
		t.Errorf("Port = %d, want 8080", got.Port)
	}
	if !got.Debug {
		t.Error("Debug = false, want true")
	}
	if len(got.Tags) != 2 || got.Tags[1] != "b" {
		t.Errorf("Tags = %v", got.Tags)
	}
	if got.Server.Addr != ":9000" {
		t.Errorf("Server.Addr = %q, want :9000", got.Server.Addr)
	}
}

func TestLoadMissingFileIsZero(t *testing.T) {
	var got AppConfig
	if err := Load(filepath.Join(t.TempDir(), "nope.yaml"), &got); err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if !reflect.DeepEqual(got, AppConfig{}) {
		t.Errorf("got %+v, want zero value", got)
	}
}

func TestLoadMalformedNamingPath(t *testing.T) {
	path := writeFile(t, "bad.yaml", "port: [\n")
	var got AppConfig
	err := Load(path, &got)
	if err == nil {
		t.Fatal("Load malformed: expected error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name path", err)
	}
}

func TestLoadErrorsTarget(t *testing.T) {
	var got AppConfig
	if err := Load("x.yaml", nil); err == nil {
		t.Error("nil dst: expected error, got nil")
	}
	if err := Load("x.yaml", got); err == nil {
		t.Error("non-pointer dst: expected error, got nil")
	}
	var ptr *AppConfig
	if err := Load("x.yaml", ptr); err == nil {
		t.Error("nil pointer dst: expected error, got nil")
	}
	if err := Load("", &got); err == nil {
		t.Error("empty path: expected error, got nil")
	}
}

func TestOverrideExplicitEnvTag(t *testing.T) {
	var got AppConfig
	if err := Load("missing.yaml", &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	lookup := func(key string) (string, bool) {
		switch key {
		case "TITLE":
			return "From Env", true
		case "HOST":
			return "10.0.0.1", true
		}
		return "", false
	}
	if err := Override(&got, lookup); err != nil {
		t.Fatalf("Override: %v", err)
	}
	if got.Title != "From Env" {
		t.Errorf("Title = %q, want From Env", got.Title)
	}
	if got.Server.Host != "10.0.0.1" {
		t.Errorf("Server.Host = %q, want 10.0.0.1", got.Server.Host)
	}
}

func TestOverrideImplicitKeyAndNesting(t *testing.T) {
	var got AppConfig
	lookup := func(key string) (string, bool) {
		if key == "APP_SERVER_ADDR" {
			return ":7000", true
		}
		if key == "APP_PORT" {
			return "6060", true
		}
		return "", false
	}
	if err := Override(&got, lookup); err != nil {
		t.Fatalf("Override: %v", err)
	}
	if got.Server.Addr != ":7000" {
		t.Errorf("Server.Addr = %q, want :7000 (implicit APP_SERVER_ADDR)", got.Server.Addr)
	}
	if got.Port != 6060 {
		t.Errorf("Port = %d, want 6060 (implicit APP_PORT)", got.Port)
	}
}

func TestOverrideWithPrefix(t *testing.T) {
	var got AppConfig
	got.Server.Addr = ":9999"
	lookup := func(key string) (string, bool) {
		if key == "MYAPP_SERVER_ADDR" {
			return ":5000", true
		}
		return "", false
	}
	if err := Override(&got, lookup, WithPrefix("MYAPP")); err != nil {
		t.Fatalf("Override: %v", err)
	}
	if got.Server.Addr != ":5000" {
		t.Errorf("Server.Addr = %q, want :5000 (MyApp prefix)", got.Server.Addr)
	}
}

func TestPrecedenceZeroFileEnv(t *testing.T) {
	path := writeFile(t, "p.yaml", "title: From File\nport: 1000\n")
	var got AppConfig
	if err := Load(path, &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	lookup := func(key string) (string, bool) {
		if key == "TITLE" {
			return "From Env", true
		}
		return "", false
	}
	if err := Override(&got, lookup); err != nil {
		t.Fatalf("Override: %v", err)
	}
	if got.Title != "From Env" {
		t.Errorf("env should win: Title = %q", got.Title)
	}
	if got.Port != 1000 {
		t.Errorf("file should beat zero: Port = %d, want 1000", got.Port)
	}
}

func TestOverrideEmptyEnvIsUnset(t *testing.T) {
	var got AppConfig
	lookup := func(key string) (string, bool) {
		if key == "APP_TITLE" {
			return "", true
		}
		return "", false
	}
	if err := Override(&got, lookup); err != nil {
		t.Fatalf("Override: %v", err)
	}
	if got.Title != "" {
		t.Errorf("empty env should stay unset, got %q", got.Title)
	}
}

func TestOverrideSlicesNotOverlaid(t *testing.T) {
	var got AppConfig
	got.Tags = []string{"keep"}
	lookup := func(key string) (string, bool) {
		if key == "APP_TAGS" {
			return "[x]", true
		}
		return "", false
	}
	if err := Override(&got, lookup); err != nil {
		t.Fatalf("Override: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "keep" {
		t.Errorf("slice must not be overlaid, got %v", got.Tags)
	}
}

func TestOverrideDecodeErrorNamesFieldAndKey(t *testing.T) {
	var got AppConfig
	lookup := func(key string) (string, bool) {
		if key == "APP_PORT" {
			return "not-a-number", true
		}
		return "", false
	}
	err := Override(&got, lookup)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "port") || !strings.Contains(err.Error(), "APP_PORT") {
		t.Errorf("error %q must name field and key", err)
	}
}

func TestOverrideErrorsTarget(t *testing.T) {
	if err := Override(nil, func(string) (string, bool) { return "", false }); err == nil {
		t.Error("nil dst: expected error")
	}
	var got string
	if err := Override(&got, func(string) (string, bool) { return "", false }); err == nil {
		t.Error("non-struct dst: expected error")
	}
}

func TestLoadOrDefault(t *testing.T) {
	var got AppConfig
	if err := LoadOrDefault(filepath.Join(t.TempDir(), "absent.yaml"), &got); err != nil {
		t.Fatalf("LoadOrDefault missing: %v", err)
	}
	if !reflect.DeepEqual(got, AppConfig{}) {
		t.Errorf("got %+v, want zero", got)
	}
}
