package oidc

import (
	"errors"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
)

const SessionCookieName = "lk_bff_session"

const defaultSessionTTLSeconds = 28800

func SessionTTLSeconds() int {
	ttl := viper.GetInt("auth.bff_session_ttl")
	if ttl <= 0 {
		return defaultSessionTTLSeconds
	}
	return ttl
}

func IssueSessionToken(sub, edxUserID, email string, role uint, sid string) (string, error) {
	if strings.TrimSpace(sid) == "" {
		return "", errors.New("session sid is required")
	}
	claims := models.OidcSessionClaims{
		Sub:       sub,
		EdxUserID: edxUserID,
		Email:     email,
		Role:      role,
		Sid:       sid,
	}
	ttl := SessionTTLSeconds()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         claims.Sub,
		"edx_user_id": claims.EdxUserID,
		"email":       claims.Email,
		"role":        claims.Role,
		"sid":         claims.Sid,
		"exp":         time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
		"typ":         sessionTokenType,
	})
	return token.SignedString([]byte(viper.GetString("auth.access_signing_key")))
}

// sessionTokenType marks BFF session JWTs; play tokens and legacy JWTs share the signing key.
const sessionTokenType = "lk_bff"

func ParseSessionToken(token string) (*models.OidcSessionClaims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected session token alg")
		}
		return []byte(viper.GetString("auth.access_signing_key")), nil
	})
	if err != nil {
		return nil, err
	}
	raw, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid session claims")
	}
	if asString(raw["typ"]) != sessionTokenType {
		return nil, errors.New("not a session token")
	}
	return &models.OidcSessionClaims{
		Sub:       asString(raw["sub"]),
		EdxUserID: asString(raw["edx_user_id"]),
		Email:     asString(raw["email"]),
		Role:      uint(asInt64(raw["role"])),
		Sid:       asString(raw["sid"]),
	}, nil
}
