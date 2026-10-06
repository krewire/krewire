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

// TestWIDGET_001_ButtonVariants verifies every variant setter reaches the class
// list, since the setter writes a literal that Render turns into a modifier.
func TestWIDGET_001_ButtonVariants(t *testing.T) {
	cases := []struct {
		name  string
		build func() *Button
		want  string
	}{
		{"default", func() *Button { return NewButton("t") }, "forge-btn-default"},
		{"primary", func() *Button { return NewButton("t").Primary() }, "forge-btn-primary"},
		{"secondary", func() *Button { return NewButton("t").Secondary() }, "forge-btn-secondary"},
		{"error", func() *Button { return NewButton("t").Error() }, "forge-btn-error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustContain(t, string(c.build().Render()), c.want, c.name)
		})
	}
}

// TestWIDGET_002_ButtonSizes verifies the size modifiers, and that the medium
// default emits no size class because the stylesheet's base rule covers it.
func TestWIDGET_002_ButtonSizes(t *testing.T) {
	mustNotContain(t, string(NewButton("t").Render()), "forge-btn-md", "default size")
	mustContain(t, string(NewButton("t").Small().Render()), "forge-btn-sm", "small")
	mustContain(t, string(NewButton("t").Large().Render()), "forge-btn-lg", "large")
}

// TestWIDGET_003_ButtonModifiersAreComposable verifies WithIcon, Submit, WithClass,
// and the click handler all reach the markup, and that the handler is escaped.
func TestWIDGET_003_ButtonModifiersAreComposable(t *testing.T) {
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

// TestWIDGET_004_ButtonEscapesTextAndIcon verifies user text cannot break out of
// the button content or the attribute values.
func TestWIDGET_004_ButtonEscapesTextAndIcon(t *testing.T) {
	out := string(NewButton("<script>alert(1)</script>").WithIcon(`" onmouseover="x`).Render())
	mustNotContain(t, out, "<script>", "raw script tag")
	mustNotContain(t, out, `onmouseover="x"`, "attribute injection via icon")
	mustContain(t, out, "&lt;script&gt;", "escaped text")
}

// TestWIDGET_005_ButtonEscapesCustomClass verifies a caller-supplied class cannot
// inject an attribute.
func TestWIDGET_005_ButtonEscapesCustomClass(t *testing.T) {
	out := string(NewButton("t").WithClass(`x" onclick="alert(1)`).Render())
	mustNotContain(t, out, `onclick="alert(1)"`, "class attribute injection")
}

// TestWIDGET_006_BadgeVariants verifies every badge variant, including that
// `danger` is not a synonym for error.
func TestWIDGET_006_BadgeVariants(t *testing.T) {
	cases := []struct {
		name  string
		build func() *Badge
		want  string
	}{
		{"default", func() *Badge { return NewBadge("t") }, "forge-badge-default"},
		{"primary", func() *Badge { return NewBadge("t").Primary() }, "forge-badge-primary"},
		{"secondary", func() *Badge { return NewBadge("t").Secondary() }, "forge-badge-secondary"},
		{"success", func() *Badge { return NewBadge("t").Success() }, "forge-badge-success"},
		{"warning", func() *Badge { return NewBadge("t").Warning() }, "forge-badge-warning"},
		{"error", func() *Badge { return NewBadge("t").Error() }, "forge-badge-error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustContain(t, string(c.build().Render()), c.want, c.name)
		})
	}
	mustNotContain(t, string(NewBadge("t").Render()), "danger", "danger is not a variant")
}

// TestWIDGET_007_AlertVariantsCarryTheirIcon verifies each variant pairs with its
// documented glyph, since the pairing is what makes an alert readable at a glance.
func TestWIDGET_007_AlertVariantsCarryTheirIcon(t *testing.T) {
	cases := []struct {
		name    string
		build   func() *Alert
		variant string
		icon    string
	}{
		{"info default", func() *Alert { return NewAlert("m") }, "forge-alert-info", iconInfo},
		{"success", func() *Alert { return NewAlert("m").Success() }, "forge-alert-success", iconSuccess},
		{"warning", func() *Alert { return NewAlert("m").Warning() }, "forge-alert-warning", iconWarning},
		{"error", func() *Alert { return NewAlert("m").Error() }, "forge-alert-error", iconError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := string(c.build().Render())
			mustContain(t, out, c.variant, c.name+" variant")
			mustContain(t, out, c.icon, c.name+" icon")
		})
	}
}

// TestWIDGET_008_AlertEscapesMessage verifies the message cannot inject markup.
func TestWIDGET_008_AlertEscapesMessage(t *testing.T) {
	out := string(NewAlert(`<img src=x onerror=alert(1)>`).Render())
	mustNotContain(t, out, "<img", "raw img tag")
	mustContain(t, out, "&lt;img", "escaped message")
}

