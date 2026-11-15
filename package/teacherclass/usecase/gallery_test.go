package usecase

import (
	"errors"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"gorm.io/gorm"
)

// The other tests on fileGateway list assignments but have no gallery.
func (g *fileGateway) CountGalleryItems([]string) (map[string]int, error) {
	return map[string]int{}, nil
}

// galleryGateway is fileGateway with lesson galleries and a history log.
type galleryGateway struct {
	*fileGateway
	items   map[string]*models.ClassGalleryItemDB
	seq     int
	history []models.ClassHistoryEventDB
}

func (g *galleryGateway) AddHistoryEvent(e *models.ClassHistoryEventDB) error {
	g.history = append(g.history, *e)
	return nil
}

func (g *galleryGateway) live(it *models.ClassGalleryItemDB) bool {
	_, ok := g.files[it.UserFileID]
	return ok
}

func (g *galleryGateway) ListGalleryItems(assignmentID string) ([]models.ClassGalleryItemDB, error) {
	var out []models.ClassGalleryItemDB
	for _, it := range g.items {
		if it.AssignmentID == assignmentID && g.live(it) {
			out = append(out, *it)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		return out[i].Position < out[j].Position
	})
	return out, nil
}

func (g *galleryGateway) CountGalleryItems(ids []string) (map[string]int, error) {
	out := map[string]int{}
	for _, id := range ids {
		list, _ := g.ListGalleryItems(id)
		if len(list) > 0 {
			out[id] = len(list)
		}
	}
	return out, nil
}

