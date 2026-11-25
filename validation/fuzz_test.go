package validation_test

import (
	"testing"

	"github.com/krewire/krewire/validation"
)

func FuzzValidation(f *testing.F) {
	seeds := []string{
		"user@example.com",
		"not-an-email",
		"https://krewire.com/docs",
		"javascript:alert(1)",
		"12345",
		"-999",
		"abcdef",
		"01234567-89ab-cdef-0123-456789abcdef",
		"",
		"   ",
		"null",
		"DROP TABLE users;--",
		"admin' OR '1'='1",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	type FuzzTarget struct {
		Email        string `validate:"email"`
		URL          string `validate:"url"`
		Numeric      string `validate:"numeric"`
		Alpha        string `validate:"alpha"`
		Alphanumeric string `validate:"alphanumeric"`
		UUID         string `validate:"uuid"`
	}

	f.Fuzz(func(t *testing.T, val string) {
		// Validating arbitrary strings across struct rules must never panic
		target := FuzzTarget{
			Email:        val,
			URL:          val,
			Numeric:      val,
			Alpha:        val,
			Alphanumeric: val,
			UUID:         val,
		}
		_ = validation.Struct(target)
	})
}
