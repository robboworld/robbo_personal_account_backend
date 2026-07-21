package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := "./keys/licensing"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(filepath.Join(outDir, "addon"), 0o755); err != nil {
		panic(err)
	}
	privPath := filepath.Join(outDir, "dev-private.pem")
	pubPath := filepath.Join(outDir, "dev-public.pem")
	addonPath := filepath.Join(outDir, "addon", "paid-addon.js")

	if _, err := os.Stat(privPath); err == nil {
		fmt.Println("private key already exists:", privPath)
	} else {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		privBytes, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			panic(err)
		}
		privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
		if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
			panic(err)
		}
		pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			panic(err)
		}
		pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
		if err := os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", privPath, "and", pubPath)
	}

	if _, err := os.Stat(addonPath); err != nil {
		placeholder := "// rs3 paid-addon placeholder for local licensing tests\n" +
			"globalThis.__RS3_PAID_ADDON_FACTORY__ = function () {\n" +
			"  return { id: 'rs3-paid-addon', capabilities: ['premium.auto_update'] };\n" +
			"};\n"
		if err := os.WriteFile(addonPath, []byte(placeholder), 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", addonPath)
	} else {
		fmt.Println("addon already exists:", addonPath)
	}
}