// TestWIDGET_009_CardSlotsAreOptional verifies each card section renders only
// when populated, so an empty card does not emit empty wrapper divs.
func TestWIDGET_009_CardSlotsAreOptional(t *testing.T) {
	bare := string(NewCard("").Render())
	mustNotContain(t, bare, "forge-card-header", "header without title")
	mustNotContain(t, bare, "forge-card-body", "body without content")
	mustNotContain(t, bare, "forge-card-footer", "footer without content")
	mustContain(t, bare, cardClassPrefix, "the root element is always present")

	full := string(NewCard("t").WithDesc("d").
		WithBody(NewText("b")).
		WithFooter(NewText("f")).
		WithActions(NewButton("a")).
		Render())
	for _, want := range []string{"forge-card-title", "forge-card-desc", "forge-card-body", "forge-card-footer"} {
		mustContain(t, full, want, "populated card")
	}
}

// TestWIDGET_010_CardEscapesTitleAndDescription verifies card text is escaped.
func TestWIDGET_010_CardEscapesTitleAndDescription(t *testing.T) {
	out := string(NewCard("<b>").WithDesc("<i>").Render())
	mustNotContain(t, out, "<b>", "raw title tag")
	mustNotContain(t, out, "<i>", "raw description tag")
}

// TestWIDGET_011_StackDirections verifies the direction drives the layout class
// and that a custom gap reaches the style attribute.
func TestWIDGET_011_StackDirections(t *testing.T) {
	mustContain(t, string(VStack().Render()), "forge-stack-v", "vertical stack")
	mustContain(t, string(HStack().Render()), "forge-stack-h", "horizontal stack")
	mustContain(t, string(VStack().WithGap("3rem").Render()), "gap:3rem", "custom gap")
	mustNotContain(t, string((&Stack{Direction: DirectionVertical}).Render()), "style=", "no gap, no style attribute")
}

// TestWIDGET_012_StackSkipsNilWidgets verifies a nil child is skipped rather than
// panicking, so a conditional child does not take the page down.
func TestWIDGET_012_StackSkipsNilWidgets(t *testing.T) {
	out := string(VStack(NewText("a"), nil, NewText("b")).Render())
	mustContain(t, out, "a", "first child")
	mustContain(t, out, "b", "child after the nil")
}

// TestWIDGET_013_GridColumnBounds verifies the supported column range and that an
// out-of-range value falls back rather than emitting an undefined class.
func TestWIDGET_013_GridColumnBounds(t *testing.T) {
	for columns, want := range map[int]string{2: "forge-grid-2", 3: "forge-grid-3", 4: "forge-grid-4"} {
		mustContain(t, string(NewGrid(columns, NewText("x")).Render()), want, "supported column count")
	}
	for _, columns := range []int{-1, 0, 1, 5, 99} {
		mustContain(t, string(NewGrid(columns, NewText("x")).Render()), "forge-grid-2", "out-of-range columns fall back to 2")
	}
}

// TestWIDGET_014_GridSkipsNilWidgets verifies nil children are skipped.
func TestWIDGET_014_GridSkipsNilWidgets(t *testing.T) {
	mustContain(t, string(NewGrid(2, nil, NewText("a"), nil).Render()), "a", "the non-nil child")
}

