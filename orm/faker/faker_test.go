package faker

import (
	"strings"
	"testing"
	"time"
)

func TestFakerDeterminism(t *testing.T) {
	fk1 := New(42)
	fk2 := New(42)

	for i := 0; i < 50; i++ {
		name1 := fk1.Name()
		name2 := fk2.Name()
		if name1 != name2 {
			t.Fatalf("deterministic faker produced different names: %q vs %q", name1, name2)
		}

		email1 := fk1.Email()
		email2 := fk2.Email()
		if email1 != email2 {
			t.Fatalf("deterministic faker produced different emails: %q vs %q", email1, email2)
		}

		num1 := fk1.IntBetween(10, 1000)
		num2 := fk2.IntBetween(10, 1000)
		if num1 != num2 {
			t.Fatalf("deterministic faker produced different numbers: %d vs %d", num1, num2)
		}
	}
}

func TestFakerOutputSanity(t *testing.T) {
	fk := New()

	if name := fk.Name(); !strings.Contains(name, " ") {
		t.Errorf("expected full name with space, got %q", name)
	}

	if email := fk.Email(); !strings.Contains(email, "@") {
		t.Errorf("expected valid email with @, got %q", email)
	}

	if phone := fk.Phone(); len(phone) < 8 {
		t.Errorf("expected valid phone string, got %q", phone)
	}

	if ipv4 := fk.IPv4(); strings.Count(ipv4, ".") != 3 {
		t.Errorf("expected IPv4 address, got %q", ipv4)
	}

	if uuid := fk.UUID(); len(uuid) != 36 || strings.Count(uuid, "-") != 4 {
		t.Errorf("expected valid UUID v4, got %q", uuid)
	}

	if addr := fk.Address(); len(addr) < 10 {
		t.Errorf("expected valid address, got %q", addr)
	}

	if company := fk.Company(); len(company) < 3 {
		t.Errorf("expected company name, got %q", company)
	}

	if price := fk.Price(10.0, 50.0); price < 10.0 || price > 50.0 {
		t.Errorf("expected price between 10 and 50, got %f", price)
	}

	sentence := fk.Sentence()
	if !strings.HasSuffix(sentence, ".") {
		t.Errorf("expected sentence ending with '.', got %q", sentence)
	}

	paragraph := fk.Paragraph()
	if len(paragraph) < 20 {
		t.Errorf("expected paragraph of reasonable length, got %q", paragraph)
	}

	paragraphs := fk.Paragraphs(3)
	if strings.Count(paragraphs, "\n\n") != 2 {
		t.Errorf("expected 3 paragraphs joined by double newline, got %d occurrences", strings.Count(paragraphs, "\n\n"))
	}

	num := fk.IntBetween(5, 10)
	if num < 5 || num > 10 {
		t.Errorf("expected int between 5 and 10, got %d", num)
	}

	lat := fk.Latitude()
	if lat < -90.0 || lat > 90.0 {
		t.Errorf("expected latitude between -90 and 90, got %f", lat)
	}

	lng := fk.Longitude()
	if lng < -180.0 || lng > 180.0 {
		t.Errorf("expected longitude between -180 and 180, got %f", lng)
	}

	past := fk.PastDate()
	if past.After(time.Now()) {
		t.Errorf("expected past date, got %v", past)
	}

	future := fk.FutureDate()
	if future.Before(time.Now()) {
		t.Errorf("expected future date, got %v", future)
	}

	shuffled := Shuffle(fk, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if len(shuffled) != 10 {
		t.Errorf("expected slice length preserved after shuffle, got %d", len(shuffled))
	}
}

func TestFakerGlobalHelpers(t *testing.T) {
	if Name() == "" {
		t.Error("expected global Name() to return value")
	}
	if Email() == "" {
		t.Error("expected global Email() to return value")
	}
	if IntBetween(1, 5) < 1 {
		t.Error("expected global IntBetween() to work")
	}
}
