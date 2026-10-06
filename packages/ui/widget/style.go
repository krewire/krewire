package widget

// Inline styles are named rather than embedded in the render functions so the
// visual language can be adjusted in one place, and so a review of "what does
// this look like" does not require reading the markup assembly. Where a value is
// already exposed as a theme token (var(--forge-...)) the token is used rather
// than a literal colour or size.
const (
	styleMutedText   = `style="color:var(--forge-muted)"`
	styleCardHeader  = ` style="display:flex; justify-content:space-between; align-items:flex-start;"`
	styleCardFooter  = ` style="border-top:1.5px solid var(--forge-border); padding-top:0.75rem; margin-top:1rem;"`
	styleHeading     = ` style="margin:0.5rem 0; font-weight:800"`
	styleDividerRule = ` style="border:none; border-top:var(--forge-pop-border); margin:1.25rem 0;"`
	styleDividerWrap = ` style="display:flex; align-items:center; gap:0.75rem; margin:1.25rem 0;"`
	styleDividerFill = ` style="flex:1; border:none; border-top:var(--forge-pop-border);"`
	styleDividerText = ` style="font-size:0.75rem; font-weight:700; color:var(--forge-muted); text-transform:uppercase;"`
	styleAlertIcon   = ` style="font-weight:900;"`
	styleModal       = ` style="border:var(--forge-pop-border); border-radius:var(--forge-radius); background:var(--forge-bg); box-shadow:var(--forge-pop-shadow); padding:0; max-width:540px; width:90%;"`
	styleModalHead   = ` style="padding:1rem 1.25rem; border-bottom:1.5px solid var(--forge-border); display:flex; justify-content:space-between; align-items:center;"`
	styleModalTitle  = ` style="margin:0; font-size:1.15rem; font-weight:800;"`
	styleModalClose  = ` style="border:none; background:none; font-size:1.5rem; line-height:1; cursor:pointer;"`
	styleModalBody   = ` style="padding:1.25rem;"`
	styleModalFoot   = ` style="padding:0.75rem 1.25rem; border-top:1.5px solid var(--forge-border); background:var(--forge-surface); display:flex; justify-content:flex-end; gap:0.5rem;"`
	styleFormActions = ` style="margin-top:0.5rem;"`
	styleCheckboxRow = ` style="flex-direction:row; align-items:center; gap:0.5rem;"`
)

// Layout bounds, named so the supported range is stated once instead of being
// implied by an if-chain.
const (
	// minHeadingLevel and maxHeadingLevel bound Heading.Level.
	minHeadingLevel, maxHeadingLevel = 1, 6
	// defaultHeadingLevel is used when a caller asks for a level outside the
	// bound, since emitting <h0> or <h7> is invalid HTML.
	defaultHeadingLevel = 2
	// minGridColumns and maxGridColumns bound Grid.Columns. The stylesheet ships
	// forge-grid-2, forge-grid-3, and forge-grid-4 only, so 2 is the lower bound:
	// a one-column layout is not a grid here.
	minGridColumns, maxGridColumns = 2, 4
	// defaultGridColumns is the fallback for an out-of-range request.
	defaultGridColumns = 2
	// defaultStackGap and defaultCheckboxRows are the layout defaults applied
	// when the caller does not choose one.
	defaultStackGap     = "1rem"
	defaultStackAlign   = "center"
	defaultCheckboxRows = 4
)
