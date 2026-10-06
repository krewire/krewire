package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/packages/kern"
)

// isolateRegistry swaps the global registry for the duration of a test so
// Register calls in one test cannot leak into another.
func isolateRegistry(t *testing.T) {
	t.Helper()
	saved := Registry
	Registry = nil
	t.Cleanup(func() { Registry = saved })
}

// stubPlugin is a minimal Plugin used to exercise the lookup helpers.
type stubPlugin struct {
	name    string
	aliases []string
	version kern.Version
}

func (s *stubPlugin) Name() string               { return s.name }
func (s *stubPlugin) Aliases() []string          { return s.aliases }
func (s *stubPlugin) Detect(string) bool         { return true }
func (s *stubPlugin) Build(string, string) error { return nil }
func (s *stubPlugin) Version() kern.Version      { return s.version }

// stubInstaller additionally satisfies the Installer contract.
type stubInstaller struct{ stubPlugin }

func (s *stubInstaller) Add(string, string) error { return nil }
func (s *stubInstaller) Remove(string) error      { return nil }

// TestPLUGIN_001_FindMatchesNameAndAliases verifies lookup is case-insensitive,
// trims surrounding space, and matches both the plugin name and its aliases.
func TestPLUGIN_001_FindMatchesNameAndAliases(t *testing.T) {
	isolateRegistry(t)
	Register(&stubPlugin{name: "tailwind", aliases: []string{"twcss"}, version: kern.MustParseVersion("1.2.3")})

	for _, query := range []string{
		"tailwind", "TAILWIND", "  tailwind  ", "twcss", "TWCSS",
	} {
		if Find(query) == nil {
			t.Errorf("Find(%q) = nil, want the stub plugin", query)
		}
	}
	if Find("nope") != nil {
		t.Error("Find must return nil for an unknown name")
	}
}

// TestPLUGIN_002_FindMapsTailwindcss verifies the npm package name resolves to
// the Tailwind plugin even when it is not declared as an alias.
func TestPLUGIN_002_FindMapsTailwindcss(t *testing.T) {
	isolateRegistry(t)
	Register(&Tailwind{})
	if Find("tailwindcss") == nil {
		t.Error("the npm name tailwindcss must resolve to the tailwind plugin")
	}
}

// TestPLUGIN_003_FindInstaller verifies the Installer narrowing only succeeds
// for plugins that implement the optional contract.
func TestPLUGIN_003_FindInstaller(t *testing.T) {
	isolateRegistry(t)

	Register(&stubPlugin{name: "plain", version: kern.MustParseVersion("1.0.0")})
	if FindInstaller("plain") != nil {
		t.Error("a plugin without Add/Remove must not satisfy Installer")
	}

	Register(&stubInstaller{stubPlugin{name: "managed", version: kern.MustParseVersion("1.0.0")}})
	if FindInstaller("managed") == nil {
		t.Error("a plugin implementing Installer must be returned")
	}
	if FindInstaller("absent") != nil {
		t.Error("FindInstaller must return nil for an unknown name")
	}
}

// TestPLUGIN_004_FindByVersion verifies the version filter rejects a plugin that
// does not satisfy the requirement.
func TestPLUGIN_004_FindByVersion(t *testing.T) {
	isolateRegistry(t)
	Register(&stubPlugin{name: "tw", version: kern.MustParseVersion("1.2.0")})

	if FindByVersion("tw", kern.MustParseVersion("1.0.0")) == nil {
		t.Error("a compatible version must be returned")
	}
	if FindByVersion("tw", kern.MustParseVersion("9.0.0")) != nil {
		t.Error("an incompatible version must be filtered out")
	}
	if FindByVersion("absent", kern.MustParseVersion("1.0.0")) != nil {
		t.Error("an unknown name must return nil")
	}
}

// TestPLUGIN_005_FindInstallerByVersion verifies both the name and version
// filters apply to the Installer lookup.
func TestPLUGIN_005_FindInstallerByVersion(t *testing.T) {
	isolateRegistry(t)
	Register(&stubInstaller{stubPlugin{name: "tw", version: kern.MustParseVersion("1.2.0")}})

	if FindInstallerByVersion("tw", kern.MustParseVersion("1.0.0")) == nil {
		t.Error("a compatible installer must be returned")
	}
	if FindInstallerByVersion("tw", kern.MustParseVersion("9.0.0")) != nil {
		t.Error("an incompatible installer must be filtered out")
	}
	if FindInstallerByVersion("absent", kern.MustParseVersion("1.0.0")) != nil {
		t.Error("an unknown name must return nil")
	}
}

