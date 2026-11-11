package node

import "strings"

// Variant represents a visual style theme variant.
type Variant string

const (
	VariantDefault   Variant = "default"
	VariantPrimary   Variant = "primary"
	VariantSecondary Variant = "secondary"
	VariantOutline   Variant = "outline"
	VariantGhost     Variant = "ghost"
	VariantError     Variant = "error"
	VariantSuccess   Variant = "success"
	VariantWarning   Variant = "warning"
	VariantInfo      Variant = "info"
)

// Size represents the dimension scale of a UI component.
type Size string

const (
	SizeSmall  Size = "sm"
	SizeMedium Size = "md"
	SizeLarge  Size = "lg"
)

// Direction represents the layout axis.
type Direction string

const (
	DirectionHorizontal Direction = "horizontal"
	DirectionVertical   Direction = "vertical"
)

// Align represents cross-axis alignment.
type Align string

const (
	AlignStart    Align = "start"
	AlignCenter   Align = "center"
	AlignEnd      Align = "end"
	AlignStretch  Align = "stretch"
	AlignBaseline Align = "baseline"
)

// Justify represents main-axis distribution.
type Justify string

const (
	JustifyStart   Justify = "start"
	JustifyCenter  Justify = "center"
	JustifyEnd     Justify = "end"
	JustifyBetween Justify = "between"
	JustifyAround  Justify = "around"
	JustifyEvenly  Justify = "evenly"
)

// HeadingLevel defines the semantic level of an HTML heading (h1–h6).
type HeadingLevel int

const (
	Heading1 HeadingLevel = 1
	Heading2 HeadingLevel = 2
	Heading3 HeadingLevel = 3
	Heading4 HeadingLevel = 4
	Heading5 HeadingLevel = 5
	Heading6 HeadingLevel = 6
)

// InputType defines the type attribute of an HTML input.
type InputType string

const (
	InputText     InputType = "text"
	InputEmail    InputType = "email"
	InputPassword InputType = "password"
	InputNumber   InputType = "number"
	InputSearch   InputType = "search"
	InputTel      InputType = "tel"
	InputURL      InputType = "url"
	InputCheckbox InputType = "checkbox"
	InputRadio    InputType = "radio"
	InputDate     InputType = "date"
	InputTime     InputType = "time"
)

// ClassNames concatenates non-empty CSS class names separated by spaces.
func ClassNames(classes ...string) string {
	var parts []string
	for _, c := range classes {
		t := strings.TrimSpace(c)
		if t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}
