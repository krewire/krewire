package widget

import (
	"strings"
	"testing"
)

// mustContain fails when want is absent from the rendered markup.
func mustContain(t *testing.T, got, want, what string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s: rendered markup missing %q\n%s", what, want, got)
	}
}

// mustNotContain fails when want is present in the rendered markup.
func mustNotContain(t *testing.T, got, want, what string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Errorf("%s: rendered markup must not contain %q\n%s", what, want, got)
	}
}

// mustContainsCount fails unless got contains want exactly n times.
func mustContainsCount(t *testing.T, got, want string, n int, what string) {
	t.Helper()
	if gotCount := strings.Count(got, want); gotCount != n {
		t.Errorf("%s: %q appears %d time(s), want %d\n%s", what, want, gotCount, n, got)
	}
}

// TestButtonVariants verifies every variant setter reaches the class
// list, since the setter writes a literal that Render turns into a modifier.
func TestButtonVariants(t *testing.T) {
	cases := []struct {
		name  string
		build func() *Button
		want  string
	}{
		{"default", func() *Button { return NewButton("t") }, "ui-btn-default"},
		{"primary", func() *Button { return NewButton("t").Primary() }, "ui-btn-primary"},
		{"secondary", func() *Button { return NewButton("t").Secondary() }, "ui-btn-secondary"},
		{"error", func() *Button { return NewButton("t").Error() }, "ui-btn-error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustContain(t, string(c.build().Render()), c.want, c.name)
		})
	}
}

// TestButtonSizes verifies the size modifiers, and that the medium
// default emits no size class because the stylesheet's base rule covers it.
func TestButtonSizes(t *testing.T) {
	mustNotContain(t, string(NewButton("t").Render()), "ui-btn-md", "default size")
	mustContain(t, string(NewButton("t").Small().Render()), "ui-btn-sm", "small")
	mustContain(t, string(NewButton("t").Large().Render()), "ui-btn-lg", "large")
}

// TestButtonModifiersAreComposable verifies WithIcon, Submit, WithClass,
// and the click handler all reach the markup, and that the handler is escaped.
func TestButtonModifiersAreComposable(t *testing.T) {
	out := string(NewButton("Save").
		WithIcon("disk").
		Submit().
		WithClass("w-full").
		WithClickHandler(`alert("x")`).
		Render())

	mustContain(t, out, `type="submit"`, "submit type")
	mustContain(t, out, "btn-icon", "icon wrapper")
	mustContain(t, out, "w-full", "extra class")
	mustContain(t, out, "onclick=", "click handler")
	mustNotContain(t, out, `alert("x")`, "the handler must be attribute-escaped")
}

// TestButtonEscapesTextAndIcon verifies user text cannot break out of
// the button content or the attribute values.
func TestButtonEscapesTextAndIcon(t *testing.T) {
	out := string(NewButton("<script>alert(1)</script>").WithIcon(`" onmouseover="x`).Render())
	mustNotContain(t, out, "<script>", "raw script tag")
	mustNotContain(t, out, `onmouseover="x"`, "attribute injection via icon")
	mustContain(t, out, "&lt;script&gt;", "escaped text")
}

// TestButtonEscapesCustomClass verifies a caller-supplied class cannot
// inject an attribute.
func TestButtonEscapesCustomClass(t *testing.T) {
	out := string(NewButton("t").WithClass(`x" onclick="alert(1)`).Render())
	mustNotContain(t, out, `onclick="alert(1)"`, "class attribute injection")
}

// TestBadgeVariants verifies every badge variant, including that
// `danger` is not a synonym for error.
func TestBadgeVariants(t *testing.T) {
	cases := []struct {
		name  string
		build func() *Badge
		want  string
	}{
		{"default", func() *Badge { return NewBadge("t") }, "ui-badge-default"},
		{"primary", func() *Badge { return NewBadge("t").Primary() }, "ui-badge-primary"},
		{"secondary", func() *Badge { return NewBadge("t").Secondary() }, "ui-badge-secondary"},
		{"success", func() *Badge { return NewBadge("t").Success() }, "ui-badge-success"},
		{"warning", func() *Badge { return NewBadge("t").Warning() }, "ui-badge-warning"},
		{"error", func() *Badge { return NewBadge("t").Error() }, "ui-badge-error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustContain(t, string(c.build().Render()), c.want, c.name)
		})
	}
	mustNotContain(t, string(NewBadge("t").Render()), "danger", "danger is not a variant")
}

