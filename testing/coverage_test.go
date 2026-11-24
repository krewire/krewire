package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleCoverProfile = `mode: set
github.com/krewire/krewire/auth/auth.go:10.15,12.2 2 1
github.com/krewire/krewire/auth/auth.go:14.2,16.2 3 0
github.com/krewire/krewire/auth/client.go:20.10,25.2 5 1
github.com/krewire/krewire/web/router.go:30.12,35.2 10 2
`

func TestParseCoverageProfile(t *testing.T) {
	report, err := ParseCoverageProfile(strings.NewReader(sampleCoverProfile))
	if err != nil {
		t.Fatalf("unexpected error parsing profile: %v", err)
	}

	if report.Mode != "set" {
		t.Errorf("got mode %q, want %q", report.Mode, "set")
	}

	// Total statements: 2 + 3 + 5 + 10 = 20
	// Covered statements: 2 + 0 + 5 + 10 = 17
	if report.TotalStatements != 20 {
		t.Errorf("got TotalStatements %d, want 20", report.TotalStatements)
	}
	if report.CoveredStatements != 17 {
		t.Errorf("got CoveredStatements %d, want 17", report.CoveredStatements)
	}

	// Percentage: 17/20 = 85.0%
	if pct := report.Percentage(); pct != 85.0 {
		t.Errorf("got Percentage %f, want 85.0", pct)
	}

	if !report.MeetsThreshold(80.0) {
		t.Error("expected 85% to meet 80% threshold")
	}
	if report.MeetsThreshold(90.0) {
		t.Error("expected 85% not to meet 90% threshold")
	}

	if len(report.Packages) != 2 {
		t.Fatalf("got %d packages, want 2", len(report.Packages))
	}

	// Verify package auth
	authPkg := report.Packages[0]
	if authPkg.PackagePath != "github.com/krewire/krewire/auth" {
		t.Errorf("got package path %q", authPkg.PackagePath)
	}
	if authPkg.TotalStatements != 10 {
		t.Errorf("auth pkg: got TotalStatements %d, want 10", authPkg.TotalStatements)
	}
	if authPkg.CoveredStatements != 7 {
		t.Errorf("auth pkg: got CoveredStatements %d, want 7", authPkg.CoveredStatements)
	}
	if pct := authPkg.Percentage(); pct != 70.0 {
		t.Errorf("auth pkg: got percentage %f, want 70.0", pct)
	}
}

func TestParseCoverageProfileFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "coverage.out")
	if err := os.WriteFile(path, []byte(sampleCoverProfile), 0o644); err != nil {
		t.Fatalf("failed to write coverage file: %v", err)
	}

	report, err := ParseCoverageProfileFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Percentage() != 85.0 {
		t.Errorf("got percentage %f, want 85.0", report.Percentage())
	}
}

func TestFormatCoverageSummary(t *testing.T) {
	report, _ := ParseCoverageProfile(strings.NewReader(sampleCoverProfile))
	summary := FormatCoverageSummary(report)
	if !strings.Contains(summary, "85.0%") {
		t.Errorf("summary %q does not contain 85.0%%", summary)
	}
	if !strings.Contains(summary, "17/20") {
		t.Errorf("summary %q does not contain 17/20", summary)
	}

	emptySummary := FormatCoverageSummary(nil)
	if !strings.Contains(emptySummary, "0.0%") {
		t.Errorf("empty summary %q does not contain 0.0%%", emptySummary)
	}
}

func TestFormatCoverageTable(t *testing.T) {
	report, _ := ParseCoverageProfile(strings.NewReader(sampleCoverProfile))
	table := FormatCoverageTable(report)

	if !strings.Contains(table, "Package / File") {
		t.Errorf("table missing header: %s", table)
	}
	if !strings.Contains(table, "github.com/krewire/krewire/auth") {
		t.Errorf("table missing auth package: %s", table)
	}
	if !strings.Contains(table, "router.go") {
		t.Errorf("table missing router.go: %s", table)
	}
	if !strings.Contains(table, "85.0%") {
		t.Errorf("table missing total percentage: %s", table)
	}
}

func TestParseCoverageProfile_EmptyAndEdgeCases(t *testing.T) {
	emptyReport, err := ParseCoverageProfile(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if emptyReport.Percentage() != 100.0 {
		t.Errorf("expected 100%% for 0 statements, got %f", emptyReport.Percentage())
	}

	// Bad lines should be gracefully skipped
	badProfile := `mode: set
invalid line without fields
foo.go:invalid 1 1
bar.go:1.1,2.2 notANumber 1
baz.go:1.1,2.2 5 notANumber
valid.go:1.1,2.2 10 1
`
	report, err := ParseCoverageProfile(strings.NewReader(badProfile))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalStatements != 10 {
		t.Errorf("got TotalStatements %d, want 10", report.TotalStatements)
	}
}
