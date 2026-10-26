package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

type allClassesGateway struct {
	studentGateway
}

func (g *allClassesGateway) ListInvitesByOwner(owner string) ([]models.ClassInviteDB, error) {
	return []models.ClassInviteDB{{ID: "mine", OwnerTeacherID: owner}}, nil
}

func (g *allClassesGateway) ListAllInvites() ([]models.ClassInviteDB, error) {
	return []models.ClassInviteDB{
		{ID: "mine", OwnerTeacherID: "1"},
		{ID: "5A", OwnerTeacherID: "10"},
		{ID: "6B", OwnerTeacherID: "10"},
	}, nil
}

func TestOnlyOwnerOrSuperAdminOpensClass(t *testing.T) {
	uc := &UseCaseImpl{gw: &studentGateway{}} // class-1 belongs to teacher 10

	if _, err := uc.GetClass("11", models.Teacher, "class-1"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("other teacher err = %v, want ErrForbidden", err)
	}
	if _, err := uc.GetClass("10", models.Teacher, "class-1"); err != nil {
		t.Fatalf("owner err = %v", err)
	}
	if _, err := uc.GetClass("1", models.SuperAdmin, "class-1"); err != nil {
		t.Fatalf("super admin err = %v", err)
	}
}

func TestSuperAdminListsEveryClass(t *testing.T) {
	uc := &UseCaseImpl{gw: &allClassesGateway{}}

	list, err := uc.ListMyClasses("1", models.SuperAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("super admin classes = %+v, want all three", list)
	}
	list, err = uc.ListMyClasses("11", models.Teacher)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].OwnerTeacherID != "11" {
		t.Fatalf("teacher classes = %+v, want only own", list)
	}
}

func TestClassListLooksUpEachOwnerOnce(t *testing.T) {
	prev := ownerDisplayName
	lookups := map[string]int{}
	ownerDisplayName = func(id string) string {
		lookups[id]++
		return "Teacher " + id
	}
	t.Cleanup(func() { ownerDisplayName = prev })
	uc := &UseCaseImpl{gw: &allClassesGateway{}}

	list, err := uc.ListMyClasses("1", models.SuperAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if lookups["1"] != 1 || lookups["10"] != 1 {
		t.Fatalf("lookups = %v, want one per owner", lookups)
	}
	if list[2].OwnerName != "Teacher 10" {
		t.Fatalf("6B owner = %q", list[2].OwnerName)
	}
}
