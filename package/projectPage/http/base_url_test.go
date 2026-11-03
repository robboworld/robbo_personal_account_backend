package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBackendBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		headers map[string]string
		want    string
	}{
		{name: "direct", host: "localhost:8080", want: "http://localhost:8080"},
		{
			name:    "proxy rewrites host",
			host:    "127.0.0.1:8080",
			headers: map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "lk.example.com"},
			want:    "https://lk.example.com",
		},
		{
			name:    "first forwarded value",
			host:    "127.0.0.1:8080",
			headers: map[string]string{"X-Forwarded-Proto": "https, http", "X-Forwarded-Host": "lk.example.com, inner"},
			want:    "https://lk.example.com",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/projectPage/1/play-token", nil)
			req.Host = tc.host
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req
			if got := backendBaseURL(c); got != tc.want {
				t.Fatalf("backendBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}
