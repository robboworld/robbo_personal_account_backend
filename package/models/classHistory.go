package models

import "time"

// ClassHistoryEventDB is one line of a class's history. Detail by action:
//   - HistoryClassRenamed: the old name, Title is the new one;
//   - HistoryCoTeacherAdded, HistoryCoTeacherExtended: the end of the term, RFC 3339;
//   - HistoryAssignmentIssuedLate: how many copies were issued;
//   - HistoryWorkSubmitted: HistoryLate for late work;
//   - HistoryWorkReviewed: the review status;
//   - HistoryWorkEdited, HistoryWorkRestored: nothing, the blocks are the work itself;
//   - HistoryMessageSent: how many students got it; TargetUserID is set for one student;
//   - HistoryMemberRemoved: the LMS login when its user id could not be found.
type ClassHistoryEventDB struct {
	ID           string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	ClassID      string    `gorm:"column:class_id;type:uuid"`
	Action       string    `gorm:"column:action"`
	ActorUserID  string    `gorm:"column:actor_user_id"`
	ActorRole    string    `gorm:"column:actor_role"`
	TargetUserID string    `gorm:"column:target_user_id"`
	Title        string    `gorm:"column:title"`
	Detail       string    `gorm:"column:detail"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (ClassHistoryEventDB) TableName() string { return "class_history_events" }

// Who acted, in terms of this class.
const (
	HistoryActorOwner     = "owner"
	HistoryActorCoTeacher = "co_teacher"
	HistoryActorAdmin     = "admin"
	HistoryActorStudent   = "student"
)

const (
	HistoryClassCreated  = "class_created"
	HistoryClassRenamed  = "class_renamed"
	HistoryClassArchived = "class_archived"
	HistoryInviteRotated = "invite_rotated"

	HistoryMemberJoined  = "member_joined"
	HistoryMemberAdded   = "member_added"
	HistoryMemberRemoved = "member_removed"
	HistoryMemberLeft    = "member_left"

	HistoryCoTeacherAdded    = "co_teacher_added"
	HistoryCoTeacherExtended = "co_teacher_extended"
	HistoryCoTeacherRemoved  = "co_teacher_removed"
	HistoryCoTeacherLeft     = "co_teacher_left"

	HistoryAssignmentCreated    = "assignment_created"
	HistoryAssignmentClosed     = "assignment_closed"
	HistoryAssignmentReopened   = "assignment_reopened"
	HistoryAssignmentIssuedLate = "assignment_issued_late"

	HistoryWorkSubmitted = "work_submitted"
	HistoryWorkReviewed  = "work_reviewed"
	HistoryWorkEdited    = "work_edited"
	HistoryWorkRestored  = "work_restored"

	HistoryMessageSent = "message_sent"
)

// HistoryLate is the detail of HistoryWorkSubmitted for work handed in after the due date.
const HistoryLate = "late"
