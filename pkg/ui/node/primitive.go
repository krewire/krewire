package node

import (
	"fmt"
	"strings"
)

// ButtonProps defines options for a Button component.
type ButtonProps struct {
	Variant  Variant // primary, secondary, outline, ghost, error, default
	Size     Size    // sm, md, lg
	Type     string  // button, submit, reset
	Disabled bool    // disabled state
	URL      string  // if provided, renders an <a> anchor styled as button
	Icon     string  // leading raw SVG or text icon
	OnClick  string  // client-side handler or action attribute
	Class    string
	Attrs    Attrs
}

// Button renders a clickable button or anchor link with Pop-Brutalist styling.
func Button(props ButtonProps, children ...Node) Node {
	var tag string
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	classes := []string{"kiw-btn"}
	switch props.Variant {
	case VariantPrimary:
		classes = append(classes, "kiw-btn-primary")
	case VariantSecondary:
		classes = append(classes, "kiw-btn-secondary")
	case VariantOutline:
		classes = append(classes, "kiw-btn-outline")
	case VariantGhost:
		classes = append(classes, "kiw-btn-ghost")
	case VariantError:
		classes = append(classes, "kiw-btn-error")
	case VariantSuccess:
		classes = append(classes, "kiw-btn-success")
	default:
		classes = append(classes, "kiw-btn-default")
	}

	switch props.Size {
	case SizeSmall:
		classes = append(classes, "kiw-btn-sm")
	case SizeLarge:
		classes = append(classes, "kiw-btn-lg")
	default:
		classes = append(classes, "kiw-btn-md")
	}

	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	var content []Node
	if props.Icon != "" {
		content = append(content, RawHTML(props.Icon))
	}
	content = append(content, children...)

	if props.URL != "" {
		tag = "a"
		attrs["href"] = SafeURL(props.URL)
		if props.Disabled {
			attrs["aria-disabled"] = "true"
			attrs["tabindex"] = "-1"
		}
	} else {
		tag = "button"
		btnType := props.Type
		if btnType == "" {
			btnType = "button"
		}
		attrs["type"] = btnType
		if props.Disabled {
			attrs["disabled"] = true
		}
	}

	if props.OnClick != "" {
		attrs["onclick"] = props.OnClick
	}

	return Element(tag, attrs, content...)
}

// InputProps defines options for an Input field.
type InputProps struct {
	Type        InputType // text, email, password, etc.
	Name        string
	Value       string
	Placeholder string
	Required    bool
	Disabled    bool
	ReadOnly    bool
	AutoFocus   bool
	Class       string
	Attrs       Attrs
}

// Input renders an HTML form input element.
func Input(props InputProps) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	inputType := string(props.Type)
	if inputType == "" {
		inputType = string(InputText)
	}
	attrs["type"] = inputType

	if props.Name != "" {
		attrs["name"] = props.Name
	}
	if props.Value != "" {
		attrs["value"] = props.Value
	}
	if props.Placeholder != "" {
		attrs["placeholder"] = props.Placeholder
	}
	if props.Required {
		attrs["required"] = true
	}
	if props.Disabled {
		attrs["disabled"] = true
	}
	if props.ReadOnly {
		attrs["readonly"] = true
	}
	if props.AutoFocus {
		attrs["autofocus"] = true
	}

	classes := []string{"kiw-input"}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	return Element("input", attrs)
}

// HeadingProps defines options for a Heading element.
type HeadingProps struct {
	Level HeadingLevel // 1–6 (defaults to 1)
	Class string
	Attrs Attrs
}

// Heading renders a semantic HTML heading element (h1 to h6).
func Heading(props HeadingProps, children ...Node) Node {
	level := int(props.Level)
	if level < 1 || level > 6 {
		level = 1
	}
	tag := fmt.Sprintf("h%d", level)

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	classes := []string{fmt.Sprintf("kiw-heading-%d", level)}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	return Element(tag, attrs, children...)
}

