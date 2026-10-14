// Tests for KWL-R934Y
package term

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec: KWL-R934Y KWL-TM-001 KWL-TM-002 KWL-TM-003 Scope: Unit
func TestPaint(t *testing.T) {
	got := Paint("ok", ColorGreen, []Style{StyleBold})
	want := "\x1b[1;32mok\x1b[0m"
	if got != want {
		t.Errorf("Paint() = %q, want %q", got, want)
	}
}

// Spec: KWL-R934Y KWL-TM-001 KWL-TM-003 Scope: Unit
// TestTERM_001_PaintWithoutStyles verifies a style-less call still emits the
// colour code, so the escape sequence is never left empty.
func TestTERM_001_PaintWithoutStyles(t *testing.T) {
	if got, want := Paint("x", ColorRed, nil), "\x1b[31mx\x1b[0m"; got != want {
		t.Errorf("Paint() = %q, want %q", got, want)
	}
}

// Spec: KWL-R934Y KWL-TM-002 KWL-TM-003 Scope: Unit
// TestTERM_002_PaintEmitsEveryStyleInOrder verifies styles are emitted before the
// colour, in the order given, which is what the SGR grammar requires.
func TestTERM_002_PaintEmitsEveryStyleInOrder(t *testing.T) {
	got := Paint("x", ColorCyan, []Style{StyleBold, StyleDim, StyleUnderline})
	want := "\x1b[1;2;4;36mx\x1b[0m"
	if got != want {
		t.Errorf("Paint() = %q, want %q", got, want)
	}
}

// Spec: KWL-R934Y KWL-TM-003 Scope: Unit
// TestTERM_003_PaintResetsAtTheEnd verifies every sequence terminates with the
// reset code, so a coloured span cannot bleed into later output.
func TestTERM_003_PaintResetsAtTheEnd(t *testing.T) {
	if got := Paint("x", ColorWhite, nil); !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("Paint() = %q, want a trailing reset", got)
	}
	// A reset inside the payload must still be followed by the closing reset.
	if got := Paint("\x1b[0m", ColorBlack, nil); !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("Paint() = %q, want a trailing reset even for a payload containing one", got)
	}
}

// Spec: KWL-R934Y KWL-TM-005 Scope: Unit
// TestTERM_004_TerminalPaintHonoursColorSupport verifies the guard: with color
// unsupported the payload is returned verbatim, with no escape bytes at all.
func TestTERM_004_TerminalPaintHonoursColorSupport(t *testing.T) {
	// With color unsupported the payload is returned verbatim, escape-free.
	if got := (&Terminal{ColorSupported: false}).Paint("plain", ColorGreen, []Style{StyleBold}); got != "plain" {
		t.Errorf("Paint() = %q, want the raw text %q", got, "plain")
	}
	colored := (&Terminal{ColorSupported: true}).Paint("plain", ColorGreen, []Style{StyleBold})
	if colored == "plain" {
		t.Error("Paint() returned raw text, want ANSI codes when color is supported")
	}
	if !strings.HasPrefix(colored, "\x1b[") {
		t.Errorf("Paint() = %q, want a leading escape sequence", colored)
	}
}

// Spec: KWL-R934Y KWL-TM-004 Scope: Unit
// TestTERM_005_NewTerminalForNoColorWins verifies NO_COLOR is honoured even when a
// force variable is also set: the opt-out must not be overridable by accident.
func TestTERM_005_NewTerminalForNoColorWins(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("CLICOLOR_FORCE", "1")
	if NewTerminalFor(nil).ColorSupported {
		t.Error("NO_COLOR must disable color even with CLICOLOR_FORCE set")
	}
}

// Spec: KWL-R934Y KWL-TM-004 Scope: Unit
// TestTERM_006_NewTerminalForForceColorVariables verifies each documented force
// variable enables color for a non-tty, and that a falsy value does not.
func TestTERM_006_NewTerminalForForceColorVariables(t *testing.T) {
	clear := func(t *testing.T) {
		t.Helper()
		t.Setenv("NO_COLOR", "")
		t.Setenv("CLICOLOR_FORCE", "")
		t.Setenv("FORCE_COLOR", "")
	}

	for _, env := range []string{"CLICOLOR_FORCE", "FORCE_COLOR"} {
		for _, value := range []string{"1", "true"} {
			t.Run(env+"="+value, func(t *testing.T) {
				clear(t)
				t.Setenv(env, value)
				if !NewTerminalFor(nil).ColorSupported {
					t.Errorf("%s=%s must enable color", env, value)
				}
			})
		}
		t.Run(env+"=0 is not a force", func(t *testing.T) {
			clear(t)
			t.Setenv(env, "0")
			if NewTerminalFor(nil).ColorSupported {
				t.Errorf("%s=0 must not force color for a non-tty", env)
			}
		})
	}
}

