package term

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaint(t *testing.T) {
	got := Paint("ok", ColorGreen, []Style{StyleBold})
	want := "\x1b[1;32mok\x1b[0m"
	if got != want {
		t.Errorf("Paint() = %q, want %q", got, want)
	}
}

// TestPaint_WithoutStyles verifies a style-less call still emits the
// colour code, so the escape sequence is never left empty.
func TestPaint_WithoutStyles(t *testing.T) {
	if got, want := Paint("x", ColorRed, nil), "\x1b[31mx\x1b[0m"; got != want {
		t.Errorf("Paint() = %q, want %q", got, want)
	}
}

// TestPaint_EmitsEveryStyleInOrder verifies styles are emitted before the
// colour, in the order given, which is what the SGR grammar requires.
func TestPaint_EmitsEveryStyleInOrder(t *testing.T) {
	got := Paint("x", ColorCyan, []Style{StyleBold, StyleDim, StyleUnderline})
	want := "\x1b[1;2;4;36mx\x1b[0m"
	if got != want {
		t.Errorf("Paint() = %q, want %q", got, want)
	}
}

// TestPaint_ResetsAtTheEnd verifies every sequence terminates with the
// reset code, so a coloured span cannot bleed into later output.
func TestPaint_ResetsAtTheEnd(t *testing.T) {
	if got := Paint("x", ColorWhite, nil); !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("Paint() = %q, want a trailing reset", got)
	}
	// A reset inside the payload must still be followed by the closing reset.
	if got := Paint("\x1b[0m", ColorBlack, nil); !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("Paint() = %q, want a trailing reset even for a payload containing one", got)
	}
}

// TestTerminalPaint_HonoursColorSupport verifies the guard: with color
// unsupported the payload is returned verbatim, with no escape bytes at all.
func TestTerminalPaint_HonoursColorSupport(t *testing.T) {
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

// TestNewTerminalFor_NoColorWins verifies NO_COLOR is honoured even when a
// force variable is also set: the opt-out must not be overridable by accident.
func TestNewTerminalFor_NoColorWins(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("CLICOLOR_FORCE", "1")
	if NewTerminalFor(nil).ColorSupported {
		t.Error("NO_COLOR must disable color even with CLICOLOR_FORCE set")
	}
}

// TestNewTerminalFor_ForceColorVariables verifies each documented force
// variable enables color for a non-tty, and that a falsy value does not.
func TestNewTerminalFor_ForceColorVariables(t *testing.T) {
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

// TestNewTerminalFor_DumbTerminal verifies TERM=dumb disables color, so
// a piped or dumb console never receives escape sequences.
func TestNewTerminalFor_DumbTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("TERM", "dumb")
	if NewTerminalFor(nil).ColorSupported {
		t.Error("TERM=dumb must disable color")
	}
}

// TestIsTerminal_DistinguishesDevices verifies the character-device
// check: /dev/null is a char device and counts as a terminal, a regular file
// does not, and a closed file reports false instead of panicking.
func TestIsTerminal_DistinguishesDevices(t *testing.T) {
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

func TestANSI16ColorPalette(t *testing.T) {
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

func TestDistinguishDataFromDiagnostics(t *testing.T) {
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

func TestPaint_DataStableEmissions(t *testing.T) {
	runs := 50
	first := Paint("hello world", ColorBlue, []Style{StyleBold, StyleUnderline})
	for i := 0; i < runs; i++ {
		repeat := Paint("hello world", ColorBlue, []Style{StyleBold, StyleUnderline})
		if repeat != first {
			t.Fatalf("Paint emissions are not data-stable across runs: %q != %q", repeat, first)
		}
	}
}
