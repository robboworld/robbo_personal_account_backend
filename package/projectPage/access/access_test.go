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
