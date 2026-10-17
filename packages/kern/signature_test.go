package kern

import (
	"testing"
)

func TestBuildSignature(t *testing.T) {
	if got := BuildSignature(); got != "build with krewire" {
		t.Errorf("BuildSignature() = %q, want %q", got, "build with krewire")
	}

	old := Signature
	defer func() { Signature = old }()

	Signature = "custom-signature"
	if got := BuildSignature(); got != "custom-signature" {
		t.Errorf("BuildSignature() after override = %q, want %q", got, "custom-signature")
	}
}
