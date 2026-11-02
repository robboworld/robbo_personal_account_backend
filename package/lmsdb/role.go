package lmsdb

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// courseTeamRolesSQL selects the Open edX course team roles (Studio "Staff" and "Admin") in
// student_courseaccessrole. A member of at least one course team is a teacher in the LK.
const courseTeamRolesSQL = `role IN ('staff', 'instructor') AND course_id <> ''`

// courseTeamLookupChunk caps one IN (...) list in CourseTeamMemberIDs.
const courseTeamLookupChunk = 500

// RoleFromFlags maps LMS flags to the LK role: superuser → SuperAdmin; global staff or a
// course team member → Teacher; anyone else → Student.
func RoleFromFlags(isStaff, isSuperuser, isCourseTeam bool) models.Role {
	if isSuperuser {
		return models.SuperAdmin
	}
	if isStaff || isCourseTeam {
		return models.Teacher
	}
	return models.Student
}

// ResolveRole is RoleFromFlags with the course team lookup. When that lookup fails the user
// gets the role from the flags alone (no teacher access through a course team).
func (r *Reader) ResolveRole(userID int64, isStaff, isSuperuser bool) models.Role {
	if isStaff || isSuperuser {
		return RoleFromFlags(isStaff, isSuperuser, false)
	}
	isCourseTeam, err := r.IsCourseTeamMember(userID)
	if err != nil {
		log.Printf("lmsdb: course team lookup for user %d: %v", userID, err)
	}
	return RoleFromFlags(false, false, isCourseTeam)
}

// ResolveRoleFromConfig is ResolveRole on the configured LMS reader.
func ResolveRoleFromConfig(userID int64, isStaff, isSuperuser bool) models.Role {
	reader, err := NewReaderFromConfig()
	if err != nil {
		log.Printf("lmsdb: reader for role of user %d: %v", userID, err)
		return RoleFromFlags(isStaff, isSuperuser, false)
	}
	defer reader.Close()
	return reader.ResolveRole(userID, isStaff, isSuperuser)
}

// IsCourseTeamMember reports whether the user is staff or instructor on at least one course.
func (r *Reader) IsCourseTeamMember(userID int64) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var one int
	err := r.db.QueryRow(
		`SELECT 1 FROM student_courseaccessrole WHERE user_id = ? AND `+courseTeamRolesSQL+` LIMIT 1`,
		userID,
	).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CourseTeamMemberIDs returns which of the given users are on at least one course team.
func (r *Reader) CourseTeamMemberIDs(ids []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	var valid []int64
	for _, id := range ids {
		if _, seen := out[id]; id > 0 && !seen {
			out[id] = false
			valid = append(valid, id)
		}
	}
	for start := 0; start < len(valid); start += courseTeamLookupChunk {
		end := start + courseTeamLookupChunk
		if end > len(valid) {
			end = len(valid)
		}
		chunk := valid[start:end]
		args := make([]interface{}, 0, len(chunk))
		for _, id := range chunk {
			args = append(args, id)
		}
		rows, err := r.db.Query(
			`SELECT DISTINCT user_id FROM student_courseaccessrole WHERE user_id IN (`+
				strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")+`) AND `+courseTeamRolesSQL,
			args...,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			out[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ListCourseTeamCourseIDs returns the courses where the user is staff or instructor.
func (r *Reader) ListCourseTeamCourseIDs(userID int64) ([]string, error) {
	if userID <= 0 {
		return nil, nil
	}
	rows, err := r.db.Query(
		`SELECT DISTINCT course_id FROM student_courseaccessrole WHERE user_id = ? AND `+courseTeamRolesSQL,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
