package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeLMS answers like the LMS service API; the first data call gets 401 when expireOnce is set.
func fakeLMS(t *testing.T, expireOnce bool) (*httptest.Server, *[]map[string]interface{}, *int32) {
	t.Helper()
	var reads []map[string]interface{}
	var tokens int32
	var expired int32
	if !expireOnce {
		expired = 1
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth2/access_token":
			if err := r.ParseForm(); err != nil || r.Form.Get("grant_type") != "client_credentials" ||
				r.Form.Get("client_id") != "lk-service" || r.Form.Get("client_secret") != "s3cret" ||
				r.Form.Get("token_type") != "jwt" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			n := atomic.AddInt32(&tokens, 1)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "tok" + string(rune('0'+n)), "expires_in": 3600})
		case strings.HasPrefix(r.URL.Path, lmsServicePath):
			if !strings.HasPrefix(r.Header.Get("Authorization"), "JWT tok") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if atomic.CompareAndSwapInt32(&expired, 0, 1) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			switch strings.TrimPrefix(r.URL.Path, lmsServicePath) {
			case "feed/":
				if r.URL.Query().Get("user_id") != "789" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				_, _ = w.Write([]byte(`{"enabled": true, "next_before": null,
					"links": {"feed": "https://lms/account/notifications", "settings": "https://lms/account/#notifications"},
					"items": [
					 {"id": 7, "type": "ora_grade_assigned", "label": "Задание проверено", "text": "Ваш ответ проверен",
					  "course_name": "Курс", "url": "https://lms/x", "created": "2026-10-27T10:00:00.123456+00:00",
					  "read": false, "read_at": null},
					 {"id": 6, "type": "course_updates", "label": "Обновление курса", "text": "Урок перенесён",
					  "course_name": "Курс", "url": "", "created": "2026-10-27T09:00:00+00:00",
					  "read": true, "read_at": "2026-10-27T09:30:00+00:00"}]}`))
			case "unread-count/":
				_, _ = w.Write([]byte(`{"enabled": true, "unread": 3, "unseen": 1}`))
			case "read/":
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				reads = append(reads, body)
				_, _ = w.Write([]byte(`{"read": 1}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &reads, &tokens
}

func TestLMSGatewayFeed(t *testing.T) {
	srv, _, _ := fakeLMS(t, false)
	g := NewLMSGateway(srv.URL+"/", "lk-service", "s3cret", srv.Client())
	feed, err := g.Feed("789", 30)
	if err != nil {
		t.Fatal(err)
	}
	if !feed.Enabled || len(feed.Items) != 2 || feed.Links == nil || feed.Links.Settings != "https://lms/account/#notifications" {
		t.Fatalf("feed = %+v", feed)
	}
	first, second := feed.Items[0], feed.Items[1]
	if first.FeedKind != "lms" || first.ID != "7" || first.Title != "Задание проверено" || first.Body != "Ваш ответ проверен" ||
		first.Source != "lms" || first.CourseName != "Курс" || first.ReadAt != nil || first.ActionURL == nil ||
		*first.ActionURL != "https://lms/x" || first.CreatedAt.Hour() != 10 {
		t.Fatalf("first = %+v", first)
	}
	if second.ReadAt == nil || second.ReadAt.Minute() != 30 || second.ActionURL != nil {
		t.Fatalf("second = %+v", second)
	}
}

func TestLMSGatewayRenewsTokenOn401(t *testing.T) {
	srv, _, tokens := fakeLMS(t, true)
	g := NewLMSGateway(srv.URL, "lk-service", "s3cret", srv.Client())
	count, err := g.UnreadCount("789")
	if err != nil || count != 3 {
		t.Fatalf("count = %d, err = %v", count, err)
	}
	if *tokens != 2 {
		t.Fatalf("tokens = %d, want 2 (renewed once after 401)", *tokens)
	}
	if _, err := g.UnreadCount("789"); err != nil || *tokens != 2 {
		t.Fatalf("second call: err = %v, tokens = %d (the token must be reused)", err, *tokens)
	}
}

func TestLMSGatewayMarkRead(t *testing.T) {
	srv, reads, _ := fakeLMS(t, false)
	g := NewLMSGateway(srv.URL, "lk-service", "s3cret", srv.Client())
	if err := g.MarkRead("789", []int64{7}); err != nil {
		t.Fatal(err)
	}
	if err := g.MarkAllRead("789"); err != nil {
		t.Fatal(err)
	}
	if len(*reads) != 2 || (*reads)[0]["user_id"].(float64) != 789 || (*reads)[1]["all"] != true {
		t.Fatalf("reads = %v", *reads)
	}
}

func TestLMSGatewayRejectsNonLMSUserIDs(t *testing.T) {
	srv, reads, tokens := fakeLMS(t, false)
	g := NewLMSGateway(srv.URL, "lk-service", "s3cret", srv.Client())
	for _, id := range []string{"", "abc", "0", "-5"} {
		if _, err := g.Feed(id, 10); err == nil {
			t.Fatalf("Feed(%q): no error", id)
		}
		if err := g.MarkAllRead(id); err == nil {
			t.Fatalf("MarkAllRead(%q): no error", id)
		}
	}
	if *tokens != 0 || len(*reads) != 0 {
		t.Fatal("no LMS call for a bad user id")
	}
}

func TestLMSGatewayWrongSecret(t *testing.T) {
	srv, _, _ := fakeLMS(t, false)
	g := NewLMSGateway(srv.URL, "lk-service", "wrong", srv.Client())
	if _, err := g.UnreadCount("789"); err == nil {
		t.Fatal("expected an error with a wrong secret")
	}
}
