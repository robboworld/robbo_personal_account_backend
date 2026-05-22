package lmsdb

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const djangoPBKDF2Iterations = 600000

// VerifyDjangoPassword checks a plain password against an Open edX / Django auth_user hash
// (e.g. pbkdf2_sha256$600000$salt$base64hash).
func VerifyDjangoPassword(plainPassword, encoded string) bool {
	if plainPassword == "" || encoded == "" || strings.HasPrefix(encoded, "!") {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 {
		return false
	}
	if parts[0] != "pbkdf2_sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt := parts[2]
	expected, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		expected, err = base64.RawStdEncoding.DecodeString(parts[3])
		if err != nil {
			return false
		}
	}
	derived := pbkdf2.Key([]byte(plainPassword), []byte(salt), iterations, len(expected), sha256.New)
	return subtle.ConstantTimeCompare(derived, expected) == 1
}

// EncodeDjangoPassword hashes a password like Django/Open edX auth_user.password.
func EncodeDjangoPassword(plainPassword string) (string, error) {
	if plainPassword == "" {
		return "", fmt.Errorf("empty password")
	}
	salt := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	saltStr := base64.RawURLEncoding.EncodeToString(salt)[:16]
	derived := pbkdf2.Key([]byte(plainPassword), []byte(saltStr), djangoPBKDF2Iterations, 32, sha256.New)
	hash := base64.StdEncoding.EncodeToString(derived)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", djangoPBKDF2Iterations, saltStr, hash), nil
}
