// Package term provides terminal I/O, output formatting, and color
// conventions for the Krewire ecosystem.
package term

import (
	"io"
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
// support them, and distinguishes data output (stdout) from diagnostics (stderr).
type Terminal struct {
	// ColorSupported reports whether ANSI color codes should be emitted.
	ColorSupported bool
	// Out is the destination for data output (defaults to os.Stdout).
	Out io.Writer
	// Err is the destination for diagnostics and warnings (defaults to os.Stderr).
	Err io.Writer
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
	var colorSupported bool
	if noColor, ok := os.LookupEnv("NO_COLOR"); ok && noColor != "" {
		colorSupported = false
	} else if force, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && force != "" && force != "0" {
		colorSupported = true
	} else if force, ok := os.LookupEnv("FORCE_COLOR"); ok && force != "" && force != "0" {
		colorSupported = true
	} else if ter, _ := os.LookupEnv("TERM"); ter == "dumb" {
		colorSupported = false
	} else {
		colorSupported = IsTerminal(f)
	}

	var out io.Writer = f
	if f == nil {
		out = os.Stdout
	}
	return &Terminal{
		ColorSupported: colorSupported,
		Out:            out,
		Err:            os.Stderr,
	}
}

// DataWriter returns the writer for data output (Out or os.Stdout).
func (t *Terminal) DataWriter() io.Writer {
	if t != nil && t.Out != nil {
		return t.Out
	}
	return os.Stdout
}

// DiagWriter returns the writer for diagnostics and warnings (Err or os.Stderr).
func (t *Terminal) DiagWriter() io.Writer {
	if t != nil && t.Err != nil {
		return t.Err
	}
	return os.Stderr
}

// Paint colors text, degrading to plain text when color is unsupported.
func (t *Terminal) Paint(text string, color Color, styles []Style) string {
	if !t.ColorSupported {
		return text
	}
	return Paint(text, color, styles)
}
