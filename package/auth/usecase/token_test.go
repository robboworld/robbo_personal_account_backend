package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

var testKey = []byte("0123456789abcdef0123456789abcdef")

// hs256 signs a raw JSON payload, i.e. a token as dgrijalva/jwt-go v4 used to issue it.
func hs256(payload string) string {
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, testKey)
	mac.Write([]byte(unsigned))
	return unsigned + "." + enc.EncodeToString(mac.Sum(nil))
}

func TestParseTokenAcceptsPreV5Payload(t *testing.T) {
	exp := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	tok := hs256(`{"exp":` + exp + `,"Id":"7","Role":1,"sid":"s-1"}`)
	claims, err := (&AuthUseCaseImpl{}).ParseToken(tok, testKey)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Id != "7" || claims.Role != models.Role(1) || claims.Sid != "s-1" {
		t.Fatalf("unexpected claims %+v", claims)
	}
}

func TestGenerateTokenRoundTrip(t *testing.T) {
	uc := &AuthUseCaseImpl{}
	tok, err := uc.GenerateToken(&models.UserCore{Id: "7", Role: models.Role(1)}, "s-1", 60, testKey)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := uc.ParseToken(tok, testKey)
	if err != nil || claims.Id != "7" || claims.Sid != "s-1" {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
}

func TestParseTokenRejectsExpiredAndForeignAlg(t *testing.T) {
	uc := &AuthUseCaseImpl{}
	expired := hs256(`{"exp":` + strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10) + `,"Id":"7"}`)
	if _, err := uc.ParseToken(expired, testKey); err == nil {
		t.Fatal("expired token accepted")
	}
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, models.UserClaims{Id: "7"}).SignedString(testKey)
	if _, err := uc.ParseToken(hs512, testKey); err == nil {
		t.Fatal("HS512 token accepted")
	}
}
