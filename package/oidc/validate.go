package oidc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/spf13/viper"
)

type IDTokenClaims struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Aud   interface{} `json:"aud"`
	Exp   int64  `json:"exp"`
	Nonce string `json:"nonce"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (c *Config) ValidateIDToken(idToken, expectedNonce string) (*IDTokenClaims, error) {
	if c.jwks == nil {
		return nil, errors.New("oidc: jwks not configured")
	}
	parser := &jwt.Parser{}
	unverified, _, err := parser.ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	kid := ""
	if h, ok := unverified.Header["kid"].(string); ok {
		kid = h
	}
	key, err := c.jwks.getKey(kid)
	if err != nil {
		return nil, err
	}
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected alg %s", token.Method.Alg())
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	raw, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("oidc: invalid claims")
	}
	claims := &IDTokenClaims{
		Iss: asString(raw["iss"]),
		Sub: asString(raw["sub"]),
		Aud: raw["aud"],
		Exp: asInt64(raw["exp"]),
		Nonce: asString(raw["nonce"]),
		Email: asString(raw["email"]),
		Name: asString(raw["name"]),
	}
	if claims.Iss != c.Issuer {
		return nil, errors.New("oidc: invalid_issuer")
	}
	if !audienceMatches(claims.Aud, c.ClientID) {
		return nil, errors.New("oidc: invalid_audience")
	}
	if claims.Exp > 0 && time.Now().Unix() >= claims.Exp {
		return nil, errors.New("oidc: token_expired")
	}
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return nil, errors.New("oidc: invalid_nonce")
	}
	if claims.Sub == "" {
		return nil, errors.New("oidc: missing_sub")
	}
	return claims, nil
}

func audienceMatches(aud interface{}, clientID string) bool {
	switch v := aud.(type) {
	case string:
		return v == clientID
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == clientID {
				return true
			}
		}
	}
	return false
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	}
	return 0
}

type Config struct {
	Issuer                string
	AuthorizationEndpoint string
	TokenEndpoint         string
	JWKSURI               string
	ClientID              string
	RedirectURI           string
	Scopes                string
	jwks                  *jwksCache
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Issuer:                viper.GetString("oidc.issuer"),
		AuthorizationEndpoint: viper.GetString("oidc.authorizationEndpoint"),
		TokenEndpoint:         viper.GetString("oidc.tokenEndpoint"),
		JWKSURI:               viper.GetString("oidc.jwksUri"),
		ClientID:              viper.GetString("oidc.clientId"),
		RedirectURI:           viper.GetString("oidc.redirectUri"),
		Scopes:                viper.GetString("oidc.scopes"),
	}
	if cfg.Scopes == "" {
		cfg.Scopes = "openid profile email"
	}
	if cfg.Issuer == "" || cfg.AuthorizationEndpoint == "" || cfg.TokenEndpoint == "" ||
		cfg.JWKSURI == "" || cfg.ClientID == "" || cfg.RedirectURI == "" {
		return nil, errors.New("oidc: incomplete configuration")
	}
	cfg.jwks = newJWKSCache(cfg.JWKSURI)
	return cfg, nil
}

func (c *Config) IsEnabled() bool {
	return viper.GetBool("oidc.enabled") || strings.EqualFold(viper.GetString("auth.mode"), "oidc_bff")
}
