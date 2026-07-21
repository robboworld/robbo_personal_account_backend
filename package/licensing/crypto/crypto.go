package crypto

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

type Signer struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Kid        string
	Issuer     string
}

func LoadSigner(privateKeyPath, kid, issuer string) (*Signer, error) {
	pemBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM private key")
	}
	var priv *rsa.PrivateKey
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		var ok bool
		priv, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not RSA private key")
		}
	} else if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		priv = key
	} else {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	if issuer == "" {
		issuer = "rs3-licensing"
	}
	if kid == "" {
		kid = "rs3-prod-1"
	}
	return &Signer{
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		Kid:        kid,
		Issuer:     issuer,
	}, nil
}

func b64url(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func b64urlDecode(s string) ([]byte, error) {
	padded := s
	switch len(s) % 4 {
	case 2:
		padded += "=="
	case 3:
		padded += "="
	}
	return base64.URLEncoding.DecodeString(padded)
}

type TokenClaims struct {
	Iss              string   `json:"iss"`
	Sub              string   `json:"sub"`
	LicenseID        string   `json:"licenseId"`
	EntitlementID    string   `json:"entitlementId"`
	SeatID           string   `json:"seatId"`
	Iat              int64    `json:"iat"`
	Nbf              int64    `json:"nbf"`
	Exp              int64    `json:"exp"`
	Capabilities     []string `json:"capabilities"`
	DeviceBinding    string   `json:"deviceBinding"`
	AddonManifestUrl string   `json:"addonManifestUrl"`
}

func (s *Signer) SignJWT(claims TokenClaims) (string, error) {
	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": s.Kid}
	hJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	data := b64url(hJSON) + "." + b64url(pJSON)
	hash := sha256.Sum256([]byte(data))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return data + "." + b64url(sig), nil
}

func (s *Signer) VerifyJWT(token string) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid_token_shape")
	}
	hJSON, err := b64urlDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid_token_shape")
	}
	var header map[string]string
	if err := json.Unmarshal(hJSON, &header); err != nil {
		return nil, fmt.Errorf("invalid_token_shape")
	}
	if header["alg"] != "RS256" {
		return nil, fmt.Errorf("unsupported_alg")
	}
	sig, err := b64urlDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid_signature")
	}
	data := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(data))
	if err := rsa.VerifyPKCS1v15(s.PublicKey, crypto.SHA256, hash[:], sig); err != nil {
		return nil, fmt.Errorf("invalid_signature")
	}
	pJSON, err := b64urlDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid_token_shape")
	}
	var claims TokenClaims
	if err := json.Unmarshal(pJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid_token_shape")
	}
	now := time.Now().Unix()
	if claims.Exp > 0 && now >= claims.Exp {
		return nil, fmt.Errorf("license_expired")
	}
	return &claims, nil
}

func (s *Signer) SignManifest(manifest map[string]interface{}) (string, error) {
	ordered := make(map[string]interface{})
	keys := make([]string, 0, len(manifest))
	for k := range manifest {
		if k == "manifestSignature" || k == "signatureKid" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ordered[k] = manifest[k]
	}
	canonical, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(canonical)
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return b64url(sig), nil
}

func DeriveAddonKey(fingerprint, licenseID string) []byte {
	sum := sha256.Sum256([]byte(fingerprint + ":" + licenseID))
	return sum[:]
}

func EncryptAddonAESGCM(plaintext []byte, fingerprint, licenseID string) (string, error) {
	key := DeriveAddonKey(fingerprint, licenseID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, iv, plaintext, nil)
	// Seal appends tag at end: ciphertext || tag
	tagSize := gcm.Overhead()
	if len(ciphertext) < tagSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	ct, tag := ciphertext[:len(ciphertext)-tagSize], ciphertext[len(ciphertext)-tagSize:]
	// Wire format matching mock: iv || tag || ciphertext
	out := append(append(iv, tag...), ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func Sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
