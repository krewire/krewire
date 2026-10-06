package form

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mustContain(t *testing.T, got, want, what string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s: missing %q\n%s", what, want, got)
	}
}

func mustNotContain(t *testing.T, got, want, what string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Errorf("%s: must not contain %q\n%s", what, want, got)
	}
}

// TestFORM_001_BaseFieldAccessors verifies the name and label reach the markup
// and the accessors, since both drive the label/control association.
func TestFORM_001_BaseFieldAccessors(t *testing.T) {
	f := NewText("username", "User Name")
	if f.Name() != "username" {
		t.Errorf("Name() = %q, want username", f.Name())
	}
	if f.Label() != "User Name" {
		t.Errorf("Label() = %q, want User Name", f.Label())
	}
	out := string(f.Render("", nil))
	mustContain(t, out, `for="field-username"`, "label target")
	mustContain(t, out, `id="field-username"`, "control id")
	mustContain(t, out, `name="username"`, "control name")
}

// TestFORM_002_RequiredMarkerAndAttribute verifies a required field carries both
// the visual marker and the required attribute: the marker alone is not announced.
func TestFORM_002_RequiredMarkerAndAttribute(t *testing.T) {
	out := string(NewText("u", "User").MakeRequired().Render("", nil))
	mustContain(t, out, fieldRequiredCls, "required marker")
	mustContain(t, out, " required", "required attribute")

	mustNotContain(t, string(NewText("u", "User").Render("", nil)), " required", "optional field")
}

// TestFORM_003_PlaceholderAndHelpReachMarkup verifies the two optional text
// slots render only when set.
func TestFORM_003_PlaceholderAndHelpReachMarkup(t *testing.T) {
	bare := string(NewText("u", "User").Render("", nil))
	mustNotContain(t, bare, "placeholder=", "no placeholder when unset")
	mustNotContain(t, bare, "forge-field-help", "no help when unset")

	full := string(NewText("u", "User").WithPlaceholder("you@example.com").WithHelp("We never share this").Render("", nil))
	mustContain(t, full, `placeholder="you@example.com"`, "placeholder")
	mustContain(t, full, "We never share this", "help text")
}

// TestFORM_004_EmailValidation verifies the email validator accepts a valid
// address, rejects a malformed one, and treats an empty value as the required
// check's business rather than an email failure.
func TestFORM_004_EmailValidation(t *testing.T) {
	f := NewText("email", "Email").AsEmail()
	if got := f.InputType; got != "email" {
		t.Errorf("InputType = %q, want email", got)
	}
	if errs := f.Validate("alex@example.com"); len(errs) != 0 {
		t.Errorf("valid address rejected: %v", errs)
	}
	if errs := f.Validate("not-an-address"); len(errs) == 0 {
		t.Error("malformed address accepted")
	}
	if errs := f.Validate(""); len(errs) != 0 {
		t.Errorf("empty value produced email errors: %v", errs)
	}
	// Required plus empty must report the required error, not the email one.
	required := NewText("email", "Email").AsEmail().MakeRequired()
	errs := required.Validate("")
	if len(errs) != 1 || !strings.Contains(errs[0], "required") {
		t.Errorf("required empty field errors = %v, want a single required error", errs)
	}
}

// TestFORM_005_PasswordAndMinLength verifies the password input type and the
// minimum-length rule, including that an empty value skips the length check so a
// required field is not double-reported.
func TestFORM_005_PasswordAndMinLength(t *testing.T) {
	if got := NewText("p", "Password").AsPassword().InputType; got != "password" {
		t.Errorf("InputType = %q, want password", got)
	}
	f := NewText("u", "User").MinLength(3)
	if errs := f.Validate("ab"); len(errs) == 0 {
		t.Error("short value accepted")
	}
	if errs := f.Validate("abc"); len(errs) != 0 {
		t.Errorf("value at the minimum rejected: %v", errs)
	}
	if errs := f.Validate(""); len(errs) != 0 {
		t.Errorf("empty value produced a length error: %v", errs)
	}
}

// TestFORM_006_NumberBounds verifies the numeric validator rejects non-numeric
// input and values outside the configured bounds, and that the bounds reach the
// rendered min and max attributes.
func TestFORM_006_NumberBounds(t *testing.T) {
	f := NewNumber("age", "Age").SetMin(18).SetMax(65)

	if errs := f.Validate("not a number"); len(errs) == 0 {
		t.Error("non-numeric input accepted")
	}
	if errs := f.Validate("17"); len(errs) == 0 {
		t.Error("value below the minimum accepted")
	}
	if errs := f.Validate("66"); len(errs) == 0 {
		t.Error("value above the maximum accepted")
	}
	if errs := f.Validate("18"); len(errs) != 0 {
		t.Errorf("value at the minimum rejected: %v", errs)
	}
	if errs := f.Validate("65"); len(errs) != 0 {
		t.Errorf("value at the maximum rejected: %v", errs)
	}

	out := string(f.Render("30", nil))
	mustContain(t, out, `type="number"`, "numeric input type")
	mustContain(t, out, `min="18"`, "min attribute")
	mustContain(t, out, `max="65"`, "max attribute")
}