// TestPLUGIN_006_TailwindMetadata verifies the shipped plugin's identity,
// aliases, version, and interface satisfaction.
func TestPLUGIN_006_TailwindMetadata(t *testing.T) {
	tw := &Tailwind{}
	if tw.Name() != "tailwind" {
		t.Errorf("Name = %q", tw.Name())
	}
	if got := tw.Aliases(); len(got) != 2 || got[0] != "twcss" {
		t.Errorf("Aliases = %v", got)
	}
	if tw.Version().Major != 3 {
		t.Errorf("Version = %+v", tw.Version())
	}
	// The plugin must satisfy the optional contracts it claims to implement.
	var _ Plugin = tw
	var _ Installer = tw
	var _ AssetProvider = tw
}

// TestPLUGIN_007_TailwindDetect verifies config-file presence drives detection
// across every supported config extension.
func TestPLUGIN_007_TailwindDetect(t *testing.T) {
	tw := &Tailwind{}
	if tw.Detect(t.TempDir()) {
		t.Error("Detect must be false without a config file")
	}
	for _, name := range tailwindConfigFiles {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("module.exports={}"), 0o600); err != nil {
			t.Fatal(err)
		}
		if !tw.Detect(dir) {
			t.Errorf("Detect must be true when %s exists", name)
		}
	}
}

// TestPLUGIN_008_TailwindAssets verifies the declared asset appears only when the
// input stylesheet exists, so kiw build does not link a missing file.
func TestPLUGIN_008_TailwindAssets(t *testing.T) {
	tw := &Tailwind{}
	root := t.TempDir()
	if got := tw.Assets(root); got != nil {
		t.Errorf("Assets = %v, want nil without the input file", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "tailwind.css"), []byte("@tailwind base;"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := tw.Assets(root)
	if len(got) != 1 || got[0] != "assets/tailwind.css" {
		t.Errorf("Assets = %v", got)
	}
}

// TestPLUGIN_009_TailwindBuildSkipsWithoutInput verifies a missing input stylesheet
// is a no-op rather than a failure, since Tailwind is an optional plugin.
func TestPLUGIN_009_TailwindBuildSkipsWithoutInput(t *testing.T) {
	tw := &Tailwind{}
	if err := tw.Build(t.TempDir(), t.TempDir()); err != nil {
		t.Errorf("Build without input must succeed as a no-op, got %v", err)
	}
}

// TestPLUGIN_010_EnsureConfigIsIdempotent verifies Add's helpers create the
// config and input files once and never overwrite an existing one.
func TestPLUGIN_010_EnsureConfigIsIdempotent(t *testing.T) {
	root := t.TempDir()

	if err := ensureTailwindConfig(root); err != nil {
		t.Fatalf("ensureTailwindConfig: %v", err)
	}
	cfgPath := filepath.Join(root, tailwindConfigFile)
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config was not created: %v", err)
	}

	// A second call must leave an existing config untouched.
	if err := os.WriteFile(cfgPath, []byte("// customized"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureTailwindConfig(root); err != nil {
		t.Fatalf("ensureTailwindConfig (second): %v", err)
	}
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "// customized" {
		t.Errorf("ensureTailwindConfig overwrote an existing config: %q", after)
	}

	if err := ensureTailwindInput(root); err != nil {
		t.Fatalf("ensureTailwindInput: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "assets", "tailwind.css")); err != nil {
		t.Fatalf("input was not created: %v", err)
	}
	if err := ensureTailwindInput(root); err != nil {
		t.Fatalf("ensureTailwindInput (second): %v", err)
	}
}

// TestPLUGIN_011_RegistrySelfRegistration verifies the Tailwind plugin is present
// in the package registry without any explicit Register call, since init() does it.
func TestPLUGIN_011_RegistrySelfRegistration(t *testing.T) {
	if Find("tailwind") == nil {
		t.Error("the Tailwind plugin must self-register via init()")
	}
	if FindInstaller("twcss") == nil {
		t.Error("Tailwind must be reachable as an Installer through its alias")
	}
}
