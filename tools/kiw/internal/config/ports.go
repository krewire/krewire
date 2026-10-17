package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// DetectedPort records a port declared by a specific runtime (FRK-CONF-030).
type DetectedPort struct {
	Runtime string // "krewire", "nodejs", "laravel", "env"
	Source  string // "krewire.yaml", "package.json", ".env", "composer.json"
	Port    int
	Detail  string
}

var (
	portFlagRegex = regexp.MustCompile(`(?:--port(?:=|\s+)|-p\s+|PORT=)(\d{2,5})`)
	envPortRegex  = regexp.MustCompile(`(?m)^(?:APP_PORT|SERVER_PORT|PORT|VITE_PORT)=(\d{2,5})`)
)

// DetectRuntimePorts scans the project root and krewire.yaml to discover declared
// ports across runtimes and flags potential conflicts (FRK-CONF-003, FRK-CONF-030).
func DetectRuntimePorts(root string, cfg *Config) ([]DetectedPort, []string) {
	var detected []DetectedPort

	// 1. Krewire declared ports
	if cfg != nil {
		if cfg.Runtime.Port > 0 {
			detected = append(detected, DetectedPort{
				Runtime: "krewire",
				Source:  "krewire.yaml",
				Port:    cfg.Runtime.Port,
				Detail:  "runtime.port",
			})
		}
		if cfg.Dev.Port > 0 && cfg.Dev.Port != cfg.Runtime.Port {
			detected = append(detected, DetectedPort{
				Runtime: "krewire",
				Source:  "krewire.yaml",
				Port:    cfg.Dev.Port,
				Detail:  "dev.port",
			})
		}
		if cfg.Server.Port > 0 && cfg.Server.Port != cfg.Runtime.Port && cfg.Server.Port != cfg.Dev.Port {
			detected = append(detected, DetectedPort{
				Runtime: "krewire",
				Source:  "krewire.yaml",
				Port:    cfg.Server.Port,
				Detail:  "server.port",
			})
		}
		for _, svc := range cfg.Services {
			if svc.Port > 0 {
				detected = append(detected, DetectedPort{
					Runtime: "krewire",
					Source:  "krewire.yaml",
					Port:    svc.Port,
					Detail:  "service: " + svc.Name,
				})
			}
		}
	}

	// 2. Node.js (package.json)
	pkgPath := filepath.Join(root, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Port    any               `json:"port"`
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			if p, ok := pkg.Port.(float64); ok && p > 0 && p <= 65535 {
				detected = append(detected, DetectedPort{
					Runtime: "nodejs",
					Source:  "package.json",
					Port:    int(p),
					Detail:  "port config",
				})
			}
			for scriptName, cmd := range pkg.Scripts {
				matches := portFlagRegex.FindStringSubmatch(cmd)
				if len(matches) == 2 {
					if portNum, err := strconv.Atoi(matches[1]); err == nil && portNum > 0 && portNum <= 65535 {
						detected = append(detected, DetectedPort{
							Runtime: "nodejs",
							Source:  "package.json",
							Port:    portNum,
							Detail:  fmt.Sprintf("scripts.%s: %q", scriptName, cmd),
						})
					}
				}
			}
		}
	}

	// 3. PHP / Laravel (composer.json)
	composerPath := filepath.Join(root, "composer.json")
	if data, err := os.ReadFile(composerPath); err == nil {
		var comp struct {
			Scripts map[string]any `json:"scripts"`
		}
		if json.Unmarshal(data, &comp) == nil {
			for scriptName, val := range comp.Scripts {
				if cmd, ok := val.(string); ok {
					matches := portFlagRegex.FindStringSubmatch(cmd)
					if len(matches) == 2 {
						if portNum, err := strconv.Atoi(matches[1]); err == nil && portNum > 0 && portNum <= 65535 {
							detected = append(detected, DetectedPort{
								Runtime: "laravel",
								Source:  "composer.json",
								Port:    portNum,
								Detail:  fmt.Sprintf("scripts.%s", scriptName),
							})
						}
					}
				}
			}
		}
	}

	// 4. Dotenv (.env and .env.example)
	for _, envFileName := range []string{".env", ".env.example"} {
		envPath := filepath.Join(root, envFileName)
		if data, err := os.ReadFile(envPath); err == nil {
			matches := envPortRegex.FindAllStringSubmatch(string(data), -1)
			for _, m := range matches {
				if len(m) == 2 {
					if portNum, err := strconv.Atoi(m[1]); err == nil && portNum > 0 && portNum <= 65535 {
						// Don't duplicate if already detected from another file
						exists := false
						for _, d := range detected {
							if d.Port == portNum && d.Source == envFileName {
								exists = true
								break
							}
						}
						if !exists {
							detected = append(detected, DetectedPort{
								Runtime: "dotenv",
								Source:  envFileName,
								Port:    portNum,
								Detail:  m[0],
							})
						}
					}
				}
			}
		}
	}

	// 5. Check for conflicts between Krewire and other runtimes (FRK-CONF-003)
	var warnings []string
	krewirePorts := make(map[int]DetectedPort)
	for _, d := range detected {
		if d.Runtime == "krewire" {
			krewirePorts[d.Port] = d
		}
	}

	for _, d := range detected {
		if d.Runtime != "krewire" {
			if kw, hasConflict := krewirePorts[d.Port]; hasConflict {
				warnings = append(warnings, fmt.Sprintf(
					"port conflict detected: port %d is declared by Krewire (%s %s) and %s (%s)",
					d.Port, kw.Source, kw.Detail, d.Runtime, d.Source,
				))
			}
		}
	}

	return detected, warnings
}
