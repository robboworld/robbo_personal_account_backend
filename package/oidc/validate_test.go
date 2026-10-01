package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type idpFixture struct {
	key *rsa.PrivateKey
	cfg *Config
}

func newIdPFixture(t *testing.T, issuer string) *idpFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	return &idpFixture{key: key, cfg: &Config{Issuer: issuer, ClientID: "lk-web", Audience: "openedx", jwks: newJWKSCache(srv.URL)}}
}

func (f *idpFixture) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "k1"
	s, err := tok.SignedString(f.key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestValidateIDTokenAcceptsValidToken(t *testing.T) {
	f := newIdPFixture(t, "https://lms.example.com/oauth2")
	tok := f.sign(t, jwt.MapClaims{"iss": f.cfg.Issuer, "sub": "alice", "aud": "openedx", "exp": time.Now().Add(time.Hour).Unix()})
	if _, err := f.cfg.ValidateIDToken(tok, ""); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
}

func TestValidateIDTokenRequiresExp(t *testing.T) {
	f := newIdPFixture(t, "https://lms.example.com/oauth2")
	tok := f.sign(t, jwt.MapClaims{"iss": f.cfg.Issuer, "sub": "alice", "aud": "openedx"})
	if _, err := f.cfg.ValidateIDToken(tok, ""); err == nil {
		t.Fatal("token without exp accepted")
	}
}

func TestValidateIDTokenPrivateIssuerNeedsAudUnlessMockAllowed(t *testing.T) {
	f := newIdPFixture(t, "http://10.0.0.5/oauth2")
	tok := f.sign(t, jwt.MapClaims{"iss": f.cfg.Issuer, "sub": "alice", "exp": time.Now().Add(time.Hour).Unix()})
	if _, err := f.cfg.ValidateIDToken(tok, ""); err == nil {
		t.Fatal("token without aud from private-IP issuer accepted")
	}
	viper.Set("oidc.allowMockAudience", true)
	t.Cleanup(func() { viper.Set("oidc.allowMockAudience", false) })
	if _, err := f.cfg.ValidateIDToken(tok, ""); err != nil {
		t.Fatalf("mock audience opt-in rejected token: %v", err)
	}
}
