package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

const (
	defaultMaxCoTeachers     = 3
	defaultCoTeacherMaxDays  = 90
	coTeacherRemindBefore    = 24 * time.Hour
	defaultCoTeacherInterval = time.Hour
)

// maxCoTeachers is how many co-teachers a class may have at a time.
func maxCoTeachers() int {
	if n := viper.GetInt("classes.maxCoTeachers"); n > 0 {
		return n
	}
	return defaultMaxCoTeachers
}

// coTeacherMaxDays is the longest term, counted from each appointment or extension.
func coTeacherMaxDays() int {
	if n := viper.GetInt("classes.coTeacherMaxDays"); n > 0 {
		return n
	}
	return defaultCoTeacherMaxDays
}

func checkCoTeacherTerm(expiresAt, now time.Time) error {
	if !expiresAt.After(now) || expiresAt.After(now.AddDate(0, 0, coTeacherMaxDays())) {
		return teacherclass.ErrBadCoTeacherTerm
	}
	return nil
}

// coTeacherRemindedAt is the reminder mark of a new term: one ending within the reminder window
// is on both sides' minds already, so it counts as reminded.
func coTeacherRemindedAt(expiresAt, now time.Time) *time.Time {
	if expiresAt.Sub(now) <= coTeacherRemindBefore {
		return &now
	}
	return nil
}

// lookupTeacherByEmail returns the id of the active user with the email if the LK role of that
// user is Teacher, otherwise "". A SuperAdmin already manages every class.
// A variable so tests can stand in for LMS MySQL.
var lookupTeacherByEmail = func(email string) (string, error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	profile, err := reader.LookupAuthUserProfileByEmail(email)
	if err != nil || profile == nil || !profile.IsActive {
		return "", err
	}
	if reader.ResolveRole(profile.ID, profile.IsStaff, profile.IsSuperuser) != models.Teacher {
		return "", nil
	}
	return strconv.FormatInt(profile.ID, 10), nil
}

func coTeacherDTO(row *models.ClassCoTeacherDB) *teacherclass.CoTeacherDTO {
	return &teacherclass.CoTeacherDTO{
		TeacherID: row.TeacherID,
		Name:      ownerDisplayName(row.TeacherID),
		GrantedAt: row.GrantedAt,
		ExpiresAt: row.ExpiresAt,
	}
}

func (u *UseCaseImpl) ListCoTeachers(teacherID string, role models.Role, classID string) ([]teacherclass.CoTeacherDTO, error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	rows, err := u.gw.ListCoTeachers(inv.ID)
	if err != nil {
		return nil, err
	}
	out := make([]teacherclass.CoTeacherDTO, 0, len(rows))
	for i := range rows {
		out = append(out, *coTeacherDTO(&rows[i]))
	}
	return out, nil
}

func (u *UseCaseImpl) AddCoTeacher(teacherID string, role models.Role, classID, email string, expiresAt time.Time) (*teacherclass.CoTeacherDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expiresAt = expiresAt.UTC()
	if err := checkCoTeacherTerm(expiresAt, now); err != nil {
		return nil, err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, teacherclass.ErrBadRequest
	}
	coTeacherID, err := lookupTeacherByEmail(email)
	if err != nil {
		return nil, err
	}
	if coTeacherID == "" {
		return nil, teacherclass.ErrNotATeacher
	}
	if coTeacherID == inv.OwnerTeacherID {
		return nil, teacherclass.ErrAlreadyCoTeacher
	}
	row := &models.ClassCoTeacherDB{
		ClassID:    inv.ID,
		TeacherID:  coTeacherID,
		GrantedBy:  teacherID,
		ExpiresAt:  expiresAt,
		RemindedAt: coTeacherRemindedAt(expiresAt, now),
	}
	if err := u.gw.AddCoTeacher(row, maxCoTeachers()); err != nil {
		return nil, err
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryCoTeacherAdded, models.ClassHistoryEventDB{
		TargetUserID: coTeacherID, Detail: expiresAt.Format(time.RFC3339),
	})
	u.notifyCoTeacherChange(teacherID, inv, coTeacherID,
		"Назначен со-преподаватель",
		fmt.Sprintf("Класс «%s»: %s, до %s.", inv.DisplayName, teacherName(coTeacherID), coTeacherDate(expiresAt)))
	return coTeacherDTO(row), nil
}