func (g *galleryGateway) GetGalleryItem(id string) (*models.ClassGalleryItemDB, error) {
	if it, ok := g.items[id]; ok && g.live(it) {
		cp := *it
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (g *galleryGateway) AddGalleryItem(item *models.ClassGalleryItemDB, maxPerUploader int) (bool, error) {
	mine, last := 0, 0
	for _, it := range g.items {
		if it.AssignmentID != item.AssignmentID {
			continue
		}
		if it.UserFileID == item.UserFileID {
			*item = *it
			return false, nil
		}
		if it.UploaderUserID == item.UploaderUserID && g.live(it) {
			mine++
		}
		last = max(last, it.Position)
	}
	if maxPerUploader > 0 && mine >= maxPerUploader {
		return false, teacherclass.ErrTooManyGalleryFiles
	}
	g.seq++
	item.ID = "g" + strconv.Itoa(g.seq)
	item.Position = last + 1
	item.CreatedAt = time.Now()
	cp := *item
	g.items[item.ID] = &cp
	return true, nil
}

func (g *galleryGateway) DeleteGalleryItem(id string) error {
	if _, ok := g.items[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	delete(g.items, id)
	return nil
}

func (g *galleryGateway) UpdateGalleryItem(id string, in teacherclass.GalleryItemUpdate) error {
	it, ok := g.items[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if in.Caption != nil {
		it.Caption = *in.Caption
	}
	if in.Pinned != nil {
		it.Pinned = *in.Pinned
	}
	return nil
}

func (g *galleryGateway) ReorderGallery(assignmentID string, ids []string) error {
	for i, id := range ids {
		if it, ok := g.items[id]; ok && it.AssignmentID == assignmentID {
			it.Position = i + 1
		}
	}
	return nil
}

func (g *galleryGateway) SetGalleryClosed(assignmentID string, closed bool) error {
	a := g.assignments[assignmentID]
	a.GalleryClosedAt = nil
	if closed {
		now := time.Now()
		a.GalleryClosedAt = &now
	}
	return nil
}

// newGalleryUseCase: class-1 of teacher "10" with students "797" and "798"; "555" is not in
// it. Assignment "a1" is a Scratch one: any lesson has a gallery.
func newGalleryUseCase(t *testing.T) (*UseCaseImpl, *galleryGateway) {
	t.Helper()
	uc, fgw := newFileUseCase(t)
	fgw.members["798"] = true
	fgw.files["p1"] = models.UserFileDB{ID: "p1", OwnerUserID: "797", OriginalName: "robot.png", Kind: "png"}
	fgw.files["p2"] = models.UserFileDB{ID: "p2", OwnerUserID: "798", OriginalName: "car.png", Kind: "png"}
	fgw.files["t1"] = models.UserFileDB{ID: "t1", OwnerUserID: "10", OriginalName: "board.jpg", Kind: "jpg"}
	gw := &galleryGateway{fileGateway: fgw, items: map[string]*models.ClassGalleryItemDB{}}
	uc.gw = gw
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", teacherclass.CreateAssignmentInput{
		Title: "Урок 1", AssignmentKey: "l1", Kind: models.AssignmentFile,
	}); err != nil {
		t.Fatal(err)
	}
	gw.history = nil
	return uc, gw
}

func itemIDs(g *teacherclass.GalleryDTO) []string {
	var out []string
	for _, it := range g.Items {
		out = append(out, it.ID)
	}
	return out
}

func TestGalleryEveryoneInClassSeesSharedFiles(t *testing.T) {
	uc, gw := newGalleryUseCase(t)
	if _, err := uc.AddGalleryFile("797", models.Student, "a1", "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddGalleryFile("10", models.Teacher, "a1", "t1"); err != nil {
		t.Fatal(err)
	}

	got, err := uc.GetGallery("798", models.Student, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.CanManage || !got.CanUpload {
		t.Fatalf("classmate's gallery = %+v", got)
	}
	if got.Items[0].File.Name != "robot.png" || got.Items[0].UploaderID != "797" || got.Items[0].CanRemove || got.Items[0].Mine {
		t.Fatalf("classmate sees %+v; want someone else's file, not removable", got.Items[0])
	}

	f, rc, err := uc.OpenGalleryFile("798", models.Student, "a1", got.Items[0].ID)
	if err != nil || f.ID != "p1" {
		t.Fatalf("classmate download = %v, %v", f, err)
	}
	rc.Close()

	if _, err := uc.GetGallery("555", models.Student, "a1"); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("outsider: err = %v, want forbidden", err)
	}
	if _, _, err := uc.OpenGalleryFile("555", models.Student, "a1", got.Items[0].ID); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("outsider download: err = %v, want forbidden", err)
	}

	list, err := uc.ListAssignments("798", models.Student, "class-1")
	if err != nil || len(list) != 1 || list[0].GalleryCount != 2 || !list[0].GalleryEnabled {
		t.Fatalf("assignments = %+v, %v; want galleryCount 2", list, err)
	}
	if len(gw.history) != 2 || gw.history[0].Action != models.HistoryGalleryFileAdded ||
		gw.history[0].ActorRole != models.HistoryActorStudent || gw.history[0].Detail != "robot.png" {
		t.Fatalf("history = %+v", gw.history)
	}
}

func TestGalleryOnlyOwnFiles(t *testing.T) {
	uc, _ := newGalleryUseCase(t)
	if _, err := uc.AddGalleryFile("798", models.Student, "a1", "p1"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("someone else's file: err = %v, want not found", err)
	}
	uc.files.(*fakeFiles).infected = map[string]bool{"p1": true}
	if _, err := uc.AddGalleryFile("797", models.Student, "a1", "p1"); err == nil {
		t.Fatal("infected file was shared")
	}
}

func TestGalleryStudentRemovesOnlyOwn(t *testing.T) {
	uc, gw := newGalleryUseCase(t)
	mine, _ := uc.AddGalleryFile("797", models.Student, "a1", "p1")
	theirs, _ := uc.AddGalleryFile("798", models.Student, "a1", "p2")
	theirID := theirs.Items[1].ID

	if _, err := uc.RemoveGalleryItem("797", models.Student, "a1", theirID); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("removing a classmate's file: err = %v, want forbidden", err)
	}
	got, err := uc.RemoveGalleryItem("797", models.Student, "a1", mine.Items[0].ID)
	if err != nil || len(got.Items) != 1 {
		t.Fatalf("removing own = %+v, %v", got, err)
	}
	if _, ok := gw.files["p1"]; !ok {
		t.Fatal("the file left the student's Мои файлы")
	}

	// The teacher removes anyone's; the history names whose it was.
	if _, err := uc.RemoveGalleryItem("10", models.Teacher, "a1", theirID); err != nil {
		t.Fatal(err)
	}
	last := gw.history[len(gw.history)-1]
	if last.Action != models.HistoryGalleryFileRemoved || last.TargetUserID != "798" || last.ActorRole != models.HistoryActorOwner {
		t.Fatalf("history = %+v", last)
	}
}