// TestAlertVariantsCarryTheirIcon verifies each variant pairs with its
// documented glyph, since the pairing is what makes an alert readable at a glance.
func TestAlertVariantsCarryTheirIcon(t *testing.T) {
	cases := []struct {
		name    string
		build   func() *Alert
		variant string
		icon    string
	}{
		{"info default", func() *Alert { return NewAlert("m") }, "ui-alert-info", iconInfo},
		{"success", func() *Alert { return NewAlert("m").Success() }, "ui-alert-success", iconSuccess},
		{"warning", func() *Alert { return NewAlert("m").Warning() }, "ui-alert-warning", iconWarning},
		{"error", func() *Alert { return NewAlert("m").Error() }, "ui-alert-error", iconError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := string(c.build().Render())
			mustContain(t, out, c.variant, c.name+" variant")
			mustContain(t, out, c.icon, c.name+" icon")
		})
	}
}

// TestAlertEscapesMessage verifies the message cannot inject markup.
func TestAlertEscapesMessage(t *testing.T) {
	out := string(NewAlert(`<img src=x onerror=alert(1)>`).Render())
	mustNotContain(t, out, "<img", "raw img tag")
	mustContain(t, out, "&lt;img", "escaped message")
}

// TestCardSlotsAreOptional verifies each card section renders only
// when populated, so an empty card does not emit empty wrapper divs.
func TestCardSlotsAreOptional(t *testing.T) {
	bare := string(NewCard("").Render())
	mustNotContain(t, bare, "ui-card-header", "header without title")
	mustNotContain(t, bare, "ui-card-body", "body without content")
	mustNotContain(t, bare, "ui-card-footer", "footer without content")
	mustContain(t, bare, cardClassPrefix, "the root element is always present")

	full := string(NewCard("t").WithDesc("d").
		WithBody(NewText("b")).
		WithFooter(NewText("f")).
		WithActions(NewButton("a")).
		Render())
	for _, want := range []string{"ui-card-title", "ui-card-desc", "ui-card-body", "ui-card-footer"} {
		mustContain(t, full, want, "populated card")
	}
}

// TestCardEscapesTitleAndDescription verifies card text is escaped.
func TestCardEscapesTitleAndDescription(t *testing.T) {
	out := string(NewCard("<b>").WithDesc("<i>").Render())
	mustNotContain(t, out, "<b>", "raw title tag")
	mustNotContain(t, out, "<i>", "raw description tag")
}

// TestStackDirections verifies the direction drives the layout class
// and that a custom gap reaches the style attribute.
func TestStackDirections(t *testing.T) {
	mustContain(t, string(VStack().Render()), "ui-stack-v", "vertical stack")
	mustContain(t, string(HStack().Render()), "ui-stack-h", "horizontal stack")
	mustContain(t, string(VStack().WithGap("3rem").Render()), "gap:3rem", "custom gap")
	mustNotContain(t, string((&Stack{Direction: DirectionVertical}).Render()), "style=", "no gap, no style attribute")
}

// TestStackSkipsNilWidgets verifies a nil child is skipped rather than
// panicking, so a conditional child does not take the page down.
func TestStackSkipsNilWidgets(t *testing.T) {
	out := string(VStack(NewText("a"), nil, NewText("b")).Render())
	mustContain(t, out, "a", "first child")
	mustContain(t, out, "b", "child after the nil")
}

