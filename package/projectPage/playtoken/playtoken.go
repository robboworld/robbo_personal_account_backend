package playtoken

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/spf13/viper"
)

const scopePlay = "scratch.play"

// Claims for short-lived project play/download URLs.
type Claims struct {
	jwt.StandardClaims
	Scope         string `json:"scope"`
	ProjectPageID string `json:"ppid"`
	ProjectID     string `json:"pid"`
}

var (
	ErrInvalidToken = errors.New("invalid play token")
	ErrWrongScope   = errors.New("invalid play token scope")
)

func signingKey() []byte {
	return []byte(viper.GetString("auth.access_signing_key"))
}

func ttl() time.Duration {
	d := viper.GetDuration("projectPage.play_token_ttl")
	if d <= 0 {
		return 15 * time.Minute
	}
	return d
}

// Issue creates a signed token for play/json URLs.
func Issue(projectPageID, projectID string) (token string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(ttl())
	claims := Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(expiresAt),
			IssuedAt:  jwt.At(time.Now()),
		},
		Scope:         scopePlay,
		ProjectPageID: projectPageID,
		ProjectID:     projectID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString(signingKey())
	return
}

// Parse validates token and returns claims.
func Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	})
	if err != nil || !t.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Scope != scopePlay {
		return nil, ErrWrongScope
	}
	return claims, nil
}
