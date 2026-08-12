package lmsdb

import (
	"encoding/json"
	"testing"
)

func TestParseSetLoginStreakInMeta(t *testing.T) {
	merged, err := SetLoginStreakInMeta(`{"lk_avatar_id":"ava1"}`, LoginStreakMeta{
		Current:       3,
		Longest:       5,
		LastLoginDate: "2026-07-31",
		Timezone:      "Europe/Moscow",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := ParseLoginStreakFromMeta(merged)
	if got.Current != 3 || got.Longest != 5 || got.LastLoginDate != "2026-07-31" {
		t.Fatalf("got %+v", got)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(merged), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["lk_avatar_id"] != "ava1" {
		t.Fatalf("avatar key lost: %v", payload)
	}
}

func TestParseLoginStreakFromMeta_Empty(t *testing.T) {
	if got := ParseLoginStreakFromMeta(""); got.Current != 0 {
		t.Fatalf("%+v", got)
	}
}