// TextProps defines options for text paragraphs or spans.
type TextProps struct {
	Tag   string // p, span, div (defaults to p)
	Muted bool
	Bold  bool
	Class string
	Attrs Attrs
}

// Paragraph renders a styled text element.
func Paragraph(props TextProps, children ...Node) Node {
	tag := strings.TrimSpace(props.Tag)
	if tag == "" {
		tag = "p"
	}

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	var classes []string
	if props.Muted {
		classes = append(classes, "text-muted")
	}
	if props.Bold {
		classes = append(classes, "font-bold")
	}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	if len(classes) > 0 {
		attrs["class"] = ClassNames(classes...)
	}

	return Element(tag, attrs, children...)
}

// BadgeProps defines options for a Badge component.
type BadgeProps struct {
	Variant Variant // default, primary, secondary, success, error, warning
	Size    Size    // sm, md, lg
	Class   string
	Attrs   Attrs
}

// Badge renders an inline tag or badge indicator.
func Badge(props BadgeProps, children ...Node) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	classes := []string{"kiw-badge"}
	switch props.Variant {
	case VariantPrimary:
		classes = append(classes, "kiw-badge-primary")
	case VariantSecondary:
		classes = append(classes, "kiw-badge-secondary")
	case VariantSuccess:
		classes = append(classes, "kiw-badge-success")
	case VariantError:
		classes = append(classes, "kiw-badge-error")
	case VariantWarning:
		classes = append(classes, "kiw-badge-warning")
	default:
		classes = append(classes, "kiw-badge-default")
	}

	if props.Size == SizeSmall {
		classes = append(classes, "kiw-badge-sm")
	} else if props.Size == SizeLarge {
		classes = append(classes, "kiw-badge-lg")
	}

	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	return Element("span", attrs, children...)
}

// CardProps defines options for Card component.
type CardProps struct {
	Title    string
	Subtitle string
	Variant  Variant
	Slots    SlotMap // named slots: "header", "footer", "actions"
	Class    string
	Attrs    Attrs
}

// Card renders a container card with optional header, footer, and body slots.
func Card(props CardProps, children ...Node) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	classes := []string{"kiw-card"}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	var cardContent []Node

	// Header
	headerSlot := Slot("header")
	if props.Title != "" || props.Subtitle != "" {
		var headerNodes []Node
		if props.Title != "" {
			headerNodes = append(headerNodes, Heading(HeadingProps{Level: 3}, Text(props.Title)))
		}
		if props.Subtitle != "" {
			headerNodes = append(headerNodes, Paragraph(TextProps{Muted: true}, Text(props.Subtitle)))
		}
		headerSlot = Slot("header", Element("div", Attrs{"class": "kiw-card-header"}, headerNodes...))
	}
	cardContent = append(cardContent, headerSlot)

	// Body / default slot
	if len(children) > 0 {
		cardContent = append(cardContent, Element("div", Attrs{"class": "kiw-card-body"}, children...))
	} else {
		cardContent = append(cardContent, Slot("default"))
	}

	// Footer slot
	cardContent = append(cardContent, Slot("footer"))

	nodeTree := Element("div", attrs, cardContent...)
	if len(props.Slots) > 0 {
		return WithSlots(nodeTree, props.Slots)
	}
	return nodeTree
}

// AlertProps defines options for an Alert box.
type AlertProps struct {
	Variant     Variant // error, warning, success, info, default
	Title       string
	Dismissible bool
	Class       string
	Attrs       Attrs
}

// Alert renders an alert message box.
func Alert(props AlertProps, children ...Node) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	classes := []string{"kiw-alert"}
	switch props.Variant {
	case VariantError:
		classes = append(classes, "kiw-alert-error")
	case VariantWarning:
		classes = append(classes, "kiw-alert-warning")
	case VariantSuccess:
		classes = append(classes, "kiw-alert-success")
	case VariantInfo:
		classes = append(classes, "kiw-alert-info")
	default:
		classes = append(classes, "kiw-alert-default")
	}

	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)
	attrs["role"] = "alert"

	var content []Node
	if props.Title != "" {
		content = append(content, Element("strong", Attrs{"class": "kiw-alert-title"}, Text(props.Title)))
	}
	content = append(content, children...)

	return Element("div", attrs, content...)
}

