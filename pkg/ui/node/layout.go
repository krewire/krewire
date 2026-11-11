package node

import (
	"fmt"
	"strings"
)

// ContainerProps defines options for Container.
type ContainerProps struct {
	MaxWidth string // e.g. "max-w-7xl", "max-w-md", "1200px"
	Padding  string // e.g. "px-4 py-8", "2rem"
	Centered bool   // adds mx-auto
	Class    string
	Attrs    Attrs
}

// Container renders a responsive centered layout wrapper.
func Container(props ContainerProps, children ...Node) Node {
	classes := []string{"kiw-container"}
	if props.Centered || props.MaxWidth != "" {
		classes = append(classes, "mx-auto")
	}
	if props.MaxWidth != "" {
		if strings.HasPrefix(props.MaxWidth, "max-w-") {
			classes = append(classes, props.MaxWidth)
		}
	}
	if props.Padding != "" && !strings.Contains(props.Padding, ":") {
		classes = append(classes, props.Padding)
	}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}
	attrs["class"] = ClassNames(classes...)

	var inlineStyles []string
	if props.MaxWidth != "" && !strings.HasPrefix(props.MaxWidth, "max-w-") {
		inlineStyles = append(inlineStyles, "max-width:"+props.MaxWidth)
	}
	if props.Padding != "" && strings.Contains(props.Padding, ":") {
		inlineStyles = append(inlineStyles, "padding:"+props.Padding)
	}
	if len(inlineStyles) > 0 {
		if existing, ok := attrs["style"].(string); ok && existing != "" {
			attrs["style"] = existing + ";" + strings.Join(inlineStyles, ";")
		} else {
			attrs["style"] = strings.Join(inlineStyles, ";")
		}
	}

	return Element("div", attrs, children...)
}

// StackProps defines options for layout stacks (VStack, HStack).
type StackProps struct {
	Direction Direction // horizontal or vertical
	Gap       string    // e.g. "gap-4", "1rem", "1.5rem"
	Align     Align     // items-start, items-center, items-end, items-stretch
	Justify   Justify   // justify-start, justify-center, justify-end, justify-between
	Wrap      bool      // flex-wrap
	Class     string
	Attrs     Attrs
}

// Stack renders a flexible flexbox layout stack.
func Stack(props StackProps, children ...Node) Node {
	classes := []string{"flex"}
	if props.Direction == DirectionVertical {
		classes = append(classes, "flex-col")
	} else {
		classes = append(classes, "flex-row")
	}

	if props.Wrap {
		classes = append(classes, "flex-wrap")
	}

	if props.Align != "" {
		classes = append(classes, "items-"+string(props.Align))
	}
	if props.Justify != "" {
		classes = append(classes, "justify-"+string(props.Justify))
	}
	if props.Gap != "" && strings.HasPrefix(props.Gap, "gap-") {
		classes = append(classes, props.Gap)
	}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}
	attrs["class"] = ClassNames(classes...)

	if props.Gap != "" && !strings.HasPrefix(props.Gap, "gap-") {
		gapStyle := "gap:" + props.Gap
		if existing, ok := attrs["style"].(string); ok && existing != "" {
			attrs["style"] = existing + ";" + gapStyle
		} else {
			attrs["style"] = gapStyle
		}
	}

	return Element("div", attrs, children...)
}

// VStack renders a vertical stack (flex-direction: column).
func VStack(props StackProps, children ...Node) Node {
	props.Direction = DirectionVertical
	return Stack(props, children...)
}

// HStack renders a horizontal stack (flex-direction: row).
func HStack(props StackProps, children ...Node) Node {
	props.Direction = DirectionHorizontal
	return Stack(props, children...)
}

// GridProps defines options for Grid.
type GridProps struct {
	Columns int    // number of columns (1-12)
	Gap     string // e.g. "gap-6", "1.5rem"
	Class   string
	Attrs   Attrs
}

