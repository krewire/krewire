// Package term provides terminal I/O, output formatting, and color
// conventions for the Krewire ecosystem.
package term

import (
	"os"
	"strconv"
	"strings"
)

// Color is an ANSI 16-color palette value.
type Color int

const (
	// ColorBlack is ANSI black.
	ColorBlack Color = 30
	// ColorRed is ANSI red.
	ColorRed Color = 31
	// ColorGreen is ANSI green.
	ColorGreen Color = 32
	// ColorYellow is ANSI yellow.
	ColorYellow Color = 33
	// ColorBlue is ANSI blue.
	ColorBlue Color = 34
	// ColorMagenta is ANSI magenta.
	ColorMagenta Color = 35
	// ColorCyan is ANSI cyan.
	ColorCyan Color = 36
	// ColorWhite is ANSI white.
	ColorWhite Color = 37
	// ColorDefault is the terminal default foreground.
	ColorDefault Color = 39
)

// Style is a text rendering style.
type Style int

const (
	// StyleBold is bold weight.
	StyleBold Style = 1
	// StyleDim is dim intensity.
	StyleDim Style = 2
	// StyleUnderline is underlined text.
	StyleUnderline Style = 4
)

// Paint wraps text in ANSI escape sequences for the given color and styles.
// Use Terminal.Paint instead when the output may not support colors.
func Paint(text string, color Color, styles []Style) string {
	codes := make([]string, 0, len(styles)+1)
	for _, s := range styles {
		codes = append(codes, strconv.Itoa(int(s)))
	}
	codes = append(codes, strconv.Itoa(int(color)))
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

// Terminal guards against emitting color codes when the destination does not
// support them.
type Terminal struct {
	// ColorSupported reports whether ANSI color codes should be emitted.
	ColorSupported bool
}

// IsTerminal reports whether f is an interactive terminal character device.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// NewTerminal detects color support from environment variables (NO_COLOR, CLICOLOR_FORCE, TERM)
// and verifies whether standard output is an interactive terminal.
func NewTerminal() *Terminal {
	return NewTerminalFor(os.Stdout)
}

// NewTerminalFor detects color support for the given file handle.
//
// The NO_COLOR convention (https://no-color.org) applies when the variable is
// present *and not empty*, so an exported-but-blank NO_COLOR does not silently
// disable colour for a whole process tree.
func NewTerminalFor(f *os.File) *Terminal {
	if noColor, ok := os.LookupEnv("NO_COLOR"); ok && noColor != "" {
		return &Terminal{ColorSupported: false}
	}
	if force, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && force != "" && force != "0" {
		return &Terminal{ColorSupported: true}
	}
	if force, ok := os.LookupEnv("FORCE_COLOR"); ok && force != "" && force != "0" {
		return &Terminal{ColorSupported: true}
	}
	ter, _ := os.LookupEnv("TERM")
	if ter == "dumb" {
		return &Terminal{ColorSupported: false}
	}
	return &Terminal{ColorSupported: IsTerminal(f)}
}

// Paint colors text, degrading to plain text when color is unsupported.
func (t *Terminal) Paint(text string, color Color, styles []Style) string {
	if !t.ColorSupported {
		return text
	}
	return Paint(text, color, styles)
}
