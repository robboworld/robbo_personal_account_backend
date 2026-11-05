package access

import (
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func TestResolve_ownerPrivate(t *testing.T) {
	p := &models.ScratchProjectDB{OwnerUserID: "user-1", IsPublic: false}
	acc := Resolve("user-1", p)
	if !acc.IsOwner || !acc.CanRead || !acc.CanWrite {
		t.Fatalf("owner private: got %+v", acc)
	}
}

func TestResolve_ownerPublic(t *testing.T) {
	p := &models.ScratchProjectDB{OwnerUserID: "user-1", IsPublic: true}
	acc := Resolve("user-1", p)
	if !acc.IsOwner || !acc.CanRead || !acc.CanWrite {
		t.Fatalf("owner public: got %+v", acc)
	}
}

func TestResolve_viewerPublicProject(t *testing.T) {
	p := &models.ScratchProjectDB{OwnerUserID: "student-1", IsPublic: true}
	acc := Resolve("teacher-9", p)
	if acc.IsOwner || !acc.CanRead || acc.CanWrite {
		t.Fatalf("teacher viewing public: got %+v", acc)
	}
}

func TestResolve_viewerPrivateProject(t *testing.T) {
	p := &models.ScratchProjectDB{OwnerUserID: "student-1", IsPublic: false}
	acc := Resolve("teacher-9", p)
	if acc.CanRead || acc.CanWrite || acc.IsOwner {
		t.Fatalf("teacher viewing private: got %+v", acc)
	}
}

func TestResolve_ownerWithWhitespace(t *testing.T) {
	p := &models.ScratchProjectDB{OwnerUserID: " 42 ", IsPublic: false}
	acc := Resolve("42", p)
	if !acc.IsOwner || !acc.CanWrite {
		t.Fatalf("expected owner with trimmed ids: got %+v", acc)
	}
}

func TestResolve_anyRolePublicReadOnly(t *testing.T) {
	p := &models.ScratchProjectDB{OwnerUserID: "owner", IsPublic: true}
	for _, viewer := range []string{"parent-1", "listener-1", "admin-1"} {
		acc := Resolve(viewer, p)
		if !acc.CanRead || acc.CanWrite || acc.IsOwner {
			t.Fatalf("viewer %s public: got %+v", viewer, acc)
		}
	}
}

func TestSuperAdminReadsClassWork(t *testing.T) {
	cohort, blank := "lk:class-1", " "
	cases := []struct {
		name   string
		role   models.Role
		cohort *string
		want   bool
	}{
		{"superadmin class project", models.SuperAdmin, &cohort, true},
		{"superadmin personal project", models.SuperAdmin, nil, false},
		{"superadmin blank cohort", models.SuperAdmin, &blank, false},
		{"teacher class project", models.Teacher, &cohort, false},
		{"unit admin class project", models.UnitAdmin, &cohort, false},
	}
	for _, tc := range cases {
		p := &models.ScratchProjectDB{OwnerUserID: "student-1", EdxCohortID: tc.cohort}
		if got := SuperAdminReadsClassWork(tc.role, p); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestResolve_classTeacherReadsOnlyHandedIn(t *testing.T) {
	prev := CohortTeacherReader
	CohortTeacherReader = func(teacherID, _, cohortID string) bool { return teacherID == "teacher-1" && cohortID == "7" }
	t.Cleanup(func() { CohortTeacherReader = prev })

	cohort := "7"
	statuses := []struct {
		status *string
		want   bool
	}{
		{nil, false},
		{strPtr(models.ReviewDraft), false},
		{strPtr(models.ReviewSubmitted), true},
		{strPtr(models.ReviewReturned), true},
		{strPtr(models.ReviewAccepted), true},
		{strPtr(models.ReviewShownInClass), true},
	}
	for _, tc := range statuses {
		p := &models.ScratchProjectDB{OwnerUserID: "student-1", EdxCohortID: &cohort, ReviewStatus: tc.status}
		if got := Resolve("teacher-1", p).CanRead; got != tc.want {
			t.Fatalf("status %v: teacher CanRead = %v, want %v", derefStr(tc.status), got, tc.want)
		}
		if !Resolve("student-1", p).CanRead {
			t.Fatalf("status %v: owner lost read", derefStr(tc.status))
		}
	}
}

func strPtr(s string) *string { return &s }

func TestHomeworkLocked(t *testing.T) {
	assignment := "a-1"
	cases := []struct {
		name         string
		assignmentID *string
		status       string
		want         bool
	}{
		{"draft copy", &assignment, models.ReviewDraft, false},
		{"returned copy", &assignment, models.ReviewReturned, false},
		{"submitted copy", &assignment, models.ReviewSubmitted, true},
		{"accepted copy", &assignment, models.ReviewAccepted, true},
		{"shown in class", &assignment, models.ReviewShownInClass, true},
		{"own project, stray status", nil, models.ReviewSubmitted, false},
	}
	for _, c := range cases {
		status := c.status
		p := &models.ScratchProjectDB{OwnerUserID: "student-1", AssignmentID: c.assignmentID, ReviewStatus: &status}
		if got := HomeworkLocked(p); got != c.want {
			t.Errorf("%s: HomeworkLocked=%v, want %v", c.name, got, c.want)
		}
	}
	if HomeworkLocked(&models.ScratchProjectDB{AssignmentID: &assignment}) {
		t.Error("copy without a status is a draft, not locked")
	}
}

func withClassTeacher(t *testing.T, teacherID string) {
	t.Helper()
	saved := CohortTeacherReader
	t.Cleanup(func() { CohortTeacherReader = saved })
	CohortTeacherReader = func(id, _, _ string) bool { return id == teacherID }
}

func TestTeacherCanEdit(t *testing.T) {
	withClassTeacher(t, "teacher-1")
	cohort, assignment := "lk:class-1", "a-1"
	copyWith := func(status string) *models.ScratchProjectDB {
		return &models.ScratchProjectDB{
			OwnerUserID: "student-1", EdxCohortID: &cohort, AssignmentID: &assignment, ReviewStatus: &status,
		}
	}
	for _, status := range []string{models.ReviewSubmitted, models.ReviewAccepted, models.ReviewShownInClass} {
		if !TeacherCanEdit("teacher-1", copyWith(status)) {
			t.Errorf("%s: the class teacher may edit", status)
		}
	}
	// Returned work and a draft are the student's: a teacher's save would race theirs.
	for _, status := range []string{models.ReviewReturned, models.ReviewDraft} {
		if TeacherCanEdit("teacher-1", copyWith(status)) {
			t.Errorf("%s: the class teacher may not edit", status)
		}
	}
	if TeacherCanEdit("teacher-2", copyWith(models.ReviewSubmitted)) {
		t.Error("a teacher of another class edited the work")
	}
	if TeacherCanEdit("", copyWith(models.ReviewSubmitted)) {
		t.Error("a guest edited the work")
	}
	// The class owner handed in a copy of their own: the lock holds for them as for anyone.
	own := copyWith(models.ReviewSubmitted)
	own.OwnerUserID = "teacher-1"
	if TeacherCanEdit("teacher-1", own) {
		t.Error("the teacher edited their own handed-in copy")
	}
	plain := &models.ScratchProjectDB{OwnerUserID: "student-1", IsPublic: true}
	if TeacherCanEdit("teacher-1", plain) {
		t.Error("a teacher edited a project outside any class")
	}
}