// DividerProps defines options for a Divider line.
type DividerProps struct {
	Label    string
	Vertical bool
	Class    string
	Attrs    Attrs
}

// Divider renders a horizontal or vertical rule with optional text label.
func Divider(props DividerProps) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	if props.Vertical {
		classes := []string{"kiw-divider-v"}
		if props.Class != "" {
			classes = append(classes, props.Class)
		}
		attrs["class"] = ClassNames(classes...)
		attrs["role"] = "separator"
		attrs["aria-orientation"] = "vertical"
		return Element("div", attrs)
	}

	if props.Label != "" {
		classes := []string{"kiw-divider-with-label"}
		if props.Class != "" {
			classes = append(classes, props.Class)
		}
		attrs["class"] = ClassNames(classes...)
		attrs["role"] = "separator"
		return Element("div", attrs,
			Element("span", Attrs{"class": "kiw-divider-line"}),
			Element("span", Attrs{"class": "kiw-divider-text"}, Text(props.Label)),
			Element("span", Attrs{"class": "kiw-divider-line"}),
		)
	}

	classes := []string{"kiw-divider"}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)
	return Element("hr", attrs)
}

// ModalProps defines options for a dialog modal.
type ModalProps struct {
	ID    string
	Title string
	Open  bool
	Slots SlotMap // "footer", "body"
	Class string
	Attrs Attrs
}

// Modal renders a dialog or modal window.
func Modal(props ModalProps, children ...Node) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	if props.ID != "" {
		attrs["id"] = props.ID
	}
	if props.Open {
		attrs["open"] = true
	}

	classes := []string{"kiw-modal"}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	var modalBox []Node
	if props.Title != "" {
		modalBox = append(modalBox, Element("div", Attrs{"class": "kiw-modal-header"},
			Heading(HeadingProps{Level: 3}, Text(props.Title)),
		))
	}

	if len(children) > 0 {
		modalBox = append(modalBox, Element("div", Attrs{"class": "kiw-modal-body"}, children...))
	} else {
		modalBox = append(modalBox, Slot("default"))
	}

	modalBox = append(modalBox, Slot("footer"))

	box := Element("div", Attrs{"class": "kiw-modal-box"}, modalBox...)
	tree := Element("dialog", attrs, box)

	if len(props.Slots) > 0 {
		return WithSlots(tree, props.Slots)
	}
	return tree
}

// TableProps defines options for a Table component.
type TableProps struct {
	Headers   []string
	Rows      [][]Node
	Striped   bool
	Hoverable bool
	Class     string
	Attrs     Attrs
}

// Table renders an HTML <table> with structured headers and rows.
func Table(props TableProps) Node {
	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}

	classes := []string{"kiw-table"}
	if props.Striped {
		classes = append(classes, "kiw-table-striped")
	}
	if props.Hoverable {
		classes = append(classes, "kiw-table-hover")
	}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	var tableChildren []Node

	if len(props.Headers) > 0 {
		var headerCells []Node
		for _, h := range props.Headers {
			headerCells = append(headerCells, Element("th", nil, Text(h)))
		}
		tableChildren = append(tableChildren,
			Element("thead", nil, Element("tr", nil, headerCells...)),
		)
	}

	var rowNodes []Node
	for _, row := range props.Rows {
		var cells []Node
		for _, cell := range row {
			cells = append(cells, Element("td", nil, cell))
		}
		rowNodes = append(rowNodes, Element("tr", nil, cells...))
	}
	tableChildren = append(tableChildren, Element("tbody", nil, rowNodes...))

	return Element("table", attrs, tableChildren...)
}
