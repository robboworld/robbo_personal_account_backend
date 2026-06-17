package server

import (
	"testing"
)

func TestCorsOriginAllowed_defaults(t *testing.T) {
	t.Setenv("CORS_ALLOW_PRIVATE_NETWORK", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	if !corsOriginAllowed("http://localhost:3030") {
		t.Fatal("expected localhost:3030 to be allowed")
	}
	if corsOriginAllowed("http://192.168.1.10:3030") {
		t.Fatal("private IP should be blocked without CORS_ALLOW_PRIVATE_NETWORK")
	}
}

func TestCorsOriginAllowed_privateNetwork(t *testing.T) {
	t.Setenv("CORS_ALLOW_PRIVATE_NETWORK", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	if !corsOriginAllowed("http://192.168.88.67:3030") {
		t.Fatal("expected LAN frontend origin to be allowed")
	}
	if corsOriginAllowed("http://192.168.88.67:8080") {
		t.Fatal("backend port should not be allowed as frontend origin")
	}
	if corsOriginAllowed("http://evil.example.com:3030") {
		t.Fatal("public host should not be allowed")
	}
}

func TestCorsOriginAllowed_extraOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOW_PRIVATE_NETWORK", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://lk.example.com, https://staging.example.com")

	if !corsOriginAllowed("https://lk.example.com") {
		t.Fatal("expected explicit origin from CORS_ALLOWED_ORIGINS")
	}
}
