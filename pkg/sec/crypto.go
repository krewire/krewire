package sec

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	// DefaultPBKDF2Iterations is the recommended iteration count for PBKDF2-HMAC-SHA256 (OWASP minimum).
	DefaultPBKDF2Iterations = 100_000

	// DefaultSaltBytes is the byte length for generated cryptographic salts.
	DefaultSaltBytes = 16

	// DefaultKeyBytes is the byte length for derived cryptographic keys.
	DefaultKeyBytes = 32
)

var (
	// ErrInvalidHashFormat is returned when an encoded hash does not match the expected PBKDF2 scheme.
	ErrInvalidHashFormat = errors.New("sec: invalid encoded hash format")

	// ErrShortSalt is returned when a provided salt length is less than 8 bytes.
	ErrShortSalt = errors.New("sec: salt length must be at least 8 bytes")
)

// DeriveKeyPBKDF2 derives a cryptographic key from password and salt using
// PBKDF2 with HMAC-SHA256 per RFC 2898 / RFC 8018.
// Built entirely on Go standard library crypto primitives (stdlib-first).
func DeriveKeyPBKDF2(password, salt []byte, iter, keyLen int) []byte {
	if iter <= 0 || keyLen <= 0 {
		return nil
	}
	h := hmac.New(sha256.New, password)
	hashLen := h.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var (
		buf   [4]byte
		dk    = make([]byte, 0, numBlocks*hashLen)
		u     = make([]byte, hashLen)
		block = make([]byte, hashLen)
	)

	for i := 1; i <= numBlocks; i++ {
		// U_1 = PRF(P, S || INT_32_BE(i))
		h.Reset()
		h.Write(salt)
		binary.BigEndian.PutUint32(buf[:], uint32(i))
		h.Write(buf[:])
		u = h.Sum(u[:0])
		copy(block, u)

		// U_2 ... U_c
		for j := 2; j <= iter; j++ {
			h.Reset()
			h.Write(u)
			u = h.Sum(u[:0])
			for k := 0; k < hashLen; k++ {
				block[k] ^= u[k]
			}
		}

		dk = append(dk, block...)
	}

	return dk[:keyLen]
}

// HashPasswordPBKDF2 hashes password using PBKDF2-HMAC-SHA256 with a cryptographically secure random salt.
// The returned string format is "pbkdf2:sha256:<iterations>:<hex-salt>:<hex-derived-key>".
func HashPasswordPBKDF2(password string, iter int) (string, error) {
	if iter <= 0 {
		iter = DefaultPBKDF2Iterations
	}
	salt := make([]byte, DefaultSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("sec: generate salt: %w", err)
	}

	dk := DeriveKeyPBKDF2([]byte(password), salt, iter, DefaultKeyBytes)
	return fmt.Sprintf("pbkdf2:sha256:%d:%s:%s", iter, hex.EncodeToString(salt), hex.EncodeToString(dk)), nil
}

// VerifyPasswordPBKDF2 verifies plain password against an encoded hash produced by HashPasswordPBKDF2.
// It uses constant-time comparison (crypto/subtle.ConstantTimeCompare) to prevent timing side-channel attacks.
func VerifyPasswordPBKDF2(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, ":")
	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
		return false
	}

	iter, err := strconv.Atoi(parts[2])
	if err != nil || iter <= 0 {
		return false
	}

	salt, err := hex.DecodeString(parts[3])
	if err != nil || len(salt) == 0 {
		return false
	}

	expectedKey, err := hex.DecodeString(parts[4])
	if err != nil || len(expectedKey) == 0 {
		return false
	}

	derived := DeriveKeyPBKDF2([]byte(password), salt, iter, len(expectedKey))
	return subtle.ConstantTimeCompare(derived, expectedKey) == 1
}