// TestFORM_007_NumberBoundsAreOptional verifies min and max are emitted only when
// configured, so an unconstrained number field has no bogus attributes.
func TestFORM_007_NumberBoundsAreOptional(t *testing.T) {
	bare := string(NewNumber("n", "N").Render("1", nil))
	mustNotContain(t, bare, "min=", "no min attribute when unset")
	mustNotContain(t, bare, "max=", "no max attribute when unset")
	mustContain(t, string(NewNumber("n", "N").SetMax(5).Render("", nil)), `max="5"`, "max only")
}

// TestFORM_008_TextareaRows verifies the row count reaches the markup, since a
// one-line textarea defeats the purpose of the control.
func TestFORM_008_TextareaRows(t *testing.T) {
	out := string(NewTextarea("bio", "Bio").Render("hello", nil))
	mustContain(t, out, "<textarea", "the textarea element")
	mustContain(t, out, "hello", "the current value")
	mustContain(t, out, "rows=", "a row count")
}

// TestFORM_009_SelectMarksTheCurrentValue verifies exactly one option is selected
// and it is the one matching the submitted value.
func TestFORM_009_SelectMarksTheCurrentValue(t *testing.T) {
	out := string(NewSelect("plan", "Plan",
		SelectOption{Value: "free", Label: "Free"},
		SelectOption{Value: "pro", Label: "Pro"},
	).Render("pro", nil))

	mustContain(t, out, `<option value="pro" selected>`, "the matching option is selected")
	mustNotContain(t, out, `<option value="free" selected>`, "no other option is selected")
	mustContainsOnce(t, out, "selected", 1, "exactly one selected option")
}

// mustContainsOnce fails unless want appears exactly n times.
func mustContainsOnce(t *testing.T, got, want string, n int, what string) {
	t.Helper()
	if c := strings.Count(got, want); c != n {
		t.Errorf("%s: %q appears %d time(s), want %d\n%s", what, want, c, n, got)
	}
}

// TestFORM_010_SelectWithNoMatchingValue verifies a submitted value that matches
// no option leaves every option unselected rather than selecting the first.
func TestFORM_010_SelectWithNoMatchingValue(t *testing.T) {
	out := string(NewSelect("plan", "Plan", SelectOption{Value: "free", Label: "Free"}).Render("enterprise", nil))
	mustNotContain(t, out, "selected", "no option matches the submitted value")
}

// TestFORM_011_CheckboxTruthyValues verifies the submitted values a browser sends
// for a checked box, and that anything else renders unchecked.
func TestFORM_011_CheckboxTruthyValues(t *testing.T) {
	for _, value := range []string{"true", "on", "1"} {
		out := string(NewCheckbox("agree", "Agree").Render(value, nil))
		mustContain(t, out, "checked", "value "+value+" must render checked")
	}
	for _, value := range []string{"", "false", "0", "yes", "TRUE"} {
		out := string(NewCheckbox("agree", "Agree").Render(value, nil))
		mustNotContain(t, out, "checked", "value "+value+" must render unchecked")
	}
}

// TestFORM_012_CheckboxRendersOneLabel verifies the inline checkbox emits exactly
// one label, since a duplicated label is announced twice by a screen reader.
func TestFORM_012_CheckboxRendersOneLabel(t *testing.T) {
	out := string(NewCheckbox("agree", "Agree").Render("", nil))
	mustContainsOnce(t, out, "<label", 1, "exactly one label")
	mustContainsOnce(t, out, "Agree", 1, "the label text appears once")
}

// TestFORM_013_FieldsEscapeNamesLabelsAndValues verifies every user-controlled
// string reaches the markup escaped, so a form cannot be used to inject markup.
func TestFORM_013_FieldsEscapeNamesLabelsAndValues(t *testing.T) {
	attack := `<script>alert(1)</script>`
	renders := map[string]string{
		"text":     string(NewText(attack, attack).WithPlaceholder(attack).WithHelp(attack).Render(attack, []string{attack})),
		"number":   string(NewNumber(attack, attack).Render(attack, nil)),
		"textarea": string(NewTextarea(attack, attack).Render(attack, []string{attack})),
		"select":   string(NewSelect(attack, attack, SelectOption{Value: attack, Label: attack}).Render(attack, nil)),
		"checkbox": string(NewCheckbox(attack, attack).Render("", []string{attack})),
	}
	for name, out := range renders {
		if strings.Contains(out, "<script>") {
			t.Errorf("%s field emitted a raw script tag:\n%s", name, out)
		}
	}
}