func TestGalleryTeacherEdits(t *testing.T) {
	uc, _ := newGalleryUseCase(t)
	uc.AddGalleryFile("797", models.Student, "a1", "p1")
	g, _ := uc.AddGalleryFile("798", models.Student, "a1", "p2")
	first, second := g.Items[0].ID, g.Items[1].ID

	caption, pinned := "  Лучший робот  ", true
	if _, err := uc.UpdateGalleryItem("797", models.Student, "a1", first, teacherclass.GalleryItemUpdate{Caption: &caption}); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("student caption: err = %v, want forbidden", err)
	}
	g, err := uc.UpdateGalleryItem("10", models.Teacher, "a1", first, teacherclass.GalleryItemUpdate{Caption: &caption})
	if err != nil || g.Items[0].Caption != "Лучший робот" {
		t.Fatalf("caption = %+v, %v", g, err)
	}
	long := string(make([]rune, maxGalleryCaptionRunes+1))
	if _, err := uc.UpdateGalleryItem("10", models.Teacher, "a1", first, teacherclass.GalleryItemUpdate{Caption: &long}); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("long caption: err = %v", err)
	}

	g, err = uc.ReorderGallery("10", models.Teacher, "a1", []string{second, first})
	if err != nil || itemIDs(g)[0] != second {
		t.Fatalf("reorder = %v, %v", itemIDs(g), err)
	}
	g, err = uc.UpdateGalleryItem("10", models.Teacher, "a1", first, teacherclass.GalleryItemUpdate{Pinned: &pinned})
	if err != nil || itemIDs(g)[0] != first || !g.Items[0].Pinned {
		t.Fatalf("pinned first = %v, %v", itemIDs(g), err)
	}
	if _, err := uc.ReorderGallery("10", models.Teacher, "a1", []string{first, first}); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("duplicate ids: err = %v", err)
	}
}

func TestGalleryClosedToStudents(t *testing.T) {
	uc, gw := newGalleryUseCase(t)
	if _, err := uc.SetGalleryClosed("797", models.Student, "a1", true); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("student closes: err = %v", err)
	}
	g, err := uc.SetGalleryClosed("10", models.Teacher, "a1", true)
	if err != nil || !g.Closed {
		t.Fatalf("close = %+v, %v", g, err)
	}
	if _, err := uc.AddGalleryFile("797", models.Student, "a1", "p1"); !errors.Is(err, teacherclass.ErrGalleryClosed) {
		t.Fatalf("student adds to closed: err = %v", err)
	}
	if g, err := uc.GetGallery("797", models.Student, "a1"); err != nil || g.CanUpload {
		t.Fatalf("closed gallery for a student = %+v, %v", g, err)
	}
	if _, err := uc.AddGalleryFile("10", models.Teacher, "a1", "t1"); err != nil {
		t.Fatalf("teacher adds to closed: %v", err)
	}
	if _, err := uc.SetGalleryClosed("10", models.Teacher, "a1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AddGalleryFile("797", models.Student, "a1", "p1"); err != nil {
		t.Fatalf("after reopen: %v", err)
	}

	now := time.Now()
	gw.inv.ArchivedAt = &now
	if _, err := uc.AddGalleryFile("10", models.Teacher, "a1", "t1"); !errors.Is(err, teacherclass.ErrGalleryClosed) {
		t.Fatalf("archived class: err = %v", err)
	}
}

func TestGalleryLimitPerStudent(t *testing.T) {
	uc, gw := newGalleryUseCase(t)
	for i := 0; i < defaultMaxGalleryFilesPerUser+1; i++ {
		id := "x" + strconv.Itoa(i)
		gw.files[id] = models.UserFileDB{ID: id, OwnerUserID: "797", OriginalName: id + ".png", Kind: "png"}
		_, err := uc.AddGalleryFile("797", models.Student, "a1", id)
		if i < defaultMaxGalleryFilesPerUser && err != nil {
			t.Fatalf("file %d: %v", i, err)
		}
		if i == defaultMaxGalleryFilesPerUser && !errors.Is(err, teacherclass.ErrTooManyGalleryFiles) {
			t.Fatalf("over the limit: err = %v", err)
		}
	}
	// The same file twice is one item and one history event.
	events := len(gw.history)
	g, err := uc.AddGalleryFile("797", models.Student, "a1", "x0")
	if err != nil || len(g.Items) != defaultMaxGalleryFilesPerUser {
		t.Fatalf("re-adding = %d items, %v", len(g.Items), err)
	}
	if len(gw.history) != events {
		t.Fatalf("re-adding wrote history: %+v", gw.history[events:])
	}
}

func TestGalleryNeedsFileExchange(t *testing.T) {
	uc, _ := newGalleryUseCase(t)
	uc.files.(*fakeFiles).exchange = false
	if _, err := uc.GetGallery("797", models.Student, "a1"); !errors.Is(err, teacherclass.ErrFileAssignmentsDisabled) {
		t.Fatalf("err = %v, want disabled", err)
	}
}

func TestGalleryItemOfAnotherAssignment(t *testing.T) {
	uc, _ := newGalleryUseCase(t)
	g, _ := uc.AddGalleryFile("797", models.Student, "a1", "p1")
	if _, err := uc.CreateAssignment("10", models.Teacher, "class-1", teacherclass.CreateAssignmentInput{
		Title: "Урок 2", AssignmentKey: "l2", Kind: models.AssignmentFile,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := uc.OpenGalleryFile("798", models.Student, "a2", g.Items[0].ID); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("item through another lesson: err = %v, want not found", err)
	}
}
