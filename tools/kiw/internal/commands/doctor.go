package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/krewire/krewire/pkg/errs"
	"github.com/krewire/krewire/pkg/term"
	"github.com/krewire/krewire/tools/kiw/internal/config"
	kiwver "github.com/krewire/krewire/tools/kiw/internal/version"
)

var (
	doctorFlagJSON bool
)

// RegisterDoctor configures flags for the `kiw doctor` command.
func RegisterDoctor(fs *flag.FlagSet) {
	fs.BoolVar(&doctorFlagJSON, "json", false, "output diagnostic report in structured JSON format")
}

// CheckStatus represents the outcome of an individual diagnostic check.
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
)

// DoctorCheck represents a single diagnostic evaluation.
type DoctorCheck struct {
	Category    string      `json:"category"`
	Name        string      `json:"name"`
	Status      CheckStatus `json:"status"`
	Message     string      `json:"message"`
	Remediation string      `json:"remediation,omitempty"`
}

// DoctorReport holds the complete diagnostic evaluation of the environment.
type DoctorReport struct {
	Timestamp   string            `json:"timestamp"`
	Environment map[string]string `json:"environment"`
	Checks      []DoctorCheck     `json:"checks"`
	Passed      bool              `json:"passed"`
	Summary     string            `json:"summary"`
}

// RunDoctor performs system, toolchain, and workspace diagnostics.
func RunDoctor(fs *flag.FlagSet) errs.ExitCode {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	report := DiagnoseEnvironment(cwd)

	if doctorFlagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fail(err)
		}
		if !report.Passed {
			return errs.ExitCodeFailure
		}
		return errs.ExitCodeSuccess
	}

	renderDoctorConsole(report)

	if !report.Passed {
		return errs.ExitCodeFailure
	}
	return errs.ExitCodeSuccess
}

