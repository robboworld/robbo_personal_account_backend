package usersearch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestNewFromConfigDoesNotWaitForElasticsearch(t *testing.T) {
	release := make(chan struct{})
	es := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select { // hang like an overloaded cluster
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(release); es.Close() })
	viper.Set("elasticsearch.enabled", true)
	viper.Set("elasticsearch.url", es.URL)
	viper.Set("elasticsearch.reindexIntervalMinutes", 60)
	t.Cleanup(func() {
		viper.Set("elasticsearch.enabled", false)
		viper.Set("elasticsearch.url", "")
		viper.Set("elasticsearch.reindexIntervalMinutes", 0)
	})

	start := time.Now()
	s := NewFromConfig()
	if d := time.Since(start); d > time.Second {
		t.Fatalf("NewFromConfig blocked for %v", d)
	}
	if s.ready() {
		t.Fatal("ready before Elasticsearch answered")
	}
	start = time.Now()
	s.Stop()
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Stop blocked for %v", d)
	}
}
