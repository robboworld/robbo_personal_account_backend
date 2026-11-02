package lmsdb

import (
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func TestRoleFromFlags(t *testing.T) {
	cases := []struct {
		name                               string
		isStaff, isSuperuser, isCourseTeam bool
		want                               models.Role
	}{
		{"plain user", false, false, false, models.Student},
		{"course team member", false, false, true, models.Teacher},
		{"global staff", true, false, false, models.Teacher},
		{"global staff on a course team", true, false, true, models.Teacher},
		{"superuser", false, true, false, models.SuperAdmin},
		{"superuser on a course team", true, true, true, models.SuperAdmin},
	}
	for _, c := range cases {
		if got := RoleFromFlags(c.isStaff, c.isSuperuser, c.isCourseTeam); got != c.want {
			t.Errorf("%s: RoleFromFlags(%v, %v, %v) = %v, want %v",
				c.name, c.isStaff, c.isSuperuser, c.isCourseTeam, got, c.want)
		}
	}
}
