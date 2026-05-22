package lmsdb

import "testing"

func TestVerifyDjangoPassword_openedxSample(t *testing.T) {
	// From dump: user lmslms / lms@lms.ru — password unknown; structure must parse.
	hash := "pbkdf2_sha256$600000$vTi6c8WAL15Q07eivhaZmS$cMUALu7oCqy574X/gMXBF+uO3LFaBsKgUGIgB3lKYCI="
	if VerifyDjangoPassword("", hash) {
		t.Fatal("empty password must not match")
	}
	if VerifyDjangoPassword("wrong", hash) {
		t.Fatal("wrong password should not match known hash")
	}
}

func TestVerifyDjangoPassword_rejectsUnusable(t *testing.T) {
	if VerifyDjangoPassword("x", "!tthENKcwdyhBbJiTZoNwbqDmDdvXTwbqeX1INIX8") {
		t.Fatal("unusable password must not verify")
	}
}

func TestEncodeDjangoPassword_roundtrip(t *testing.T) {
	encoded, err := EncodeDjangoPassword("123")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyDjangoPassword("123", encoded) {
		t.Fatal("encoded password must verify")
	}
	if VerifyDjangoPassword("124", encoded) {
		t.Fatal("wrong password must not verify")
	}
}
