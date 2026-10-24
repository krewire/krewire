package widget

// bounds.go — layout constraint constants for widget rendering.
// Each constant is stated once so the supported range is explicit and a change
// to a limit is a single edit rather than a search through if-chains.

// Heading level bounds.
const (
	// minHeadingLevel and maxHeadingLevel bound Heading.Level.
	minHeadingLevel, maxHeadingLevel = 1, 6
	// defaultHeadingLevel is used when a caller asks for a level outside the
	// bound, since emitting <h0> or <h7> is invalid HTML.
	defaultHeadingLevel = 2
)

// Grid column bounds.
const (
	// minGridColumns and maxGridColumns bound Grid.Columns. The stylesheet ships
	// ui-grid-2, ui-grid-3, and ui-grid-4 only, so 2 is the lower bound:
	// a one-column layout is not a grid here.
	minGridColumns, maxGridColumns = 2, 4
	// defaultGridColumns is the fallback for an out-of-range request.
	defaultGridColumns = 2
)

// Stack and checkbox layout defaults.
const (
	// defaultStackGap is the layout gap applied when the caller does not choose one.
	defaultStackGap = "1rem"
	// defaultStackAlign is the cross-axis alignment applied by default.
	defaultStackAlign = "center"
	// defaultCheckboxRows is the number of rows before checkbox groups wrap.
	defaultCheckboxRows = 4
)
