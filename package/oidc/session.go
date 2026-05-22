package oidc

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
)

const SessionCookieName = "lk_bff_session"

func IssueSessionToken(sub, edxUserID, email string, role uint) (string, error) {
	claims := models.OidcSessionClaims{
		Sub:       sub,
		EdxUserID: edxUserID,
		Email:     email,
		Role:      role,
	}
	ttl := viper.GetInt("auth.access_token_ttl")
	if ttl <= 0 {
		ttl = 300
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         claims.Sub,
		"edx_user_id": claims.EdxUserID,
		"email":       claims.Email,
		"role":        claims.Role,
		"exp":         time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
		"typ":         "lk_bff",
	})
	return token.SignedString([]byte(viper.GetString("auth.access_signing_key")))
}

func ParseSessionToken(token string) (*models.OidcSessionClaims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("auth.access_signing_key")), nil
	})
	if err != nil {
		return nil, err
	}
	raw, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid session claims")
	}
	return &models.OidcSessionClaims{
		Sub:       asString(raw["sub"]),
		EdxUserID: asString(raw["edx_user_id"]),
		Email:     asString(raw["email"]),
		Role:      uint(asInt64(raw["role"])),
	}, nil
}