// DiagnoseEnvironment runs all doctor checks and produces a DoctorReport.
func DiagnoseEnvironment(cwd string) DoctorReport {
	report := DoctorReport{
		Environment: map[string]string{
			"os":          runtime.GOOS,
			"arch":        runtime.GOARCH,
			"cpus":        fmt.Sprintf("%d", runtime.NumCPU()),
			"go_runtime":  runtime.Version(),
			"kiw_version": kiwver.VersionString(),
			"working_dir": cwd,
		},
		Passed: true,
	}

	// 1. Go Toolchain Check
	goPath, err := exec.LookPath("go")
	if err != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Category:    "Toolchain",
			Name:        "Go Binary",
			Status:      StatusFail,
			Message:     "Go compiler ('go') not found in PATH",
			Remediation: "Install Go 1.27.1+ from https://go.dev/dl/ and add it to your PATH.",
		})
		report.Passed = false
	} else {
		out, err := exec.Command(goPath, "version").CombinedOutput()
		verStr := strings.TrimSpace(string(out))
		if err != nil {
			report.Checks = append(report.Checks, DoctorCheck{
				Category:    "Toolchain",
				Name:        "Go Binary",
				Status:      StatusFail,
				Message:     fmt.Sprintf("Failed to query Go version: %v", err),
				Remediation: "Verify your Go installation.",
			})
			report.Passed = false
		} else {
			report.Environment["go_compiler"] = verStr
			re := regexp.MustCompile(`go(\d+)\.(\d+)`)
			matches := re.FindStringSubmatch(verStr)
			if len(matches) >= 3 {
				major := matches[1]
				minor := matches[2]
				if major == "1" && (minor < "27") {
					report.Checks = append(report.Checks, DoctorCheck{
						Category:    "Toolchain",
						Name:        "Go Version",
						Status:      StatusWarn,
						Message:     fmt.Sprintf("Found %s; Krewire targets Go 1.27.1+", verStr),
						Remediation: "Upgrade Go to 1.27.1+ for full workspace and module compatibility.",
					})
				} else {
					report.Checks = append(report.Checks, DoctorCheck{
						Category: "Toolchain",
						Name:     "Go Version",
						Status:   StatusPass,
						Message:  fmt.Sprintf("%s (valid toolchain)", verStr),
					})
				}
			} else {
				report.Checks = append(report.Checks, DoctorCheck{
					Category: "Toolchain",
					Name:     "Go Version",
					Status:   StatusPass,
					Message:  verStr,
				})
			}
		}
	}

	// 2. Git Version Control Check
	gitPath, err := exec.LookPath("git")
	if err != nil {
		report.Checks = append(report.Checks, DoctorCheck{
			Category:    "Version Control",
			Name:        "Git Binary",
			Status:      StatusFail,
			Message:     "Git executable not found in PATH",
			Remediation: "Install Git from https://git-scm.com/.",
		})
		report.Passed = false
	} else {
		out, err := exec.Command(gitPath, "version").CombinedOutput()
		gitVer := strings.TrimSpace(string(out))
		if err != nil {
			report.Checks = append(report.Checks, DoctorCheck{
				Category:    "Version Control",
				Name:        "Git Binary",
				Status:      StatusWarn,
				Message:     "Git binary found but failed to report version",
				Remediation: "Check Git executable permissions.",
			})
		} else {
			report.Checks = append(report.Checks, DoctorCheck{
				Category: "Version Control",
				Name:     "Git Binary",
				Status:   StatusPass,
				Message:  gitVer,
			})
		}

		// Git user identity check
		userName, _ := exec.Command(gitPath, "config", "user.name").CombinedOutput()
		userEmail, _ := exec.Command(gitPath, "config", "user.email").CombinedOutput()
		nameStr := strings.TrimSpace(string(userName))
		emailStr := strings.TrimSpace(string(userEmail))

		if nameStr == "" || emailStr == "" {
			report.Checks = append(report.Checks, DoctorCheck{
				Category:    "Version Control",
				Name:        "Git Identity",
				Status:      StatusWarn,
				Message:     "Git user.name or user.email is not set",
				Remediation: "Set global identity: git config --global user.name \"Name\" && git config --global user.email \"email@domain\"",
			})
		} else {
			report.Checks = append(report.Checks, DoctorCheck{
				Category: "Version Control",
				Name:     "Git Identity",
				Status:   StatusPass,
				Message:  fmt.Sprintf("%s <%s>", nameStr, emailStr),
			})
		}
	}

	// 3. Workspace (go.work) Check
	goWorkPath := findGoWork(cwd)
	if goWorkPath != "" {
		content, err := os.ReadFile(goWorkPath)
		if err != nil {
			report.Checks = append(report.Checks, DoctorCheck{
				Category:    "Workspace",
				Name:        "Go Workspace",
				Status:      StatusWarn,
				Message:     fmt.Sprintf("Found go.work at %s but unable to read: %v", goWorkPath, err),
				Remediation: "Verify file permissions on go.work.",
			})
		} else {
			modCount := strings.Count(string(content), "./")
			report.Checks = append(report.Checks, DoctorCheck{
				Category: "Workspace",
				Name:     "Go Workspace",
				Status:   StatusPass,
				Message:  fmt.Sprintf("Active workspace at %s (~%d modules registered)", filepath.Dir(goWorkPath), modCount),
			})
		}
	} else {
		report.Checks = append(report.Checks, DoctorCheck{
			Category: "Workspace",
			Name:     "Go Workspace",
			Status:   StatusPass,
			Message:  "Standalone module mode (no parent go.work)",
		})
	}

	// 4. Project Configuration (krewire.yaml) Check
	cfgPath := filepath.Join(cwd, "krewire.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, err := config.Load(cwd)
		if err != nil {
			report.Checks = append(report.Checks, DoctorCheck{
				Category:    "Project",
				Name:        "Configuration",
				Status:      StatusFail,
				Message:     fmt.Sprintf("krewire.yaml has syntax/schema errors: %v", err),
				Remediation: "Review and fix YAML syntax in krewire.yaml.",
			})
			report.Passed = false
		} else {
			pName := cfg.Project.Name
			if pName == "" {
				pName = filepath.Base(cwd)
			}
			report.Checks = append(report.Checks, DoctorCheck{
				Category: "Project",
				Name:     "Configuration",
				Status:   StatusPass,
				Message:  fmt.Sprintf("Valid krewire.yaml (project: %q, kind: %s)", pName, cfg.Project.Kind),
			})
		}
	} else {
		report.Checks = append(report.Checks, DoctorCheck{
			Category: "Project",
			Name:     "Configuration",
			Status:   StatusPass,
			Message:  "No krewire.yaml in current directory (run 'kiw init' or 'kiw new' to start one)",
		})
	}

	// Calculate summary
	passCount, warnCount, failCount := 0, 0, 0
	for _, c := range report.Checks {
		switch c.Status {
		case StatusPass:
			passCount++
		case StatusWarn:
			warnCount++
		case StatusFail:
			failCount++
		}
	}

	if failCount > 0 {
		report.Passed = false
		report.Summary = fmt.Sprintf("%d check(s) failed, %d warning(s), %d passed", failCount, warnCount, passCount)
	} else if warnCount > 0 {
		report.Summary = fmt.Sprintf("All required checks passed with %d warning(s), %d passed", warnCount, passCount)
	} else {
		report.Summary = fmt.Sprintf("All %d checks passed cleanly", passCount)
	}

	return report
}