// Grid renders a CSS grid container.
func Grid(props GridProps, children ...Node) Node {
	classes := []string{"grid"}
	if props.Columns > 0 {
		classes = append(classes, fmt.Sprintf("grid-cols-%d", props.Columns))
	}
	if props.Gap != "" && strings.HasPrefix(props.Gap, "gap-") {
		classes = append(classes, props.Gap)
	}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}
	attrs["class"] = ClassNames(classes...)

	var inlineStyles []string
	if props.Columns > 0 && props.Columns > 12 {
		inlineStyles = append(inlineStyles, fmt.Sprintf("grid-template-columns: repeat(%d, minmax(0, 1fr))", props.Columns))
	}
	if props.Gap != "" && !strings.HasPrefix(props.Gap, "gap-") {
		inlineStyles = append(inlineStyles, "gap:"+props.Gap)
	}
	if len(inlineStyles) > 0 {
		if existing, ok := attrs["style"].(string); ok && existing != "" {
			attrs["style"] = existing + ";" + strings.Join(inlineStyles, ";")
		} else {
			attrs["style"] = strings.Join(inlineStyles, ";")
		}
	}

	return Element("div", attrs, children...)
}

// SectionProps defines options for a semantic Section.
type SectionProps struct {
	Title    string
	Subtitle string
	Class    string
	Attrs    Attrs
}

// Section renders an HTML5 <section> element with optional header.
func Section(props SectionProps, children ...Node) Node {
	var nodes []Node
	if props.Title != "" || props.Subtitle != "" {
		var headerNodes []Node
		if props.Title != "" {
			headerNodes = append(headerNodes, Heading(HeadingProps{Level: 2}, Text(props.Title)))
		}
		if props.Subtitle != "" {
			headerNodes = append(headerNodes, Paragraph(TextProps{Muted: true}, Text(props.Subtitle)))
		}
		nodes = append(nodes, Element("header", Attrs{"class": "mb-6"}, headerNodes...))
	}
	nodes = append(nodes, children...)

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}
	classes := []string{"kiw-section"}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}
	attrs["class"] = ClassNames(classes...)

	return Element("section", attrs, nodes...)
}

// BoxProps defines options for a generic Box element.
type BoxProps struct {
	Padding    string
	Margin     string
	Background string
	Border     bool
	Rounded    bool
	Class      string
	Attrs      Attrs
}

// Box renders a styled container block.
func Box(props BoxProps, children ...Node) Node {
	classes := []string{"kiw-box"}
	if props.Border {
		classes = append(classes, "border-2 border-black dark:border-white")
	}
	if props.Rounded {
		classes = append(classes, "rounded-lg")
	}
	if props.Background != "" {
		classes = append(classes, props.Background)
	}
	if props.Padding != "" && !strings.Contains(props.Padding, ":") {
		classes = append(classes, props.Padding)
	}
	if props.Margin != "" && !strings.Contains(props.Margin, ":") {
		classes = append(classes, props.Margin)
	}
	if props.Class != "" {
		classes = append(classes, props.Class)
	}

	attrs := make(Attrs)
	for k, v := range props.Attrs {
		attrs[k] = v
	}
	attrs["class"] = ClassNames(classes...)

	var inlineStyles []string
	if props.Padding != "" && strings.Contains(props.Padding, ":") {
		inlineStyles = append(inlineStyles, "padding:"+props.Padding)
	}
	if props.Margin != "" && strings.Contains(props.Margin, ":") {
		inlineStyles = append(inlineStyles, "margin:"+props.Margin)
	}
	if len(inlineStyles) > 0 {
		if existing, ok := attrs["style"].(string); ok && existing != "" {
			attrs["style"] = existing + ";" + strings.Join(inlineStyles, ";")
		} else {
			attrs["style"] = strings.Join(inlineStyles, ";")
		}
	}

	return Element("div", attrs, children...)
}

// FlexProps defines options for Flex container.
type FlexProps struct {
	Direction Direction
	Gap       string
	Align     Align
	Justify   Justify
	Wrap      bool
	Class     string
	Attrs     Attrs
}

// Flex renders a generic CSS flex container.
func Flex(props FlexProps, children ...Node) Node {
	return Stack(StackProps{
		Direction: props.Direction,
		Gap:       props.Gap,
		Align:     props.Align,
		Justify:   props.Justify,
		Wrap:      props.Wrap,
		Class:     props.Class,
		Attrs:     props.Attrs,
	}, children...)
}
