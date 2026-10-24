package widget

// style.go — inline style constants for widget rendering.
// Naming these strings keeps the visual language to a single edit per value
// instead of scattering literals through the render functions. Every value
// resolves through a --ui-* custom property so callers can override the
// palette without editing this file.
const (
	styleMutedText   = `style="color:var(--ui-muted)"`
	styleCardHeader  = ` style="display:flex; justify-content:space-between; align-items:flex-start;"`
	styleCardFooter  = ` style="border-top:1.5px solid var(--ui-border); padding-top:0.75rem; margin-top:1rem;"`
	styleHeading     = ` style="margin:0.5rem 0; font-weight:800"`
	styleDividerRule = ` style="border:none; border-top:var(--ui-pop-border); margin:1.25rem 0;"`
	styleDividerWrap = ` style="display:flex; align-items:center; gap:0.75rem; margin:1.25rem 0;"`
	styleDividerFill = ` style="flex:1; border:none; border-top:var(--ui-pop-border);"`
	styleDividerText = ` style="font-size:0.75rem; font-weight:700; color:var(--ui-muted); text-transform:uppercase;"`
	styleAlertIcon   = ` style="font-weight:900;"`
	styleModal       = ` style="border:var(--ui-pop-border); border-radius:var(--ui-radius); background:var(--ui-bg); box-shadow:var(--ui-pop-shadow); padding:0; max-width:540px; width:90%;"`
	styleModalHead   = ` style="padding:1rem 1.25rem; border-bottom:1.5px solid var(--ui-border); display:flex; justify-content:space-between; align-items:center;"`
	styleModalTitle  = ` style="margin:0; font-size:1.15rem; font-weight:800;"`
	styleModalClose  = ` style="border:none; background:none; font-size:1.5rem; line-height:1; cursor:pointer;"`
	styleModalBody   = ` style="padding:1.25rem;"`
	styleModalFoot   = ` style="padding:0.75rem 1.25rem; border-top:1.5px solid var(--ui-border); background:var(--ui-surface); display:flex; justify-content:flex-end; gap:0.5rem;"`
	styleFormActions = ` style="margin-top:0.5rem;"`
	styleCheckboxRow = ` style="flex-direction:row; align-items:center; gap:0.5rem;"`
)
