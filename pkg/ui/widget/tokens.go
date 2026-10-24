package widget

// Variant names shared by Button, Badge, and Alert. They are the CSS modifier
// suffix on the ui-* class, so the value is both the Go-side vocabulary and
// the stylesheet contract: changing one without the other silently unstyles the
// component.
const (
	VariantPrimary   = "primary"
	VariantSecondary = "secondary"
	VariantSuccess   = "success"
	VariantWarning   = "warning"
	VariantError     = "error"
	VariantInfo      = "info"
	VariantDefault   = "default"
)

// Button sizes. SizeMedium is the unstyled default and is never emitted as a
// modifier class.
const (
	SizeSmall  = "sm"
	SizeMedium = "md"
	SizeLarge  = "lg"
)

// Stack directions, stored as values so the render switch is explicit.
const (
	DirectionVertical   = "vertical"
	DirectionHorizontal = "horizontal"
)

// Class prefixes. Every widget composes its class list from these so the
// stylesheet surface has one place to change. Exported so external packages
// can compose class names consistently with the Krewire widget system.
const (
	ButtonClassPrefix  = "ui-btn"
	BadgeClassPrefix   = "ui-badge"
	AlertClassPrefix   = "ui-alert"
	CardClassPrefix    = "ui-card"
	TableClassPrefix   = "ui-table"
	ModalClassPrefix   = "ui-modal"
	StackClassPrefix   = "ui-stack"
	GridClassPrefix    = "ui-grid"
	HeadingClassPrefix = "ui-heading"
)

// unexported aliases used inside the widget package — kept to avoid rewriting
// every reference in the same package while still exporting the names.
const (
	buttonClassPrefix  = ButtonClassPrefix
	badgeClassPrefix   = BadgeClassPrefix
	alertClassPrefix   = AlertClassPrefix
	cardClassPrefix    = CardClassPrefix
	tableClassPrefix   = TableClassPrefix
	modalClassPrefix   = ModalClassPrefix
	stackClassPrefix   = StackClassPrefix
	gridClassPrefix    = GridClassPrefix
	headingClassPrefix = HeadingClassPrefix
)

// Default alert icons, paired with their variants. The pairing is the point: a
// caller selecting a variant should not also have to remember which glyph goes
// with it.
const (
	iconInfo    = "ℹ"
	iconSuccess = "✓"
	iconWarning = "⚠"
	iconError   = "✕"
)

// modifier builds "<prefix>-<modifier>" and returns prefix alone for an empty
// modifier, so a default variant never emits a dangling "ui-btn-".
func modifier(prefix, name string) string {
	if name == "" {
		return prefix
	}
	return prefix + "-" + name
}
