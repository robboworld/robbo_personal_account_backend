//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	NOTIFICATIONS_TEST_DSN="host=... dbname=..." go test -tags integration ./package/notifications/gateway/
package gateway

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func digestRow(t *testing.T, g *NotificationGateway, key string) models.UserNotificationDB {
	t.Helper()
	var rows []models.UserNotificationDB
	if err := g.db.Where("dedupe_key = ?", key).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	return rows[0]
}

func TestAddToDigest(t *testing.T) {
	dsn := os.Getenv("NOTIFICATIONS_TEST_DSN")
	if dsn == "" {
		t.Skip("NOTIFICATIONS_TEST_DSN is not set")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	g := &NotificationGateway{db: db}
	key, url := "digest-test:"+uuid.NewString(), "/teacher/classes/c1"
	t.Cleanup(func() { db.Exec(`DELETE FROM user_notifications WHERE dedupe_key = ?`, key) })
	row := &models.UserNotificationDB{RecipientUserID: "t-" + uuid.NewString()[:8], Kind: "digest_test",
		Severity: "INFO", Source: "system", ActionURL: &url, DedupeKey: &key}
	add := func(item string) error { return g.AddToDigest(row, "Новых: ", "«12 + 1»: ", item) }

	if err := add("Аня Демо"); err != nil {
		t.Fatal(err)
	}
	got := digestRow(t, g, key)
	if got.Title != "Новых: 1" || got.Body != "«12 + 1»: Аня Демо" {
		t.Fatalf("first: %q / %q", got.Title, got.Body)
	}

	if err := add("Петя, 5А"); err != nil {
		t.Fatal(err)
	}
	got = digestRow(t, g, key)
	if got.Title != "Новых: 2" || got.Body != "«12 + 1»: Аня Демо, Петя, 5А" {
		t.Fatalf("second: %q / %q", got.Title, got.Body)
	}

	// The same name again is another event (a namesake, or someone back): it counts, and the
	// row comes back unread.
	if err := db.Exec(`UPDATE user_notifications SET read_at = now() WHERE dedupe_key = ?`, key).Error; err != nil {
		t.Fatal(err)
	}
	if err := add("Аня Демо"); err != nil {
		t.Fatal(err)
	}
	got = digestRow(t, g, key)
	if got.Title != "Новых: 3" || got.Body != "«12 + 1»: Аня Демо, Петя, 5А, Аня Демо" || got.ReadAt != nil {
		t.Fatalf("same name: %q / %q read=%v", got.Title, got.Body, got.ReadAt)
	}

	// A whole class joining at once: nobody lost, nobody counted twice.
	var wg sync.WaitGroup
	errs := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- add(fmt.Sprintf("Ученик-%02d", i))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got = digestRow(t, g, key)
	if got.Title != "Новых: 33" {
		t.Fatalf("concurrent title = %q", got.Title)
	}
	for i := 0; i < 30; i++ {
		if !strings.Contains(got.Body, fmt.Sprintf("Ученик-%02d", i)) {
			t.Fatalf("lost %d: %q", i, got.Body)
		}
	}

	// Past the limit the count goes on and the body stops.
	for i := 0; i < 200; i++ {
		if err := add(fmt.Sprintf("Длинное имя ученика номер %03d", i)); err != nil {
			t.Fatal(err)
		}
	}
	got = digestRow(t, g, key)
	if got.Title != "Новых: 233" || len([]rune(got.Body)) > digestBodyLimit+40 {
		t.Fatalf("limit: %q, body %d runes", got.Title, len([]rune(got.Body)))
	}
}
