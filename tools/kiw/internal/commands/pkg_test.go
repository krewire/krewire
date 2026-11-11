package commands

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewire/krewire/pkg/errs"
	kiwdsl "github.com/krewire/krewire/pkg/kiw"
)

func TestPkgCommandLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kiw-cmd-pkg-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)
	_ = os.Chdir(tempDir)

	// 1. Test 'kiw pkg init my-ui'
	fsInit := flag.NewFlagSet("pkg", flag.ContinueOnError)
	RegisterPkg(fsInit)
	if err := fsInit.Parse([]string{"init", "my-ui"}); err != nil {
		t.Fatal(err)
	}
	if code := RunPkg(fsInit); code != errs.ExitCodeSuccess {
		t.Fatalf("kiw pkg init failed with code %v", code)
	}

	manifestPath := filepath.Join(tempDir, "my-ui", kiwdsl.PackageManifestFile)
	manifest, err := kiwdsl.LoadPackageManifest(manifestPath)
	if err != nil {
		t.Fatalf("load created manifest: %v", err)
	}
	if manifest.Name != "@components/my-ui" {
		t.Errorf("manifest name = %q want @components/my-ui", manifest.Name)
	}

	// 2. Test 'kiw pkg pack my-ui'
	fsPack := flag.NewFlagSet("pkg", flag.ContinueOnError)
	RegisterPkg(fsPack)
	if err := fsPack.Parse([]string{"pack", "my-ui"}); err != nil {
		t.Fatal(err)
	}
	if code := RunPkg(fsPack); code != errs.ExitCodeSuccess {
		t.Fatalf("kiw pkg pack failed with code %v", code)
	}

	archiveName := "components-my-ui-0.1.0.kpkg"
	archivePath := filepath.Join(tempDir, archiveName)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("expected packed archive %s to exist: %v", archivePath, err)
	}

	// 3. Test 'kiw pkg add <archive>'
	projectDir := filepath.Join(tempDir, "my-app")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chdir(projectDir)

	fsAdd := flag.NewFlagSet("pkg", flag.ContinueOnError)
	RegisterPkg(fsAdd)
	if err := fsAdd.Parse([]string{"add", archivePath}); err != nil {
		t.Fatal(err)
	}
	if code := RunPkg(fsAdd); code != errs.ExitCodeSuccess {
		t.Fatalf("kiw pkg add failed with code %v", code)
	}

	installedManifestPath := filepath.Join(projectDir, ".krewire", "pkg", "components-my-ui", kiwdsl.PackageManifestFile)
	if _, err := os.Stat(installedManifestPath); err != nil {
		t.Fatalf("expected installed manifest at %s: %v", installedManifestPath, err)
	}

	// 4. Test 'kiw pkg list'
	fsList := flag.NewFlagSet("pkg", flag.ContinueOnError)
	RegisterPkg(fsList)
	if err := fsList.Parse([]string{"list"}); err != nil {
		t.Fatal(err)
	}
	if code := RunPkg(fsList); code != errs.ExitCodeSuccess {
		t.Fatalf("kiw pkg list failed with code %v", code)
	}
}