// TestGridColumnBounds verifies the supported column range and that an
// out-of-range value falls back rather than emitting an undefined class.
func TestGridColumnBounds(t *testing.T) {
	for columns, want := range map[int]string{2: "ui-grid-2", 3: "ui-grid-3", 4: "ui-grid-4"} {
		mustContain(t, string(NewGrid(columns, NewText("x")).Render()), want, "supported column count")
	}
	for _, columns := range []int{-1, 0, 1, 5, 99} {
		mustContain(t, string(NewGrid(columns, NewText("x")).Render()), "ui-grid-2", "out-of-range columns fall back to 2")
	}
}

// TestGridSkipsNilWidgets verifies nil children are skipped.
func TestGridSkipsNilWidgets(t *testing.T) {
	mustContain(t, string(NewGrid(2, nil, NewText("a"), nil).Render()), "a", "the non-nil child")
}

// TestHeadingLevels verifies valid levels pass through and out-of-range
// levels fall back, since <h0> and <h7> are not valid HTML.
func TestHeadingLevels(t *testing.T) {
	for level := minHeadingLevel; level <= maxHeadingLevel; level++ {
		mustContain(t, string(NewHeading(level, "t").Render()), headingClassPrefix, "heading class")
	}
	for _, level := range []int{0, -3, 7, 99} {
		out := string(NewHeading(level, "t").Render())
		mustNotContain(t, out, "<h0", "no <h0>")
		mustNotContain(t, out, "<h7", "no <h7>")
		mustNotContain(t, out, "<h9", "no <h9>")
		mustContain(t, out, "<h2", "out-of-range level falls back to 2")
	}
}

// TestHeadingEscapesContent verifies heading text is escaped.
func TestHeadingEscapesContent(t *testing.T) {
	mustNotContain(t, string(NewHeading(2, "<script>x</script>").Render()), "<script>", "raw script tag")
}

// TestDividerWithAndWithoutLabel verifies both branches of the divider,
// since the labelled branch assembles three elements and the unlabelled one.
func TestDividerWithAndWithoutLabel(t *testing.T) {
	bare := string(NewDivider().Render())
	mustContain(t, bare, "<hr", "the rule element")
	mustNotContain(t, bare, "<span", "no label element without a label")

	labelled := string(NewDivider("Section").Render())
	mustContain(t, labelled, "Section", "the label")
	mustContainsCount(t, labelled, "<hr", 2, "two rules around the label")

	mustContain(t, string(NewDivider("A", "B").Render()), "A", "only the first label is used")
}

// TestDividerEscapesLabel verifies the label cannot inject markup.
func TestDividerEscapesLabel(t *testing.T) {
	mustNotContain(t, string(NewDivider("<b>x</b>").Render()), "<b>", "raw tag in the label")
}

// TestModalRendersOptionalSections verifies the content and footer
// wrappers appear only when the corresponding widget is set.
func TestModalRendersOptionalSections(t *testing.T) {
	bare := string(NewModal("m", "Title").Render())
	mustContain(t, bare, modalClassPrefix, "the dialog element")
	mustContain(t, bare, "Title", "the title")
	mustNotContain(t, bare, "padding:1.25rem", "no body wrapper without content")
	mustNotContain(t, bare, "ui-surface", "no footer wrapper without a footer")

	full := string(NewModal("m", "Title").
		WithContent(NewText("body")).
		WithFooter(NewButton("ok")).
		Render())
	mustContain(t, full, "body", "content")
	mustContain(t, full, "ui-surface", "footer surface")
}

// TestModalEscapesIDAndTitle verifies the dialog id and title are
// escaped: the id lands inside an attribute and is used as a fragment target.
func TestModalEscapesIDAndTitle(t *testing.T) {
	out := string(NewModal(`m" onload="alert(1)`, "<script>x</script>").Render())
	mustNotContain(t, out, `onload="alert(1)"`, "attribute injection via the id")
	mustNotContain(t, out, "<script>", "raw script tag in the title")
}

// TestModalCloseButtonIsWired verifies the close affordance closes the
// dialog, since a modal that cannot be dismissed with a mouse is unusable.
func TestModalCloseButtonIsWired(t *testing.T) {
	mustContain(t, string(NewModal("m", "T").Render()), "this.closest('dialog').close()", "close handler")
}

