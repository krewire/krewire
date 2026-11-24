package test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// FileCoverage represents statement coverage metrics for a single source file.
type FileCoverage struct {
	FileName          string `json:"fileName"`
	PackagePath       string `json:"packagePath"`
	TotalStatements   int    `json:"totalStatements"`
	CoveredStatements int    `json:"coveredStatements"`
}

// Percentage returns the covered statements percentage (0.0 - 100.0).
func (f FileCoverage) Percentage() float64 {
	if f.TotalStatements == 0 {
		return 100.0
	}
	return (float64(f.CoveredStatements) / float64(f.TotalStatements)) * 100.0
}

// PackageCoverage represents statement coverage metrics for a package.
type PackageCoverage struct {
	PackagePath       string         `json:"packagePath"`
	Files             []FileCoverage `json:"files"`
	TotalStatements   int            `json:"totalStatements"`
	CoveredStatements int            `json:"coveredStatements"`
}

// Percentage returns the covered statements percentage for the package.
func (p PackageCoverage) Percentage() float64 {
	if p.TotalStatements == 0 {
		return 100.0
	}
	return (float64(p.CoveredStatements) / float64(p.TotalStatements)) * 100.0
}

// CoverageReport represents an aggregated coverage report across all files and packages.
type CoverageReport struct {
	Mode              string            `json:"mode"`
	Packages          []PackageCoverage `json:"packages"`
	Files             []FileCoverage    `json:"files"`
	TotalStatements   int               `json:"totalStatements"`
	CoveredStatements int               `json:"coveredStatements"`
}

// Percentage returns the overall covered statements percentage across all packages.
func (r CoverageReport) Percentage() float64 {
	if r.TotalStatements == 0 {
		return 100.0
	}
	return (float64(r.CoveredStatements) / float64(r.TotalStatements)) * 100.0
}

// MeetsThreshold checks if the total coverage percentage meets or exceeds minPercentage.
func (r CoverageReport) MeetsThreshold(minPercentage float64) bool {
	return r.Percentage() >= minPercentage
}

// ParseCoverageProfile parses Go coverage profile data from an io.Reader.
func ParseCoverageProfile(r io.Reader) (*CoverageReport, error) {
	scanner := bufio.NewScanner(r)
	report := &CoverageReport{}

	fileMap := make(map[string]*FileCoverage)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "mode:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				report.Mode = strings.TrimSpace(parts[1])
			}
			continue
		}

		// Line format: <filename>:<startLine>.<startCol>,<endLine>.<endCol> <numStmt> <count>
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}

		target := fields[0]
		colonIdx := strings.LastIndex(target, ":")
		if colonIdx == -1 {
			continue
		}
		coords := target[colonIdx+1:]
		if !strings.Contains(coords, ".") || !strings.Contains(coords, ",") {
			continue
		}
		filename := target[:colonIdx]

		numStmt, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		fc, exists := fileMap[filename]
		if !exists {
			pkgPath := path.Dir(filename)
			fc = &FileCoverage{
				FileName:    filename,
				PackagePath: pkgPath,
			}
			fileMap[filename] = fc
		}

		fc.TotalStatements += numStmt
		if count > 0 {
			fc.CoveredStatements += numStmt
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}

	// Group into packages
	pkgMap := make(map[string]*PackageCoverage)
	for _, fc := range fileMap {
		report.Files = append(report.Files, *fc)
		report.TotalStatements += fc.TotalStatements
		report.CoveredStatements += fc.CoveredStatements

		pc, exists := pkgMap[fc.PackagePath]
		if !exists {
			pc = &PackageCoverage{
				PackagePath: fc.PackagePath,
			}
			pkgMap[fc.PackagePath] = pc
		}
		pc.Files = append(pc.Files, *fc)
		pc.TotalStatements += fc.TotalStatements
		pc.CoveredStatements += fc.CoveredStatements
	}

	for _, pc := range pkgMap {
		sort.Slice(pc.Files, func(i, j int) bool {
			return pc.Files[i].FileName < pc.Files[j].FileName
		})
		report.Packages = append(report.Packages, *pc)
	}

	sort.Slice(report.Packages, func(i, j int) bool {
		return report.Packages[i].PackagePath < report.Packages[j].PackagePath
	})
	sort.Slice(report.Files, func(i, j int) bool {
		return report.Files[i].FileName < report.Files[j].FileName
	})

	return report, nil
}

// ParseCoverageProfileFile reads and parses a coverage profile file from disk.
func ParseCoverageProfileFile(filePath string) (*CoverageReport, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open coverage profile %q: %w", filePath, err)
	}
	defer f.Close()

	return ParseCoverageProfile(f)
}

// FormatCoverageSummary returns a concise one-line summary of code coverage.
func FormatCoverageSummary(report *CoverageReport) string {
	if report == nil || report.TotalStatements == 0 {
		return "coverage: 0 statements (0.0%)"
	}
	return fmt.Sprintf("coverage: %.1f%% of statements (%d/%d)", report.Percentage(), report.CoveredStatements, report.TotalStatements)
}

// FormatCoverageTable renders a structured terminal breakdown of coverage.
func FormatCoverageTable(report *CoverageReport) string {
	if report == nil || len(report.Files) == 0 {
		return "No coverage data available.\n"
	}

	var sb strings.Builder
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%-50s %10s %10s %8s\n", "Package / File", "Statements", "Covered", "Coverage"))
	sb.WriteString("--------------------------------------------------------------------------------\n")

	for _, pkg := range report.Packages {
		sb.WriteString(fmt.Sprintf("%-50s %10d %10d %7.1f%%\n", pkg.PackagePath, pkg.TotalStatements, pkg.CoveredStatements, pkg.Percentage()))
		for _, file := range pkg.Files {
			baseName := "  " + path.Base(file.FileName)
			sb.WriteString(fmt.Sprintf("%-50s %10d %10d %7.1f%%\n", baseName, file.TotalStatements, file.CoveredStatements, file.Percentage()))
		}
	}

	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%-50s %10d %10d %7.1f%%\n", "Total", report.TotalStatements, report.CoveredStatements, report.Percentage()))
	sb.WriteString("--------------------------------------------------------------------------------\n")

	return sb.String()
}