func findGoWork(startDir string) string {
	dir := startDir
	for {
		candidate := filepath.Join(dir, "go.work")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func renderDoctorConsole(report DoctorReport) {
	tm := term.NewTerminal()
	bold := func(s string) string { return tm.Paint(s, term.ColorCyan, []term.Style{term.StyleBold}) }
	green := func(s string) string { return tm.Paint(s, term.ColorGreen, []term.Style{term.StyleBold}) }
	yellow := func(s string) string { return tm.Paint(s, term.ColorYellow, []term.Style{term.StyleBold}) }
	red := func(s string) string { return tm.Paint(s, term.ColorRed, []term.Style{term.StyleBold}) }
	dim := func(s string) string { return tm.Paint(s, term.ColorDefault, []term.Style{term.StyleDim}) }

	fmt.Println()
	fmt.Printf("%s\n", bold("Krewire Environment Doctor"))
	fmt.Println(dim(strings.Repeat("─", 50)))
	fmt.Printf("OS/Arch:    %s/%s (%s CPUs)\n", report.Environment["os"], report.Environment["arch"], report.Environment["cpus"])
	fmt.Printf("Kiw CLI:    %s\n", report.Environment["kiw_version"])
	if cr, ok := report.Environment["go_compiler"]; ok {
		fmt.Printf("Go runtime: %s\n", cr)
	}
	fmt.Println(dim(strings.Repeat("─", 50)))
	fmt.Println()

	for _, c := range report.Checks {
		var badge string
		switch c.Status {
		case StatusPass:
			badge = green("[ PASS ]")
		case StatusWarn:
			badge = yellow("[ WARN ]")
		case StatusFail:
			badge = red("[ FAIL ]")
		}

		fmt.Printf("%s  %s: %s\n", badge, bold(c.Name), c.Message)
		if c.Remediation != "" {
			fmt.Printf("          %s %s\n", dim("↳ Fix:"), c.Remediation)
		}
	}

	fmt.Println()
	fmt.Println(dim(strings.Repeat("─", 50)))
	if report.Passed {
		fmt.Printf("%s %s\n", green("Doctor Result:"), report.Summary)
	} else {
		fmt.Printf("%s %s\n", red("Doctor Result:"), report.Summary)
	}
	fmt.Println()
}
