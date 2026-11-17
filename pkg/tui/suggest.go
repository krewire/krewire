package tui

import (
	"fmt"
	"strings"

	"github.com/krewire/krewire/pkg/term"
)

func (a *App) printUnknownCommand(name string) {
	msg := a.terminal.Paint("✗", term.ColorRed, []term.Style{term.StyleBold}) + " unknown command " + a.terminal.Paint(fmt.Sprintf("%q", name), term.ColorYellow, nil)
	a.stderr.Write([]byte(msg + "\n"))
	if hint, ok := a.hints[name]; ok {
		a.stderr.Write([]byte(a.terminal.Paint("  did you mean ", term.ColorDefault, []term.Style{term.StyleDim}) + a.terminal.Paint(hint, term.ColorCyan, nil) + "\n\n"))
		return
	}
	if sug := a.suggestCommand(name); sug != "" {
		hint := a.terminal.Paint("  did you mean ", term.ColorDefault, []term.Style{term.StyleDim}) + a.terminal.Paint(sug, term.ColorCyan, nil) + a.terminal.Paint(" ?", term.ColorDefault, []term.Style{term.StyleDim})
		a.stderr.Write([]byte(hint + "\n"))
	}
	a.stderr.Write([]byte("\n"))
}

func (a *App) suggestCommand(name string) string {
	names := make([]string, 0, len(a.commands))
	for _, c := range a.commands {
		names = append(names, c.Name)
	}
	return Suggest(name, names)
}

// Suggest returns the candidate a mistyped name most likely meant: the first
// candidate that name is a prefix of, otherwise the closest candidate within
// an edit distance of 2 (case-insensitive). It returns "" when none is close.
func Suggest(name string, candidates []string) string {
	best := ""
	bestDist := 3
	lname := strings.ToLower(name)
	for _, c := range candidates {
		lc := strings.ToLower(c)
		if lname != "" && strings.HasPrefix(lc, lname) {
			return c
		}
		if d := levenshtein(lname, lc); d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}
