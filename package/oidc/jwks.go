package oidc

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
	jwksURI   string
}

func newJWKSCache(jwksURI string) *jwksCache {
	return &jwksCache{
		keys:    map[string]*rsa.PublicKey{},
		ttl:     15 * time.Minute,
		jwksURI: jwksURI,
	}
}

func (c *jwksCache) getKey(kid string) (*rsa.PublicKey, error) {
	if err := c.refresh(false); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok := c.keys[kid]
	if !ok && len(c.keys) == 1 {
		for _, k := range c.keys {
			return k, nil
		}
	}
	if !ok {
		return nil, fmt.Errorf("jwks: key %q not found", kid)
	}
	return key, nil
}

func (c *jwksCache) refresh(force bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && time.Since(c.fetchedAt) < c.ttl && len(c.keys) > 0 {
		return nil
	}
	resp, err := http.Get(c.jwksURI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks http %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicKeyFromModExp(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks: no RSA keys")
	}
	c.keys = keys
	c.fetchedAt = time.Now()
	return nil
}

func rsaPublicKeyFromModExp(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())
	return &rsa.PublicKey{N: n, E: e}, nil
}
