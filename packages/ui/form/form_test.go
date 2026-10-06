package form

import (
	"net/url"
	"strings"
	"testing"
)

func TestFormValidation(t *testing.T) {
	f := New("/submit", "POST").
		Add(NewText("username", "Username").MakeRequired().MinLength(3)).
		Add(NewText("email", "Email").MakeRequired().AsEmail()).
		Add(NewNumber("age", "Age").SetMin(18)).
		WithSubmitLabel("Save Account")

	// Empty form should fail validation
	if f.Validate() {
		t.Error("expected validation to fail for empty required fields")
	}
	if len(f.Errors()["username"]) == 0 {
		t.Error("expected error for required username")
	}

	// Valid form data
	data := url.Values{
		"username": {"alex99"},
		"email":    {"alex@example.com"},
		"age":      {"25"},
	}
	f.Bind(data)
	if !f.Validate() {
		t.Errorf("expected form to be valid, got errors: %v", f.Errors())
	}
	if f.Value("username") != "alex99" {
		t.Errorf("expected username alex99, got %s", f.Value("username"))
	}

	// Render output
	html := string(f.Render())
	if !strings.Contains(html, `<form class="forge-form" action="/submit" method="POST">`) {
		t.Errorf("form tag wrong: %s", html)
	}
	if !strings.Contains(html, "Save Account") {
		t.Errorf("missing submit button label in: %s", html)
	}
}

func TestSelectAndCheckboxFields(t *testing.T) {
	f := New("/prefs", "POST").
		Add(NewSelect("plan", "Subscription Plan",
			SelectOption{Value: "free", Label: "Free"},
			SelectOption{Value: "pro", Label: "Pro"},
		)).
		Add(NewCheckbox("newsletter", "Subscribe to newsletter"))

	f.SetValue("plan", "pro")
	f.SetValue("newsletter", "true")

	html := string(f.Render())
	if !strings.Contains(html, `value="pro" selected`) {
		t.Errorf("expected pro option to be selected in: %s", html)
	}
	if !strings.Contains(html, `checked`) {
		t.Errorf("expected checkbox to be checked in: %s", html)
	}
}
