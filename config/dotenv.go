package config

import (
	"fmt"
	"os"
	"strings"
)

// DotEnvPair is one KEY=VALUE entry parsed from a .env file.
type DotEnvPair struct {
	Key   string
	Value string
}

// LoadDotEnv parses the .env file at path and exports its KEY=VALUE pairs
// into the process environment (KWL-2X1QZ CFG-DOTV-001). Variables already
// present in the process environment are never overwritten, so the real
// environment wins over .env. A missing file is not an error.
func LoadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: read %q: %w", path, err)
	}
	pairs, err := ParseDotEnv(data)
	if err != nil {
		return fmt.Errorf("config: parse %q: %w", path, err)
	}
	for _, kv := range pairs {
		if _, exists := os.LookupEnv(kv.Key); !exists {
			if err := os.Setenv(kv.Key, kv.Value); err != nil {
				return fmt.Errorf("config: setenv %q: %w", kv.Key, err)
			}
		}
	}
	return nil
}

// ParseDotEnv parses .env file content into ordered KEY=VALUE pairs.
// Blank lines and # comments are skipped; an optional "export " prefix is
// accepted; values may be multiline, wrapped in single or double quotes,
// contain escape sequences (when double-quoted), expand $VAR / ${VAR} variables,
// and unquoted values drop trailing inline comments (KWL-2X1QZ CFG-DOTV-002).
func ParseDotEnv(data []byte) ([]DotEnvPair, error) {
	var pairs []DotEnvPair
	lookup := func(k string) string {
		for i := len(pairs) - 1; i >= 0; i-- {
			if pairs[i].Key == k {
				return pairs[i].Value
			}
		}
		return os.Getenv(k)
	}

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	var inQuote byte
	var multilineKey string
	var multilineStartLine int
	var multilineVal strings.Builder

	for lineIdx, rawLine := range lines {
		lineNo := lineIdx + 1

		if inQuote != 0 {
			multilineVal.WriteByte('\n')
			if inQuote == '\'' {
				idx := strings.IndexByte(rawLine, '\'')
				if idx >= 0 {
					multilineVal.WriteString(rawLine[:idx])
					pairs = append(pairs, DotEnvPair{Key: multilineKey, Value: multilineVal.String()})
					inQuote = 0
					multilineVal.Reset()
				} else {
					multilineVal.WriteString(rawLine)
				}
			} else { // '"'
				idx := findClosingDoubleQuote(rawLine)
				if idx >= 0 {
					multilineVal.WriteString(rawLine[:idx])
					val := parseDoubleQuoted(multilineVal.String(), lookup)
					pairs = append(pairs, DotEnvPair{Key: multilineKey, Value: val})
					inQuote = 0
					multilineVal.Reset()
				} else {
					multilineVal.WriteString(rawLine)
				}
			}
			continue
		}

		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", lineNo)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNo)
		}

		trimmedVal := strings.TrimSpace(value)
		if strings.HasPrefix(trimmedVal, "'") {
			idx := strings.IndexByte(trimmedVal[1:], '\'')
			if idx >= 0 {
				val := trimmedVal[1 : 1+idx]
				pairs = append(pairs, DotEnvPair{Key: key, Value: val})
			} else {
				inQuote = '\''
				multilineKey = key
				multilineStartLine = lineNo
				multilineVal.WriteString(trimmedVal[1:])
			}
		} else if strings.HasPrefix(trimmedVal, "\"") {
			idx := findClosingDoubleQuote(trimmedVal[1:])
			if idx >= 0 {
				inner := trimmedVal[1 : 1+idx]
				val := parseDoubleQuoted(inner, lookup)
				pairs = append(pairs, DotEnvPair{Key: key, Value: val})
			} else {
				inQuote = '"'
				multilineKey = key
				multilineStartLine = lineNo
				multilineVal.WriteString(trimmedVal[1:])
			}
		} else {
			if i := strings.Index(trimmedVal, " #"); i >= 0 {
				trimmedVal = strings.TrimSpace(trimmedVal[:i])
			}
			val := expandVariables(trimmedVal, lookup)
			pairs = append(pairs, DotEnvPair{Key: key, Value: val})
		}
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("line %d: unclosed quote", multilineStartLine)
	}

	return pairs, nil
}

func findClosingDoubleQuote(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '"' {
			return i
		}
	}
	return -1
}

func parseDoubleQuoted(s string, lookup func(string) string) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			switch next {
			case 'n':
				sb.WriteByte('\n')
				i += 2
				continue
			case 'r':
				sb.WriteByte('\r')
				i += 2
				continue
			case 't':
				sb.WriteByte('\t')
				i += 2
				continue
			case '"':
				sb.WriteByte('"')
				i += 2
				continue
			case '\\':
				sb.WriteByte('\\')
				i += 2
				continue
			case '$':
				sb.WriteByte('$')
				i += 2
				continue
			default:
				sb.WriteByte('\\')
				sb.WriteByte(next)
				i += 2
				continue
			}
		}
		if s[i] == '$' && i+1 < len(s) {
			if s[i+1] == '{' {
				end := strings.IndexByte(s[i+2:], '}')
				if end >= 0 {
					varName := s[i+2 : i+2+end]
					sb.WriteString(lookup(varName))
					i = i + 2 + end + 1
					continue
				}
			} else if isVarIdentStart(s[i+1]) {
				j := i + 1
				for j < len(s) && isVarIdentPart(s[j]) {
					j++
				}
				varName := s[i+1 : j]
				sb.WriteString(lookup(varName))
				i = j
				continue
			}
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}

func expandVariables(s string, lookup func(string) string) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '$' {
			sb.WriteByte('$')
			i += 2
			continue
		}
		if s[i] == '$' && i+1 < len(s) {
			if s[i+1] == '{' {
				end := strings.IndexByte(s[i+2:], '}')
				if end >= 0 {
					varName := s[i+2 : i+2+end]
					sb.WriteString(lookup(varName))
					i = i + 2 + end + 1
					continue
				}
			} else if isVarIdentStart(s[i+1]) {
				j := i + 1
				for j < len(s) && isVarIdentPart(s[j]) {
					j++
				}
				varName := s[i+1 : j]
				sb.WriteString(lookup(varName))
				i = j
				continue
			}
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}

func isVarIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isVarIdentPart(c byte) bool {
	return isVarIdentStart(c) || (c >= '0' && c <= '9')
}
