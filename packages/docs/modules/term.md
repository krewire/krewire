# `term`

Import: `github.com/krewire/libs/term`

## Purpose

Terminal detection, ANSI colors, styles, and output conventions for command-line applications.

## Main API

- `Terminal`, `NewTerminal()`, `NewTerminalFor(file)`
- `IsTerminal(file)`
- `Paint(text, color, styles)`
- `Color` and `Style` constants

## Example

```go
t := term.NewTerminal()
fmt.Fprintln(t.Out, term.Paint("ready", term.ColorGreen, []term.Style{term.StyleBold}))
```

## Design boundary

`term` owns presentation concerns only. It should not contain business rules, configuration loading, or process lifecycle orchestration.
