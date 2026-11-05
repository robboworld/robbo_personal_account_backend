//go:build integration

// Runs against a projects database with the full schema (init scripts):
//
//	TEACHERCLASS_TEST_DSN="host=... dbname=..." go test -tags integration ./package/teacherclass/gateway/
package gateway

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

func TestHistoryEventsPagesAndFilters(t *testing.T) {
	g, db := fileTestGateway(t)
	id := uuid.NewString()
	inv := &models.ClassInviteDB{
		ID: id, EdxCohortID: models.UnlinkedCohortKey(id), OwnerTeacherID: "t-" + id[:8], DisplayName: "5А",
		InviteCode: "H" + id[:7], InviteSlug: "h-" + id[:8],
	}
	if err := g.CreateInvite(inv); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM class_invites WHERE id = ?`, inv.ID) })

	actions := []string{
		models.HistoryMemberJoined, models.HistoryWorkSubmitted, models.HistoryMemberLeft, models.HistoryWorkReviewed, models.HistoryMemberJoined,
	}
	for _, action := range actions {
		e := &models.ClassHistoryEventDB{ClassID: inv.ID, Action: action, ActorUserID: "797", ActorRole: models.HistoryActorStudent}
		if err := g.AddHistoryEvent(e); err != nil {
			t.Fatal(err)
		}
		if e.ID == "" {
			t.Fatal("no id after insert")
		}
	}
	// Two events in one moment must not fall between pages.
	if err := db.Exec(`UPDATE class_history_events SET created_at = date_trunc('second', now()) WHERE class_id = ?`, inv.ID).Error; err != nil {
		t.Fatal(err)
	}

	var seen []models.ClassHistoryEventDB
	var before *teacherclass.HistoryCursor
	for range actions {
		page, err := g.ListHistoryEvents(inv.ID, nil, before, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		seen = append(seen, page...)
		last := page[len(page)-1]
		before = &teacherclass.HistoryCursor{At: last.CreatedAt, ID: last.ID}
	}
	if len(seen) != len(actions) {
		t.Fatalf("paged over %d events, want %d", len(seen), len(actions))
	}
	ids := map[string]bool{}
	for _, e := range seen {
		ids[e.ID] = true
	}
	if len(ids) != len(actions) {
		t.Fatalf("a page repeated an event: %d distinct", len(ids))
	}

	members, err := g.ListHistoryEvents(inv.ID, []string{models.HistoryMemberJoined, models.HistoryMemberLeft}, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Fatalf("filtered = %d, want 3", len(members))
	}

	// Newest first, across the second that holds the rest.
	late := &models.ClassHistoryEventDB{ClassID: inv.ID, Action: models.HistoryClassRenamed, ActorUserID: "1", ActorRole: models.HistoryActorOwner}
	time.Sleep(time.Millisecond)
	if err := g.AddHistoryEvent(late); err != nil {
		t.Fatal(err)
	}
	newest, err := g.ListHistoryEvents(inv.ID, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(newest) != 1 || newest[0].ID != late.ID {
		t.Fatalf("newest = %+v", newest)
	}
}