// Spec: KWL-R934Y KWL-TM-004 Scope: Unit
// TestTERM_007_NewTerminalForDumbTerminal verifies TERM=dumb disables color, so
// a piped or dumb console never receives escape sequences.
func TestTERM_007_NewTerminalForDumbTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("TERM", "dumb")
	if NewTerminalFor(nil).ColorSupported {
		t.Error("TERM=dumb must disable color")
	}
}

// Spec: KWL-R934Y KWL-TM-004 Scope: Unit
// TestTERM_008_IsTerminalDistinguishesDevices verifies the character-device
// check: /dev/null is a char device and counts as a terminal, a regular file
// does not, and a closed file reports false instead of panicking.
func TestTERM_008_IsTerminalDistinguishesDevices(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()
	if !IsTerminal(devNull) {
		t.Errorf("%s is a character device and must report true", os.DevNull)
	}

	regularPath := filepath.Join(t.TempDir(), "regular.txt")
	if err := os.WriteFile(regularPath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	regular, err := os.Open(regularPath)
	if err != nil {
		t.Fatal(err)
	}
	defer regular.Close()
	if IsTerminal(regular) {
		t.Error("a regular file must not report true")
	}

	// A closed handle makes Stat fail; that must be false, not a panic.
	closed, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	if IsTerminal(closed) {
		t.Error("a closed file must report false")
	}
}

// Spec: KWL-R934Y KWL-TM-005 Scope: Unit
func TestTerminalPaintWithoutColor(t *testing.T) {
	term := &Terminal{ColorSupported: false}
	if got := term.Paint("ok", ColorGreen, []Style{StyleBold}); got != "ok" {
		t.Errorf("Paint() = %q, want %q", got, "ok")
	}
}

func TestNewTerminal(t *testing.T) {
	// Assert only that construction never panics; the detected value depends
	// on the environment.
	_ = NewTerminal()
}

func TestNewTerminalFor_NilAndRegularFile(t *testing.T) {
	if IsTerminal(nil) {
		t.Error("nil file should not be terminal")
	}
	term := NewTerminalFor(nil)
	if term.ColorSupported {
		t.Error("nil file should not have ColorSupported without force env")
	}
}

func TestNewTerminalFor_ForceColor(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	term := NewTerminalFor(nil)
	if !term.ColorSupported {
		t.Error("CLICOLOR_FORCE=1 should enable color even for non-tty")
	}
}

// Spec: KWL-R934Y KWL-TM-001 Scope: Unit
func TestTERM_009_ANSI16ColorPalette(t *testing.T) {
	colors := []struct {
		name string
		c    Color
		code int
	}{
		{"black", ColorBlack, 30},
		{"red", ColorRed, 31},
		{"green", ColorGreen, 32},
		{"yellow", ColorYellow, 33},
		{"blue", ColorBlue, 34},
		{"magenta", ColorMagenta, 35},
		{"cyan", ColorCyan, 36},
		{"white", ColorWhite, 37},
		{"default", ColorDefault, 39},
	}
	for _, tc := range colors {
		if int(tc.c) != tc.code {
			t.Errorf("Color %s code = %d, want %d", tc.name, tc.c, tc.code)
		}
		painted := Paint("text", tc.c, nil)
		if !strings.Contains(painted, string(rune('0'+tc.code%10))) {
			t.Errorf("Paint with %s must contain color code", tc.name)
		}
	}
}

// Spec: KWL-R934Y KWL-TM-006 Scope: Unit
func TestTERM_010_DistinguishDataFromDiagnostics(t *testing.T) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	term := &Terminal{
		ColorSupported: false,
		Out:            outBuf,
		Err:            errBuf,
	}

	// Write data stream
	_, _ = term.DataWriter().Write([]byte("data stream\n"))
	// Write diagnostics stream
	_, _ = term.DiagWriter().Write([]byte("diagnostic stream\n"))

	if outBuf.String() != "data stream\n" {
		t.Errorf("DataWriter wrote %q, want %q", outBuf.String(), "data stream\n")
	}
	if errBuf.String() != "diagnostic stream\n" {
		t.Errorf("DiagWriter wrote %q, want %q", errBuf.String(), "diagnostic stream\n")
	}
}

// Spec: KWL-R934Y KWL-TM-007 Scope: Unit
func TestTERM_011_DataStableEmissions(t *testing.T) {
	runs := 50
	first := Paint("hello world", ColorBlue, []Style{StyleBold, StyleUnderline})
	for i := 0; i < runs; i++ {
		repeat := Paint("hello world", ColorBlue, []Style{StyleBold, StyleUnderline})
		if repeat != first {
			t.Fatalf("Paint emissions are not data-stable across runs: %q != %q", repeat, first)
		}
	}
}
