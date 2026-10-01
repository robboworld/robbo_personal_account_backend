package oidc

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/spf13/viper"
)

const testSigningKey = "0123456789abcdef0123456789abcdef"

func withSigningKey(t *testing.T) {
	t.Helper()
	viper.Set("auth.access_signing_key", testSigningKey)
	t.Cleanup(func() { viper.Set("auth.access_signing_key", "") })
}

func signHS(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString([]byte(testSigningKey))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSessionTokenRoundTrip(t *testing.T) {
	withSigningKey(t)
	tok, err := IssueSessionToken("alice", "42", "a@example.com", 1, "sid-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseSessionToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.EdxUserID != "42" || claims.Sid != "sid-1" || claims.Role != 1 {
		t.Fatalf("unexpected claims %+v", claims)
	}
}

func TestParseSessionTokenRejectsOtherTokenTypes(t *testing.T) {
	withSigningKey(t)
	exp := time.Now().Add(time.Hour).Unix()
	cases := map[string]string{
		// Play tokens / legacy JWTs share the key but carry no typ=lk_bff.
		"no typ":    signHS(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "alice", "sid": "s", "exp": exp, "role": 5}),
		"wrong typ": signHS(t, jwt.SigningMethodHS256, jwt.MapClaims{"sub": "alice", "sid": "s", "exp": exp, "typ": "play"}),
		"HS512":     signHS(t, jwt.SigningMethodHS512, jwt.MapClaims{"sub": "alice", "sid": "s", "exp": exp, "typ": "lk_bff"}),
	}
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "alice", "sid": "s", "exp": exp, "typ": "lk_bff"}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	cases["alg none"] = none
	for name, tok := range cases {
		if _, err := ParseSessionToken(tok); err == nil {
			t.Errorf("%s: token accepted, want error", name)
		}
	}
}