func (u *UseCaseImpl) ExtendCoTeacher(teacherID string, role models.Role, classID, coTeacherID string, expiresAt time.Time) (*teacherclass.CoTeacherDTO, error) {
	inv, err := u.requireOwner(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expiresAt = expiresAt.UTC()
	if err := checkCoTeacherTerm(expiresAt, now); err != nil {
		return nil, err
	}
	row, err := u.gw.SetCoTeacherExpiry(inv.ID, coTeacherID, expiresAt, coTeacherRemindedAt(expiresAt, now))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, teacherclass.ErrNotFound
		}
		return nil, err
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryCoTeacherExtended, models.ClassHistoryEventDB{
		TargetUserID: coTeacherID, Detail: expiresAt.Format(time.RFC3339),
	})
	u.notifyCoTeacherChange(teacherID, inv, coTeacherID,
		"Срок со-преподавателя изменён",
		fmt.Sprintf("Класс «%s»: %s, теперь до %s.", inv.DisplayName, teacherName(coTeacherID), coTeacherDate(expiresAt)))
	return coTeacherDTO(row), nil
}

// RemoveCoTeacher ends the term at once: for the owner or a SuperAdmin, or a co-teacher stepping
// down. What the co-teacher did in the class stays.
func (u *UseCaseImpl) RemoveCoTeacher(teacherID string, role models.Role, classID, coTeacherID string) error {
	inv, err := u.gw.GetInviteByID(classID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return teacherclass.ErrNotFound
		}
		return err
	}
	if teacherID != coTeacherID && !canManage(inv, teacherID, role) {
		return teacherclass.ErrForbidden
	}
	if err := u.gw.RevokeCoTeacher(inv.ID, coTeacherID, teacherID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return teacherclass.ErrNotFound
		}
		return err
	}
	action := models.HistoryCoTeacherRemoved
	if teacherID == coTeacherID {
		action = models.HistoryCoTeacherLeft
	}
	u.recordTeacherAction(inv, teacherID, role, action, models.ClassHistoryEventDB{TargetUserID: coTeacherID})
	u.notifyCoTeacherChange(teacherID, inv, coTeacherID,
		"Со-преподавание завершено",
		fmt.Sprintf("Класс «%s»: %s больше не со-преподаватель. Работы и задания остались в классе.",
			inv.DisplayName, teacherName(coTeacherID)))
	return nil
}

// RemindExpiringCoTeachers needs no other cleanup: an ended term stops counting by itself.
func (u *UseCaseImpl) RemindExpiringCoTeachers(now time.Time) (int, error) {
	rows, err := u.gw.ClaimExpiringCoTeachers(now.Add(coTeacherRemindBefore))
	if err != nil {
		return 0, err
	}
	for i := range rows {
		row := &rows[i]
		inv, err := u.gw.GetInviteByID(row.ClassID)
		if err != nil {
			log.Printf("teacherclass: remind co-teacher of class %s: %v", row.ClassID, err)
			continue
		}
		u.notifyCoTeacherChange("", inv, row.TeacherID,
			"Срок со-преподавателя заканчивается",
			fmt.Sprintf("Класс «%s»: доступ %s закончится %s. Продлить срок может владелец класса.",
				inv.DisplayName, teacherName(row.TeacherID), coTeacherDate(row.ExpiresAt)))
	}
	return len(rows), nil
}

func teacherName(id string) string {
	if name := ownerDisplayName(id); name != "" {
		return name
	}
	return "учитель"
}

// coTeacherDate: terms end late in the evening of a day in Russia, which is that day in UTC too.
func coTeacherDate(t time.Time) string {
	return t.UTC().Format("02.01.2006")
}

// notifyCoTeacherChange tells the co-teacher and the owner, except whoever made the change
// (actorID, "" for the reminder). One row per class and teacher for each of them: the newest
// news about the term replaces the older. The change has already happened, so a failure to
// send is only logged.
func (u *UseCaseImpl) notifyCoTeacherChange(actorID string, inv *models.ClassInviteDB, coTeacherID, title, body string) {
	for _, recipient := range []string{coTeacherID, inv.OwnerTeacherID} {
		if recipient == actorID {
			continue
		}
		// The owner's row must not take the co-teacher's over: the key is unique across recipients.
		row := classNotification(recipient, inv, "class_co_teacher",
			"class_co_teacher:"+inv.ID+":"+coTeacherID+":"+recipient)
		row.Title = title
		row.Body = body
		if err := u.notify.CreateOrUpdateByDedupe(row); err != nil {
			log.Printf("teacherclass: notify class %s co-teacher change: %v", inv.ID, err)
		}
	}
}

// StartCoTeacherReminderWorker sends the "term ends within a day" reminders. Access does not
// depend on it: an ended term stops counting on its own.
func StartCoTeacherReminderWorker(uc teacherclass.UseCase, lc fx.Lifecycle) {
	if uc == nil {
		return
	}
	interval := viper.GetDuration("classes.coTeacherReminderInterval")
	if interval < time.Minute {
		interval = defaultCoTeacherInterval
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer close(doneCh)
				run := func() {
					if _, err := uc.RemindExpiringCoTeachers(time.Now().UTC()); err != nil {
						log.Printf("teacherclass: co-teacher reminders: %v", err)
					}
				}
				run()
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						run()
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(stopCh)
			select {
			case <-doneCh:
			case <-ctx.Done():
			}
			return nil
		},
	})
}