// TestFORM_014_ErrorsRenderEscaped verifies validation messages cannot inject
// markup, since they can embed the rejected value.
func TestFORM_014_ErrorsRenderEscaped(t *testing.T) {
	out := string(NewText("u", "U").Render("", []string{"<b>bad</b>"}))
	mustNotContain(t, out, "<b>", "raw markup in an error message")
	mustContainsOnce(t, out, fieldErrorClass, 1, "the error span")
}

// TestFORM_015_ValidateCollectsPerFieldErrors verifies Validate reports the
// offending field keys and returns false, and clears stale errors on a later
// successful pass.
func TestFORM_015_ValidateCollectsPerFieldErrors(t *testing.T) {
	f := New("/x", "POST").
		Add(NewText("username", "Username").MakeRequired()).
		Add(NewNumber("age", "Age").SetMin(18))

	if f.Validate() {
		t.Fatal("Validate() = true for an empty required form")
	}
	if got := f.Errors()["username"]; len(got) == 0 {
		t.Error("no error recorded for the required username field")
	}

	f.SetValue("username", "alex").SetValue("age", "30")
	if !f.Validate() {
		t.Errorf("Validate() = false for a valid form: %v", f.Errors())
	}
	if len(f.Errors()) != 0 {
		t.Errorf("stale errors survived a successful validation: %v", f.Errors())
	}
}

// TestFORM_016_BindRequestReadsQueryAndForm verifies BindRequest parses the
// request and binds the values, which is the entry point a real handler uses.
func TestFORM_016_BindRequestReadsQueryAndForm(t *testing.T) {
	f := New("/login", "POST").Add(NewText("user", "User").MakeRequired())

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("user=alex&pass=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := f.BindRequest(req); err != nil {
		t.Fatal(err)
	}
	if got := f.Value("user"); got != "alex" {
		t.Errorf("Value(user) = %q, want alex", got)
	}
	if !f.Validate() {
		t.Errorf("bound form failed validation: %v", f.Errors())
	}
}

// TestFORM_017_BindRequestRejectsUnparsableBody verifies a malformed body is
// reported rather than silently binding an empty form that then fails validation.
func TestFORM_017_BindRequestRejectsUnparsableBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("%zz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := New("/x", "POST").BindRequest(req); err == nil {
		t.Error("BindRequest = nil for an unparsable body, want an error")
	}
}

// TestFORM_018_ValuesReturnsACopy verifies Values hands back a copy, so a caller
// mutating it cannot corrupt the form's state.
func TestFORM_018_ValuesReturnsACopy(t *testing.T) {
	f := New("/x", "POST")
	f.SetValue("a", "1")
	snapshot := f.Values()
	snapshot["a"] = "tampered"
	if got := f.Value("a"); got != "1" {
		t.Errorf("Value(a) = %q, want 1: Values must return a copy", got)
	}
}

// TestFORM_019_BindPrefersFirstValue verifies Bind takes the first value for a
// repeated key rather than an arbitrary one.
func TestFORM_019_BindPrefersFirstValue(t *testing.T) {
	f := New("/x", "POST")
	f.Bind(url.Values{"a": {"first", "second"}})
	if got := f.Value("a"); got != "first" {
		t.Errorf("Value(a) = %q, want first", got)
	}
}

// TestFORM_020_FormDefaultsToPostAndEscapesAction verifies the default method and
// that the action is escaped rather than injected.
func TestFORM_020_FormDefaultsToPostAndEscapesAction(t *testing.T) {
	out := string(New("", "").Render())
	mustContain(t, out, `method="POST"`, "the default method is POST")

	out = string(New(`"><script>alert(1)</script>`, "POST").Render())
	mustNotContain(t, out, "<script>", "raw script tag in the action")
}

// TestFORM_021_FieldIDIsDerivedFromTheName verifies the id helper, since the label
// and the control must agree on it for the association to work.
func TestFORM_021_FieldIDIsDerivedFromTheName(t *testing.T) {
	if got, want := fieldID("user"), "field-user"; got != want {
		t.Errorf("fieldID = %q, want %q", got, want)
	}
	mustNotContain(t, fieldID(`"><script>`), "<script>", "the id must be escaped")
}