// TestTableRendersHeadersRowsAndCells verifies the table structure,
// including that a header-less table omits the thead entirely.
func TestTableRendersHeadersRowsAndCells(t *testing.T) {
	full := string(NewTable("A", "B").AddTextRow("1", "2").Render())
	mustContain(t, full, "<thead>", "header section")
	mustContainsCount(t, full, "<th>", 2, "two header cells")
	mustContainsCount(t, full, "<td>", 2, "two body cells")

	headerless := string(NewTable().AddTextRow("1").Render())
	mustNotContain(t, headerless, "<thead>", "no header section without headers")
	mustContain(t, headerless, "<tbody>", "the body is always present")
}

// TestTableAddRowAcceptsPrebuiltHTML verifies AddRow passes widget HTML
// through unescaped, which is the documented difference from AddTextRow.
func TestTableAddRowAcceptsPrebuiltHTML(t *testing.T) {
	out := string(NewTable("A").AddRow(NewButton("go").Render()).Render())
	mustContain(t, out, "<button", "prebuilt widget HTML is not escaped again")
}

// TestTableAddTextRowEscapes verifies AddTextRow escapes its input, the
// counterpart to AddRow: a raw string must not become live markup.
func TestTableAddTextRowEscapes(t *testing.T) {
	out := string(NewTable("A").AddTextRow("<script>alert(1)</script>").Render())
	mustNotContain(t, out, "<script>", "raw script tag in a text cell")
	mustContain(t, out, "&lt;script&gt;", "escaped cell")
}

// TestTableEscapesHeaders verifies header text is escaped.
func TestTableEscapesHeaders(t *testing.T) {
	out := string(NewTable(`<th onclick="x">`).Render())
	mustNotContain(t, out, `onclick="x"`, "attribute injection via a header")
}

// TestTextModifiers verifies the three text renderings: plain, bold,
// and muted, including the nesting of bold inside muted.
func TestTextModifiers(t *testing.T) {
	mustContain(t, string(NewText("plain").Render()), "plain", "plain text")
	mustContain(t, string(NewText("b").Strong().Render()), "<strong>b</strong>", "bold text")
	mustContain(t, string(NewText("m").Subtle().Render()), "var(--ui-muted)", "muted text")

	both := string(NewText("both").Strong().Subtle().Render())
	mustContainsCount(t, both, "<strong>", 1, "one strong wrapper")
	mustContainsCount(t, both, "<span", 1, "one muted wrapper")
}

// TestTextEscapesContent verifies text content is escaped.
func TestTextEscapesContent(t *testing.T) {
	out := string(NewText("<script>alert(1)</script>").Render())
	mustNotContain(t, out, "<script>", "raw script tag")
}

// TestFragmentConcatenatesAndSkipsNil verifies Fragment joins the
// widgets in order and tolerates a nil entry.
func TestFragmentConcatenatesAndSkipsNil(t *testing.T) {
	out := string(Fragment(NewText("a"), nil, NewText("b")))
	if strings.Index(out, "a") > strings.Index(out, "b") {
		t.Errorf("Fragment reordered its children: %s", out)
	}
	mustContain(t, out, "a", "first child")
	mustContain(t, out, "b", "second child")
	if out := string(Fragment()); out != "" {
		t.Errorf("Fragment() = %q, want empty", out)
	}
}

// TestHTMLWidgetIsVerbatim documents that HTML is trusted content: it
// is returned untouched, which is why SafeURL and the field renderers must escape
// their own input.
func TestHTMLWidgetIsVerbatim(t *testing.T) {
	mustContain(t, string(HTML("<b>raw</b>").Render()), "<b>raw</b>", "HTML is passed through")
}

// TestModifierHelperHandlesEmptyName verifies the class builder never
// emits a dangling prefix.
func TestModifierHelperHandlesEmptyName(t *testing.T) {
	if got, want := modifier("ui-x", ""), "ui-x"; got != want {
		t.Errorf("modifier with an empty name = %q, want %q", got, want)
	}
	if got, want := modifier("ui-x", "y"), "ui-x-y"; got != want {
		t.Errorf("modifier = %q, want %q", got, want)
	}
}
