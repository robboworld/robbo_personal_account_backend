package usecase

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
)

type fakeGateway struct {
	notifications.Gateway
	batches [][]models.UserNotificationDB
	sends   map[string]int // sends per sender already in the window
}

func (f *fakeGateway) CreateTeacherSend(senderID, sendID string, rows []models.UserNotificationDB, dailyLimit int, since time.Time) (int, error) {
	used := f.sends[senderID]
	if dailyLimit > 0 && used >= dailyLimit {
		return used, notifications.ErrTeacherDailyLimit
	}
	if f.sends == nil {
		f.sends = map[string]int{}
	}
	f.sends[senderID]++
	f.batches = append(f.batches, rows)
	return used, nil
}

func (f *fakeGateway) CountTeacherSends(senderID string, since time.Time) (int, error) {
	return f.sends[senderID], nil
}

// stubLMS stands in for LMS MySQL: teacher 10 ("Анна Иванова") teaches 797 and 798; 999 is someone else's.
func stubLMS(t *testing.T) {
	prevStudent, prevName := teacherHasStudent, senderDisplayName
	teacherHasStudent = func(teacherID, studentID string) bool {
		return teacherID == "10" && (studentID == "797" || studentID == "798")
	}
	senderDisplayName = func(userID string) string {
		if userID == "10" {
			return "Анна Иванова"
		}
		return ""
	}
	t.Cleanup(func() { teacherHasStudent, senderDisplayName = prevStudent, prevName })
}

func TestSendFromTeacher(t *testing.T) {
	stubLMS(t)

	msg := func(sender string, role models.Role, ids ...string) notifications.TeacherMessageInput {
		return notifications.TeacherMessageInput{
			SenderUserID:     sender,
			SenderRole:       role,
			RecipientUserIDs: ids,
			Title:            "Урок перенесён",
			Body:             "Занятие в четверг в 16:00",
		}
	}

	cases := []struct {
		name     string
		input    notifications.TeacherMessageInput
		wantErr  error
		wantSent int
	}{
		{"own student", msg("10", models.Teacher, "797"), nil, 1},
		{"whole class, duplicates and blanks dropped", msg("10", models.Teacher, "797", " 798 ", "797", ""), nil, 2},
		{"someone else's student", msg("10", models.Teacher, "999"), ErrNotYourStudent, 0},
		{"one stranger rejects the whole send", msg("10", models.Teacher, "797", "999"), ErrNotYourStudent, 0},
		{"another teacher", msg("11", models.Teacher, "797"), ErrNotYourStudent, 0},
		{"superadmin writes to anyone", msg("1", models.SuperAdmin, "999"), nil, 1},
		{"no recipients", msg("10", models.Teacher), ErrInvalidInput, 0},
		{"no sender", msg("", models.Teacher, "797"), ErrInvalidInput, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{}
			uc := &NotificationUseCase{gateway: gw}
			sent, _, err := uc.SendFromTeacher(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if sent != tc.wantSent {
				t.Fatalf("sent = %d, want %d", sent, tc.wantSent)
			}
			if tc.wantErr != nil && len(gw.batches) != 0 {
				t.Fatalf("rejected send still wrote %d batches", len(gw.batches))
			}
			if tc.wantSent > 0 {
				for _, row := range gw.batches[0] {
					if row.Source != "teacher" || row.Kind != "teacher_message" || row.ActionURL != nil {
						t.Fatalf("row = %+v, want teacher source, teacher_message kind, no link", row)
					}
				}
			}
		})
	}
}

func TestSendFromTeacherLimits(t *testing.T) {
	stubLMS(t)
	uc := &NotificationUseCase{gateway: &fakeGateway{}}
	base := notifications.TeacherMessageInput{
		SenderUserID: "10", SenderRole: models.Teacher, RecipientUserIDs: []string{"797"},
		Title: "t", Body: "b",
	}

	long := base
	long.Body = strings.Repeat("я", maxTeacherBodyRunes+1)
	if _, _, err := uc.SendFromTeacher(long); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("over-long body: err = %v", err)
	}

	// Cyrillic counts in characters, not bytes.
	fits := base
	fits.Body = strings.Repeat("я", maxTeacherBodyRunes)
	if _, _, err := uc.SendFromTeacher(fits); err != nil {
		t.Fatalf("body at the limit: err = %v", err)
	}

	many := base
	many.SenderRole = models.SuperAdmin // skips the class check, so only the count matters
	many.RecipientUserIDs = nil
	for i := 0; i <= maxTeacherRecipients; i++ {
		many.RecipientUserIDs = append(many.RecipientUserIDs, strings.Repeat("1", i+1))
	}
	if _, _, err := uc.SendFromTeacher(many); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("too many recipients: err = %v", err)
	}
}

func TestSendFromTeacherTitleHasSenderName(t *testing.T) {
	stubLMS(t)
	gw := &fakeGateway{}
	uc := &NotificationUseCase{gateway: gw}

	send := func(sender string, role models.Role, to string) string {
		gw.batches = nil
		_, _, err := uc.SendFromTeacher(notifications.TeacherMessageInput{
			SenderUserID: sender, SenderRole: role, RecipientUserIDs: []string{to},
			Title: " Урок перенесён ", Body: "b",
		})
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		return gw.batches[0][0].Title
	}

	if got := send("10", models.Teacher, "797"); got != "Анна Иванова: Урок перенесён" {
		t.Fatalf("title = %q", got)
	}
	// No name in the LMS: the title stays as written.
	if got := send("1", models.SuperAdmin, "797"); got != "Урок перенесён" {
		t.Fatalf("title without a name = %q", got)
	}
}

func TestSendFromTeacherDailyLimit(t *testing.T) {
	stubLMS(t)
	gw := &fakeGateway{}
	uc := &NotificationUseCase{gateway: gw}
	toClass := notifications.TeacherMessageInput{
		SenderUserID: "10", SenderRole: models.Teacher, RecipientUserIDs: []string{"797", "798"},
		Title: "t", Body: "b",
	}

	// A message to the whole class is one send, however many students it reaches.
	for i := 1; i <= teacherDailySends; i++ {
		sent, quota, err := uc.SendFromTeacher(toClass)
		if err != nil || sent != 2 {
			t.Fatalf("send %d: sent=%d err=%v", i, sent, err)
		}
		if quota.Used != i || quota.Remaining != teacherDailySends-i {
			t.Fatalf("send %d: quota = %+v", i, quota)
		}
	}
	_, quota, err := uc.SendFromTeacher(toClass)
	if !errors.Is(err, notifications.ErrTeacherDailyLimit) {
		t.Fatalf("send over the limit: err = %v", err)
	}
	if quota.Remaining != 0 || len(gw.batches) != teacherDailySends {
		t.Fatalf("over the limit: quota=%+v batches=%d", quota, len(gw.batches))
	}
	if q, _ := uc.TeacherQuota("10", models.Teacher); q.Remaining != 0 || q.Limit != teacherDailySends {
		t.Fatalf("TeacherQuota = %+v", q)
	}

	// Another teacher's sends do not count against this one; SuperAdmin has no limit.
	if q, _ := uc.TeacherQuota("11", models.Teacher); q.Remaining != teacherDailySends {
		t.Fatalf("other teacher quota = %+v", q)
	}
	admin := toClass
	admin.SenderUserID, admin.SenderRole = "1", models.SuperAdmin
	for i := 0; i < teacherDailySends+1; i++ {
		if _, _, err := uc.SendFromTeacher(admin); err != nil {
			t.Fatalf("superadmin send %d: %v", i, err)
		}
	}
}