// TestWIDGET_015_HeadingLevels verifies valid levels pass through and out-of-range
// levels fall back, since <h0> and <h7> are not valid HTML.
func TestWIDGET_015_HeadingLevels(t *testing.T) {
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

// TestWIDGET_016_HeadingEscapesContent verifies heading text is escaped.
func TestWIDGET_016_HeadingEscapesContent(t *testing.T) {
	mustNotContain(t, string(NewHeading(2, "<script>x</script>").Render()), "<script>", "raw script tag")
}

// TestWIDGET_017_DividerWithAndWithoutLabel verifies both branches of the divider,
// since the labelled branch assembles three elements and the unlabelled one.
func TestWIDGET_017_DividerWithAndWithoutLabel(t *testing.T) {
	bare := string(NewDivider().Render())
	mustContain(t, bare, "<hr", "the rule element")
	mustNotContain(t, bare, "<span", "no label element without a label")

	labelled := string(NewDivider("Section").Render())
	mustContain(t, labelled, "Section", "the label")
	mustContainsCount(t, labelled, "<hr", 2, "two rules around the label")

	mustContain(t, string(NewDivider("A", "B").Render()), "A", "only the first label is used")
}

// TestWIDGET_018_DividerEscapesLabel verifies the label cannot inject markup.
func TestWIDGET_018_DividerEscapesLabel(t *testing.T) {
	mustNotContain(t, string(NewDivider("<b>x</b>").Render()), "<b>", "raw tag in the label")
}

// TestWIDGET_019_ModalRendersOptionalSections verifies the content and footer
// wrappers appear only when the corresponding widget is set.
func TestWIDGET_019_ModalRendersOptionalSections(t *testing.T) {
	bare := string(NewModal("m", "Title").Render())
	mustContain(t, bare, modalClassPrefix, "the dialog element")
	mustContain(t, bare, "Title", "the title")
	mustNotContain(t, bare, "padding:1.25rem", "no body wrapper without content")
	mustNotContain(t, bare, "forge-surface", "no footer wrapper without a footer")

	full := string(NewModal("m", "Title").
		WithContent(NewText("body")).
		WithFooter(NewButton("ok")).
		Render())
	mustContain(t, full, "body", "content")
	mustContain(t, full, "forge-surface", "footer surface")
}

// TestWIDGET_020_ModalEscapesIDAndTitle verifies the dialog id and title are
// escaped: the id lands inside an attribute and is used as a fragment target.
func TestWIDGET_020_ModalEscapesIDAndTitle(t *testing.T) {
	out := string(NewModal(`m" onload="alert(1)`, "<script>x</script>").Render())
	mustNotContain(t, out, `onload="alert(1)"`, "attribute injection via the id")
	mustNotContain(t, out, "<script>", "raw script tag in the title")
}

// TestWIDGET_021_ModalCloseButtonIsWired verifies the close affordance closes the
// dialog, since a modal that cannot be dismissed with a mouse is unusable.
func TestWIDGET_021_ModalCloseButtonIsWired(t *testing.T) {
	mustContain(t, string(NewModal("m", "T").Render()), "this.closest('dialog').close()", "close handler")
}

// TestWIDGET_022_TableRendersHeadersRowsAndCells verifies the table structure,
// including that a header-less table omits the thead entirely.
func TestWIDGET_022_TableRendersHeadersRowsAndCells(t *testing.T) {
	full := string(NewTable("A", "B").AddTextRow("1", "2").Render())
	mustContain(t, full, "<thead>", "header section")
	mustContainsCount(t, full, "<th>", 2, "two header cells")
	mustContainsCount(t, full, "<td>", 2, "two body cells")

	headerless := string(NewTable().AddTextRow("1").Render())
	mustNotContain(t, headerless, "<thead>", "no header section without headers")
	mustContain(t, headerless, "<tbody>", "the body is always present")
}

// TestWIDGET_023_TableAddRowAcceptsPrebuiltHTML verifies AddRow passes widget HTML
// through unescaped, which is the documented difference from AddTextRow.
func TestWIDGET_023_TableAddRowAcceptsPrebuiltHTML(t *testing.T) {
	out := string(NewTable("A").AddRow(NewButton("go").Render()).Render())
	mustContain(t, out, "<button", "prebuilt widget HTML is not escaped again")
}

// TestWIDGET_024_TableAddTextRowEscapes verifies AddTextRow escapes its input, the
// counterpart to AddRow: a raw string must not become live markup.
func TestWIDGET_024_TableAddTextRowEscapes(t *testing.T) {
	out := string(NewTable("A").AddTextRow("<script>alert(1)</script>").Render())
	mustNotContain(t, out, "<script>", "raw script tag in a text cell")
	mustContain(t, out, "&lt;script&gt;", "escaped cell")
}

// TestWIDGET_025_TableEscapesHeaders verifies header text is escaped.
func TestWIDGET_025_TableEscapesHeaders(t *testing.T) {
	out := string(NewTable(`<th onclick="x">`).Render())
	mustNotContain(t, out, `onclick="x"`, "attribute injection via a header")
}

// TestWIDGET_026_TextModifiers verifies the three text renderings: plain, bold,
// and muted, including the nesting of bold inside muted.
func TestWIDGET_026_TextModifiers(t *testing.T) {
	mustContain(t, string(NewText("plain").Render()), "plain", "plain text")
	mustContain(t, string(NewText("b").Strong().Render()), "<strong>b</strong>", "bold text")
	mustContain(t, string(NewText("m").Subtle().Render()), "var(--forge-muted)", "muted text")

	both := string(NewText("both").Strong().Subtle().Render())
	mustContainsCount(t, both, "<strong>", 1, "one strong wrapper")
	mustContainsCount(t, both, "<span", 1, "one muted wrapper")
}

// TestWIDGET_027_TextEscapesContent verifies text content is escaped.
func TestWIDGET_027_TextEscapesContent(t *testing.T) {
	out := string(NewText("<script>alert(1)</script>").Render())
	mustNotContain(t, out, "<script>", "raw script tag")
}

// TestWIDGET_028_FragmentConcatenatesAndSkipsNil verifies Fragment joins the
// widgets in order and tolerates a nil entry.
func TestWIDGET_028_FragmentConcatenatesAndSkipsNil(t *testing.T) {
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

// TestWIDGET_029_HTMLWidgetIsVerbatim documents that HTML is trusted content: it
// is returned untouched, which is why SafeURL and the field renderers must escape
// their own input.
func TestWIDGET_029_HTMLWidgetIsVerbatim(t *testing.T) {
	mustContain(t, string(HTML("<b>raw</b>").Render()), "<b>raw</b>", "HTML is passed through")
}

// TestWIDGET_030_ModifierHelperHandlesEmptyName verifies the class builder never
// emits a dangling prefix.
func TestWIDGET_030_ModifierHelperHandlesEmptyName(t *testing.T) {
	if got, want := modifier("forge-x", ""), "forge-x"; got != want {
		t.Errorf("modifier with an empty name = %q, want %q", got, want)
	}
	if got, want := modifier("forge-x", "y"), "forge-x-y"; got != want {
		t.Errorf("modifier = %q, want %q", got, want)
	}
}
