package playtoken

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/spf13/viper"
)

func TestIssueAndParse_playScope(t *testing.T) {
	viper.Set("auth.access_signing_key", "test-signing-key-for-play-token")
	t.Cleanup(func() {
		viper.Set("auth.access_signing_key", "")
	})

	const pageID = "42"
	const projectID = "550e8400-e29b-41d4-a716-446655440000"

	token, expiresAt, err := Issue(pageID, projectID)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expiresAt should be in the future")
	}

	claims, err := Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Scope != scopePlay {
		t.Fatalf("scope: got %q want %q", claims.Scope, scopePlay)
	}
	if claims.ProjectPageID != pageID {
		t.Fatalf("ppid: got %q want %q", claims.ProjectPageID, pageID)
	}
	if claims.ProjectID != projectID {
		t.Fatalf("pid: got %q want %q", claims.ProjectID, projectID)
	}
}

func TestParse_rejectsWrongScope(t *testing.T) {
	viper.Set("auth.access_signing_key", "test-signing-key-for-play-token")
	t.Cleanup(func() {
		viper.Set("auth.access_signing_key", "")
	})

	expiresAt := time.Now().Add(5 * time.Minute)
	claims := Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(expiresAt),
			IssuedAt:  jwt.At(time.Now()),
		},
		Scope:         "other.scope",
		ProjectPageID: "1",
		ProjectID:     "2",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tok.SignedString(signingKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(token); err != ErrWrongScope {
		t.Fatalf("expected ErrWrongScope, got %v", err)
	}
}

func TestParse_rejectsInvalidToken(t *testing.T) {
	viper.Set("auth.access_signing_key", "test-signing-key-for-play-token")
	t.Cleanup(func() {
		viper.Set("auth.access_signing_key", "")
	})
	if _, err := Parse("not-a-jwt"); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
