package profileaccess

import (
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func TestCanRead(t *testing.T) {
	TeacherStudentReader = func(teacherID, studentID string) bool {
		return teacherID == "10" && studentID == "797"
	}
	t.Cleanup(func() { TeacherStudentReader = nil })

	cases := []struct {
		name   string
		viewer string
		role   models.Role
		target string
		want   bool
	}{
		{"super admin reads anyone", "800", models.SuperAdmin, "1", true},
		{"own profile", "797", models.Student, "797", true},
		{"teacher reads a student of their class", "10", models.Teacher, "797", true},
		{"teacher cannot read a stranger", "10", models.Teacher, "1", false},
		{"teacher cannot read an admin", "10", models.Teacher, "800", false},
		{"parent cannot read a stranger", "20", models.Parent, "797", false},
		{"unit admin cannot read a stranger", "30", models.UnitAdmin, "797", false},
		{"anonymous cannot read", "0", models.Anonymous, "797", false},
		{"empty target", "10", models.Teacher, "", false},
	}
	for _, tc := range cases {
		if got := CanRead(tc.viewer, tc.role, tc.target); got != tc.want {
			t.Errorf("%s: CanRead(%q, %v, %q) = %v, want %v", tc.name, tc.viewer, tc.role, tc.target, got, tc.want)
		}
	}
}

func TestCanReadWithoutClassWiringDeniesTeacher(t *testing.T) {
	TeacherStudentReader = nil
	if CanRead("10", models.Teacher, "797") {
		t.Fatal("without class membership wiring a teacher must not read other profiles")
	}
}
