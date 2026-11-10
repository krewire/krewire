package sec_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/krewire/krewire/pkg/sec"
)

// RFC 6070 PBKDF2 HMAC-SHA256 test vectors:
// Input: password = "password", salt = "salt", iterations = 1, dkLen = 32
// Expected: 120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b
// Input: password = "password", salt = "salt", iterations = 2, dkLen = 32
// Expected: ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c261ed46e847b38c456609
func TestDeriveKeyPBKDF2_RFC6070Vectors(t *testing.T) {
	tests := []struct {
		password string
		salt     string
		iter     int
		keyLen   int
		expected string
	}{
		{
			password: "password",
			salt:     "salt",
			iter:     1,
			keyLen:   32,
			expected: "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b",
		},
		{
			password: "password",
			salt:     "salt",
			iter:     2,
			keyLen:   32,
			expected: "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43",
		},
		{
			password: "password",
			salt:     "salt",
			iter:     4096,
			keyLen:   32,
			expected: "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a",
		},
	}

	for _, tc := range tests {
		dk := sec.DeriveKeyPBKDF2([]byte(tc.password), []byte(tc.salt), tc.iter, tc.keyLen)
		got := hex.EncodeToString(dk)
		if got != tc.expected {
			t.Errorf("PBKDF2(iter=%d) = %s, want %s", tc.iter, got, tc.expected)
		}
	}
}

func TestHashAndVerifyPasswordPBKDF2(t *testing.T) {
	password := "SecretKrewireP@ssw0rd!"
	encoded, err := sec.HashPasswordPBKDF2(password, 2048) // lower iter for fast unit test
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !strings.HasPrefix(encoded, "pbkdf2:sha256:2048:") {
		t.Errorf("unexpected hash prefix: %s", encoded)
	}

	// Verify valid password
	if !sec.VerifyPasswordPBKDF2(password, encoded) {
		t.Errorf("verification failed for correct password")
	}

	// Verify invalid password
	if sec.VerifyPasswordPBKDF2("wrongPassword", encoded) {
		t.Errorf("verification succeeded for incorrect password")
	}

	// Verify tampering with hash
	if sec.VerifyPasswordPBKDF2(password, encoded+"corrupted") {
		t.Errorf("verification succeeded for corrupted hash")
	}

	// Verify invalid format strings
	if sec.VerifyPasswordPBKDF2(password, "invalid:format") {
		t.Errorf("verification succeeded for invalid format")
	}
	if sec.VerifyPasswordPBKDF2(password, "") {
		t.Errorf("verification succeeded for empty hash")
	}
}
