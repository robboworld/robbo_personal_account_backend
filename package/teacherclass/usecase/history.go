package usecase

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// The class history for its teachers. Like the bell it is best effort: the action has already
// happened, so a failed write is only logged. Reading and a student's work on a draft are not
// recorded: teachers do not see drafts, and the history must not tell them more.

// historyPageSize keeps name lookups in LMS MySQL small: one per person on the page.
const historyPageSize = 50

// historyCategories are the filters of the history screen, by action.
var historyCategories = map[string][]string{
	"class": {
		models.HistoryClassCreated, models.HistoryClassRenamed, models.HistoryClassArchived, models.HistoryInviteRotated,
	},
	"members": {
		models.HistoryMemberJoined, models.HistoryMemberAdded, models.HistoryMemberRemoved, models.HistoryMemberLeft,
	},
	"teachers": {
		models.HistoryCoTeacherAdded, models.HistoryCoTeacherExtended, models.HistoryCoTeacherRemoved, models.HistoryCoTeacherLeft,
	},
	"assignments": {
		models.HistoryAssignmentCreated, models.HistoryAssignmentClosed, models.HistoryAssignmentReopened, models.HistoryAssignmentIssuedLate,
	},
	"work":     {models.HistoryWorkSubmitted, models.HistoryWorkReviewed, models.HistoryWorkEdited, models.HistoryWorkRestored},
	"messages": {models.HistoryMessageSent},
}

func (u *UseCaseImpl) recordHistory(inv *models.ClassInviteDB, actorID, actorRole, action string, e models.ClassHistoryEventDB) {
	e.ClassID = inv.ID
	e.ActorUserID = actorID
	e.ActorRole = actorRole
	e.Action = action
	if err := u.gw.AddHistoryEvent(&e); err != nil {
		log.Printf("teacherclass: class %s history %s: %v", inv.ID, action, err)
	}
}

// historyTeacherRole is who the teacher is in the class. It comes after the access check.
func historyTeacherRole(inv *models.ClassInviteDB, teacherID string, role models.Role) string {
	switch {
	case inv.OwnerTeacherID == teacherID:
		return models.HistoryActorOwner
	case role == models.SuperAdmin:
		return models.HistoryActorAdmin
	}
	return models.HistoryActorCoTeacher
}

func (u *UseCaseImpl) recordTeacherAction(inv *models.ClassInviteDB, teacherID string, role models.Role, action string, e models.ClassHistoryEventDB) {
	u.recordHistory(inv, teacherID, historyTeacherRole(inv, teacherID, role), action, e)
}

func (u *UseCaseImpl) recordStudentAction(inv *models.ClassInviteDB, studentID, action string, e models.ClassHistoryEventDB) {
	u.recordHistory(inv, studentID, models.HistoryActorStudent, action, e)
}

// recordSubmitted: a hand-in of either kind of work.
func (u *UseCaseImpl) recordSubmitted(inv *models.ClassInviteDB, studentID string, a *models.ClassAssignmentDB, handIn *teacherclass.HandIn) {
	e := models.ClassHistoryEventDB{Title: a.Title}
	if lateAt(a, &handIn.At) {
		e.Detail = models.HistoryLate
	}
	u.recordStudentAction(inv, studentID, models.HistoryWorkSubmitted, e)
}

// RecordTeacherEdit is called by projectPage, which saves the .sb3 and knows nothing about classes.
// Only who runs the class gets here (access.TeacherCanEdit), a SuperAdmin does not.
func (u *UseCaseImpl) RecordTeacherEdit(teacherID string, p *models.ScratchProjectDB, restored bool) {
	inv, err := u.gw.GetInviteByCourseCohort(deref(p.EdxCourseID), deref(p.EdxCohortID))
	if err != nil {
		log.Printf("teacherclass: project %s edit history: %v", p.ID, err)
		return
	}
	action := models.HistoryWorkEdited
	if restored {
		action = models.HistoryWorkRestored
	}
	u.recordTeacherAction(inv, teacherID, models.Teacher, action, models.ClassHistoryEventDB{
		TargetUserID: p.OwnerUserID, Title: p.Title,
	})
	u.notifyWorkEdited(teacherID, inv, p, restored)
}

// RecordClassMessage is called by notifications, which sends the message and knows nothing about
// classes. classID comes from the client: a teacher can only write into a class they teach.
func (u *UseCaseImpl) RecordClassMessage(teacherID string, role models.Role, classID string, recipientIDs []string) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		log.Printf("teacherclass: class %s message history: %v", classID, err)
		return
	}
	e := models.ClassHistoryEventDB{Detail: strconv.Itoa(len(recipientIDs))}
	if len(recipientIDs) == 1 {
		e.TargetUserID = recipientIDs[0]
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryMessageSent, e)
}

func historyCursor(e *models.ClassHistoryEventDB) string {
	return fmt.Sprintf("%d_%s", e.CreatedAt.UnixMicro(), e.ID)
}

func parseHistoryCursor(cursor string) (*teacherclass.HistoryCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	micros, id, ok := strings.Cut(cursor, "_")
	if !ok {
		return nil, teacherclass.ErrBadRequest
	}
	at, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return nil, teacherclass.ErrBadRequest
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, teacherclass.ErrBadRequest
	}
	return &teacherclass.HistoryCursor{At: time.UnixMicro(at).UTC(), ID: id}, nil
}

// ListHistory returns a page of the class's history, newest first; category "" is everything
// and cursor is the NextCursor of the previous page. The names are today's, from the LMS.
func (u *UseCaseImpl) ListHistory(teacherID string, role models.Role, classID, category, cursor string) (*teacherclass.HistoryPageDTO, error) {
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	actions, ok := historyCategories[category]
	if category != "" && !ok {
		return nil, teacherclass.ErrBadRequest
	}
	before, err := parseHistoryCursor(cursor)
	if err != nil {
		return nil, err
	}
	rows, err := u.gw.ListHistoryEvents(inv.ID, actions, before, historyPageSize+1)
	if err != nil {
		return nil, err
	}
	page := &teacherclass.HistoryPageDTO{Items: make([]teacherclass.HistoryEventDTO, 0, len(rows))}
	if len(rows) > historyPageSize {
		rows = rows[:historyPageSize]
		page.NextCursor = historyCursor(&rows[historyPageSize-1])
	}
	names := make(map[string]string)
	nameOf := func(id string) string {
		if id == "" {
			return ""
		}
		name, ok := names[id]
		if !ok {
			name = ownerDisplayName(id)
			names[id] = name
		}
		return name
	}
	for i := range rows {
		e := &rows[i]
		page.Items = append(page.Items, teacherclass.HistoryEventDTO{
			ID:         e.ID,
			Action:     e.Action,
			ActorID:    e.ActorUserID,
			ActorName:  nameOf(e.ActorUserID),
			ActorRole:  e.ActorRole,
			TargetID:   e.TargetUserID,
			TargetName: nameOf(e.TargetUserID),
			Title:      e.Title,
			Detail:     e.Detail,
			CreatedAt:  e.CreatedAt,
		})
	}
	return page, nil
}

// lmsUserID is the id behind an LMS login, "" when unknown. A variable so tests can stand in
// for LMS MySQL.
var lmsUserID = func(username string) string {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil || reader == nil {
		return ""
	}
	defer reader.Close()
	row, err := reader.LookupAuthUserByUsername(username)
	if err != nil || row == nil {
		return ""
	}
	return strconv.FormatInt(row.ID, 10)
}
