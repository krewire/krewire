package validation

import (
	"errors"
	"strings"
	"testing"
)

type simple struct {
	Name  string `validate:"required"`
	Email string `validate:"email"`
	Age   int    `validate:"min=0,max=120"`
}

type optional struct {
	Nick  string   `validate:"omitempty,min=3"`
	Rate  float64  `validate:"omitempty,min=0.5,max=1.5"`
	Tags  []string `validate:"omitempty"`
	Score int      `validate:"omitempty,oneof=1 2 3"`
}

type nested struct {
	User struct {
		Email string `validate:"required,email"`
	} `validate:"required"`
	Code string `validate:"len=4"`
}

func TestStructRequired(t *testing.T) {
	tests := []struct {
		name  string
		value simple
		want  []string
	}{
		{"valid", simple{Name: "krewire", Email: "a@b.c", Age: 30}, nil},
		{"empty name", simple{Email: "a@b.c", Age: 1}, []string{"Name"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Struct(&tt.value)
			if len(tt.want) == 0 {
				if err != nil {
					t.Fatalf("Struct: unexpected error: %v", err)
				}
				return
			}
			ve := (*ValidationError)(nil)
			if !errors.As(err, &ve) {
				t.Fatalf("Struct: want *ValidationError, got %T (%v)", err, err)
			}
			var got []string
			for _, f := range ve.Fields {
				got = append(got, f.Field)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("fields = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldEmail(t *testing.T) {
	if err := Field("a@b.c", "email"); err != nil {
		t.Errorf("valid email failed: %v", err)
	}
	if err := Field("nope", "email"); err == nil {
		t.Error("invalid email must fail")
	}
}

func TestFieldRequired(t *testing.T) {
	if err := Field("", "required"); err == nil {
		t.Error("empty string must fail required")
	}
	if err := Field(0, "required"); err == nil {
		t.Error("zero int must fail required")
	}
	if err := Field(false, "required"); err == nil {
		t.Error("false must fail required")
	}
	if err := Field(nil, "required"); err == nil {
		t.Error("nil must fail required")
	}
	if err := Field([]string{}, "required"); err == nil {
		t.Error("empty slice must fail required")
	}
	if err := Field("ok", "required"); err != nil {
		t.Errorf("non-zero must pass: %v", err)
	}
}

func TestFieldMinMax(t *testing.T) {
	tests := []struct {
		value any
		tag   string
		pass  bool
	}{
		{1, "min=1", true},
		{0, "min=1", false},
		{120, "max=120", true},
		{121, "max=120", false},
		{1.5, "max=1.5", true},
		{1.6, "max=1.5", false},
		{"abc", "min=3", true},
		{"ab", "min=3", false},
		{"abcdef", "max=5", false},
	}
	for _, tt := range tests {
		err := Field(tt.value, tt.tag)
		if tt.pass && err != nil {
			t.Errorf("Field(%v, %q) failed: %v", tt.value, tt.tag, err)
		}
		if !tt.pass && err == nil {
			t.Errorf("Field(%v, %q) must fail", tt.value, tt.tag)
		}
	}
}

func TestFieldLen(t *testing.T) {
	if err := Field([]string{"a", "b"}, "len=2"); err != nil {
		t.Errorf("slice len=2 failed: %v", err)
	}
	if err := Field(map[string]int{"k": 1}, "len=2"); err == nil {
		t.Error("map len=2 with one entry must fail")
	}
	if err := Field("abcd", "len=4"); err != nil {
		t.Errorf("string len=4 failed: %v", err)
	}
	if err := Field(3, "len=3"); err == nil {
		t.Error("int must not satisfy len (type error)")
	}
}

func TestFieldPattern(t *testing.T) {
	if err := Field("abc123", "pattern=^[a-z0-9]+$"); err != nil {
		t.Errorf("matching pattern failed: %v", err)
	}
	if err := Field("ABC", "pattern=^[a-z]+$"); err == nil {
		t.Error("non-matching pattern must fail")
	}
}

func TestFieldOneof(t *testing.T) {
	if err := Field("read", "oneof=read write"); err != nil {
		t.Errorf("oneof string failed: %v", err)
	}
	if err := Field("execute", "oneof=read write"); err == nil {
		t.Error("non-listed string must fail")
	}
	if err := Field(2, "oneof=1 2 3"); err != nil {
		t.Errorf("oneof int failed: %v", err)
	}
	if err := Field(9, "oneof=1 2 3"); err == nil {
		t.Error("non-listed int must fail")
	}
}

func TestOmitEmptySkipsZero(t *testing.T) {
	if err := Struct(&optional{}); err != nil {
		t.Errorf("zero struct with omitempty must pass: %v", err)
	}
	o := optional{Nick: "ab"}
	if err := Struct(&o); err == nil {
		t.Error("omitempty must still enforce rules for non-zero values")
	}
}

func TestCompositionFirstFailingRuleReports(t *testing.T) {
	err := Field("", "required,min=3")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *ValidationError, got %T", err)
	}
	if len(ve.Fields) != 1 || ve.Fields[0].Rule != "required" {
		t.Errorf("first failing rule should be required, got %+v", ve.Fields)
	}
}

func TestNestedPaths(t *testing.T) {
	var n nested
	n.Code = "abcd"
	err := Struct(&n)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *ValidationError, got %T", err)
	}
	// both the required User struct (nil) and its Email field fail
	found := false
	for _, f := range ve.Fields {
		if f.Field == "User.Email" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing nested field path User.Email, got %+v", ve.Fields)
	}
}

func TestNilPointerStructRequired(t *testing.T) {
	type Holder struct {
		Sub *struct{ A string } `validate:"required"`
	}
	if err := Struct(&Holder{}); err == nil {
		t.Error("nil pointer to struct must be a required failure")
	}
	type OptHolder struct {
		Sub *struct{ A string } `validate:"omitempty"`
	}
	if err := Struct(&OptHolder{}); err != nil {
		t.Errorf("nil pointer with omitempty must pass: %v", err)
	}
}

func TestMalformedRulesAreErrorsNotPanics(t *testing.T) {
	if err := Field("x", "min=abc"); err == nil {
		t.Error("malformed min must error")
	}
	if err := Field("x", "bogusrule"); err == nil {
		t.Error("unknown rule must error")
	}
	if err := Field("abc", "pattern=["); err == nil {
		t.Error("invalid regexp must error")
	}
	type Bad struct {
		Age int `validate:"min=abc"`
	}
	if err := Struct(&Bad{1}); err == nil {
		t.Error("malformed rule on struct must error")
	}
}

func TestValidationErrorIs(t *testing.T) {
	err := Field("", "required")
	if !errors.Is(err, &ValidationError{}) {
		t.Error("errors.Is must match *ValidationError")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Error() must aggregate failures: %q", err)
	}
}

func TestPatternWithCommaQuantifier(t *testing.T) {
	type RangeModel struct {
		Code  string `validate:"pattern=^[0-9]{1,3}$"`
		Alpha string `validate:"required,pattern='^[a-z]{2,4}$'"`
		Colon string `validate:"pattern:^[A-Z]{2,3}$"`
	}

	valid := RangeModel{Code: "123", Alpha: "abcd", Colon: "ABC"}
	if err := Struct(&valid); err != nil {
		t.Fatalf("valid RangeModel failed: %v", err)
	}

	invalid := RangeModel{Code: "1234", Alpha: "a", Colon: "A"}
	err := Struct(&invalid)
	if err == nil {
		t.Fatal("invalid RangeModel expected errors")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Fields) != 3 {
		t.Errorf("expected 3 field errors, got %d: %+v", len(ve.Fields), ve.Fields)
	}
}

func TestPatternCacheConcurrent(t *testing.T) {
	const goroutines = 20
	const iterations = 50
	errCh := make(chan error, goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < iterations; i++ {
				if err := Field("test1234", "pattern=^[a-z]+[0-9]+$"); err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}()
	}

	for g := 0; g < goroutines; g++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent pattern check failed: %v", err)
		}
	}
}

type OrderItem struct {
	Name     string `validate:"required"`
	Quantity int    `validate:"min=1"`
}

type Order struct {
	ID       string               `validate:"required"`
	Items    []OrderItem          `validate:"min=1"`
	ExtraMap map[string]OrderItem `validate:"omitempty"`
	ItemPtrs []*OrderItem         `validate:"omitempty"`
}

func TestNestedSlicesAndMaps(t *testing.T) {
	valid := Order{
		ID: "ORD-001",
		Items: []OrderItem{
			{Name: "Book", Quantity: 2},
			{Name: "Pen", Quantity: 5},
		},
		ExtraMap: map[string]OrderItem{
			"gift": {Name: "Bookmark", Quantity: 1},
		},
		ItemPtrs: []*OrderItem{
			{Name: "Notebook", Quantity: 1},
		},
	}
	if err := Struct(&valid); err != nil {
		t.Fatalf("valid order failed: %v", err)
	}

	invalid := Order{
		ID: "ORD-002",
		Items: []OrderItem{
			{Name: "Book", Quantity: 1},
			{Name: "", Quantity: 0}, // fails Items[1].Name and Items[1].Quantity
		},
		ExtraMap: map[string]OrderItem{
			"bad": {Name: "", Quantity: 3}, // fails ExtraMap[bad].Name
		},
		ItemPtrs: []*OrderItem{
			nil,                             // skipped without panic
			{Name: "Sticker", Quantity: -1}, // fails ItemPtrs[1].Quantity
		},
	}
	err := Struct(&invalid)
	if err == nil {
		t.Fatal("invalid order expected errors")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	fieldMap := make(map[string]string)
	for _, f := range ve.Fields {
		fieldMap[f.Field] = f.Rule
	}

	expectedFields := []string{
		"Items[1].Name",
		"Items[1].Quantity",
		"ExtraMap[bad].Name",
		"ItemPtrs[1].Quantity",
	}

	for _, expected := range expectedFields {
		if _, exists := fieldMap[expected]; !exists {
			t.Errorf("expected error on field %q, but got fields: %+v", expected, fieldMap)
		}
	}
}
