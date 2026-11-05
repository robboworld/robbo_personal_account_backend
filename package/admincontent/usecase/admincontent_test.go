package usecase

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/adminaudit"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
)

type fakeAudit struct {
	rows []models.AdminContentAuditDB
	fail bool
}

func (a *fakeAudit) Write(e *models.AdminContentAuditDB) error {
	if a.fail {
		return errors.New("db down")
	}
	a.rows = append(a.rows, *e)
	return nil
}

func (a *fakeAudit) ListForTarget(target string, limit int) ([]models.AdminContentAuditDB, error) {
	var out []models.AdminContentAuditDB
	for _, r := range a.rows {
		if r.TargetUserID == target {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakePages struct {
	projectPage.Gateway
	rows map[string]*models.ScratchProjectDB
}

func (p *fakePages) GetScratchProjectById(id string) (*models.ScratchProjectDB, error) {
	if r, ok := p.rows[id]; ok {
		return r, nil
	}
	return nil, projectPage.ErrPageNotFound
}

func (p *fakePages) GetLatestSb3Archive(id string) ([]byte, error) { return []byte("PK sb3"), nil }

type fakePagesUC struct {
	projectPage.UseCase
	deleted []string
}

func (u *fakePagesUC) ModerateDeleteProjectPage(id, moderator, reason string) error {
	u.deleted = append(u.deleted, id+"|"+moderator+"|"+reason)
	return nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

type fakeFiles struct {
	userfiles.UseCase
	enabled bool
	rows    map[string]*models.UserFileDB
	deleted []string
}

func (f *fakeFiles) Enabled() bool { return f.enabled }
func (f *fakeFiles) List(owner string) ([]*models.UserFileDB, error) {
	var out []*models.UserFileDB
	for _, r := range f.rows {
		if r.OwnerUserID == owner {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeFiles) Open(owner, id string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	r, ok := f.rows[id]
	if !ok || r.OwnerUserID != owner {
		return nil, nil, userfiles.ErrFileNotFound
	}
	return r, nopCloser{bytes.NewReader([]byte("data"))}, nil
}
func (f *fakeFiles) ModerateDelete(owner, id string) error {
	f.deleted = append(f.deleted, owner+"|"+id)
	delete(f.rows, id)
	return nil
}

type fakeNotif struct {
	notifications.Gateway
	sent []models.UserNotificationDB
}

func (n *fakeNotif) CreateOrUpdateByDedupe(m *models.UserNotificationDB) error {
	n.sent = append(n.sent, *m)
	return nil
}

type fakeContentGW struct {
	admincontent.Gateway
	top []admincontent.OwnerUsage
}

func (g *fakeContentGW) ListProjectsForOwner(owner string, limit int) ([]admincontent.ProjectItem, error) {
	return []admincontent.ProjectItem{{ID: "p1", Title: "Кот", SizeBytes: 10}}, nil
}
func (g *fakeContentGW) StorageTotals() (admincontent.Totals, error) {
	return admincontent.Totals{ProjectsBytes: 30, FilesBytes: 12, OwnersWithData: 2}, nil
}
func (g *fakeContentGW) TopOwners(int) ([]admincontent.OwnerUsage, error) { return g.top, nil }
func (g *fakeContentGW) ProjectsDatabaseBytes() (int64, error)            { return 1 << 20, nil }
func (g *fakeContentGW) LicensingDatabaseBytes() (int64, error)           { return 2 << 20, nil }

type fixedVolume struct{ total, free int64 }

func (v fixedVolume) TotalBytes() (int64, error) { return v.total, nil }
func (v fixedVolume) FreeBytes() (int64, error)  { return v.free, nil }

type env struct {
	uc     *AdminContentUseCase
	audit  *fakeAudit
	pages  *fakePagesUC
	files  *fakeFiles
	notify *fakeNotif
}

func newEnv() env {
	e := env{
		audit:  &fakeAudit{},
		pages:  &fakePagesUC{},
		notify: &fakeNotif{},
		files: &fakeFiles{enabled: true, rows: map[string]*models.UserFileDB{
			"f1": {ID: "f1", OwnerUserID: "7", OriginalName: "doc.pdf", SizeBytes: 4},
		}},
	}
	e.uc = NewAdminContentUseCase(Deps{
		Gateway: &fakeContentGW{top: []admincontent.OwnerUsage{
			{OwnerUserID: "7", ProjectsBytes: 10, FilesBytes: 4},
			{OwnerUserID: "8", ProjectsBytes: 20, FilesBytes: 8},
		}},
		Audit: e.audit,
		Projects: &fakePages{rows: map[string]*models.ScratchProjectDB{
			"p1": {ID: "p1", OwnerUserID: "7", Title: "Кот"},
			"p2": {ID: "p2", OwnerUserID: "8", Title: "Чужой"},
		}},
		ProjectsUC:    e.pages,
		Files:         e.files,
		Notifications: e.notify,
		Volume:        fixedVolume{total: 100 << 30, free: 40 << 30},
		Names: func(ids []string) map[string]UserName {
			return map[string]UserName{"1": {Username: "admin", FullName: "Админ"}, "8": {Username: "kid8"}}
		},
		UploadsStopBelowBytes: 1 << 30,
	})
	return e
}

func TestReasonIsRequiredEverywhere(t *testing.T) {
	e := newEnv()
	checks := map[string]error{}
	_, _, checks["project download"] = e.uc.DownloadProject("1", "7", "p1", " ")
	_, _, checks["file download"] = e.uc.DownloadFile("1", "7", "f1", "ok")
	checks["project delete"] = e.uc.DeleteProject("1", "7", "p1", "")
	checks["file delete"] = e.uc.DeleteFile("1", "7", "f1", "аб")
	for what, err := range checks {
		if !errors.Is(err, adminaudit.ErrReasonRequired) {
			t.Errorf("%s: err=%v, want REASON_REQUIRED", what, err)
		}
	}
	if len(e.audit.rows) != 0 || len(e.pages.deleted) != 0 || len(e.files.deleted) != 0 {
		t.Fatal("something happened without a reason")
	}
}

func TestDownloadsAreAudited(t *testing.T) {
	e := newEnv()
	data, name, err := e.uc.DownloadProject("1", "7", "p1", "жалоба на проект")
	if err != nil || len(data) == 0 || name == "" {
		t.Fatalf("data=%d name=%q err=%v", len(data), name, err)
	}
	row, rc, err := e.uc.DownloadFile("1", "7", "f1", "жалоба на файл")
	if err != nil || row.ID != "f1" {
		t.Fatalf("file: %v", err)
	}
	rc.Close()
	if len(e.audit.rows) != 2 || e.audit.rows[0].Action != models.AuditProjectDownload ||
		e.audit.rows[1].Action != models.AuditFileDownload || e.audit.rows[1].ActorUserID != "1" ||
		e.audit.rows[1].TargetUserID != "7" || e.audit.rows[1].Reason != "жалоба на файл" {
		t.Fatalf("audit=%+v", e.audit.rows)
	}
}

func TestNoAuditNoDownload(t *testing.T) {
	e := newEnv()
	e.audit.fail = true
	if _, _, err := e.uc.DownloadProject("1", "7", "p1", "причина"); err == nil {
		t.Fatal("project served without an audit record")
	}
	if _, _, err := e.uc.DownloadFile("1", "7", "f1", "причина"); err == nil {
		t.Fatal("file served without an audit record")
	}
	if err := e.uc.DeleteFile("1", "7", "f1", "причина"); err == nil || len(e.files.deleted) != 0 {
		t.Fatal("file deleted without an audit record")
	}
}

func TestOtherUsersObjectsAreNotFound(t *testing.T) {
	e := newEnv()
	if _, _, err := e.uc.DownloadProject("1", "7", "p2", "причина"); !errors.Is(err, admincontent.ErrNotFound) {
		t.Fatalf("project of user 8 via user 7: %v", err)
	}
	if err := e.uc.DeleteProject("1", "7", "p2", "причина"); !errors.Is(err, admincontent.ErrNotFound) {
		t.Fatalf("delete project of user 8 via user 7: %v", err)
	}
	if err := e.uc.DeleteFile("1", "8", "f1", "причина"); !errors.Is(err, admincontent.ErrNotFound) {
		t.Fatalf("file of user 7 via user 8: %v", err)
	}
	if len(e.audit.rows) != 0 {
		t.Fatal("audit written for a refused action")
	}
}

func TestDeleteFileNotifiesOwner(t *testing.T) {
	e := newEnv()
	if err := e.uc.DeleteFile("1", "7", "f1", "нарушение правил"); err != nil {
		t.Fatal(err)
	}
	if len(e.files.deleted) != 1 || e.files.deleted[0] != "7|f1" {
		t.Fatalf("deleted=%v", e.files.deleted)
	}
	if len(e.audit.rows) != 1 || e.audit.rows[0].Action != models.AuditFileDelete {
		t.Fatalf("audit=%+v", e.audit.rows)
	}
	if len(e.notify.sent) != 1 || e.notify.sent[0].RecipientUserID != "7" || e.notify.sent[0].Kind != "file_deleted" {
		t.Fatalf("notifications=%+v", e.notify.sent)
	}
}

func TestDeleteProjectGoesThroughModeration(t *testing.T) {
	e := newEnv()
	if err := e.uc.DeleteProject("1", "7", "p1", " спам "); err != nil {
		t.Fatal(err)
	}
	if len(e.pages.deleted) != 1 || e.pages.deleted[0] != "p1|1|спам" {
		t.Fatalf("deleted=%v", e.pages.deleted)
	}
}

func TestFilesOffMeansNotFound(t *testing.T) {
	e := newEnv()
	e.files.enabled = false
	if _, _, err := e.uc.DownloadFile("1", "7", "f1", "причина"); !errors.Is(err, admincontent.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	c, err := e.uc.UserContent("7")
	if err != nil || c.FilesEnabled || len(c.Files) != 0 || len(c.Projects) != 1 {
		t.Fatalf("content=%+v err=%v", c, err)
	}
}

func TestAuditListHasActorNames(t *testing.T) {
	e := newEnv()
	e.uc.DeleteFile("1", "7", "f1", "нарушение")
	items, err := e.uc.Audit("7", 0)
	if err != nil || len(items) != 1 || items[0].ActorName != "Админ" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestStorageReport(t *testing.T) {
	e := newEnv()
	r, err := e.uc.Storage()
	if err != nil {
		t.Fatal(err)
	}
	if r.Volume == nil || r.Volume.FreeBytes != 40<<30 || r.Volume.UploadsStopBelowBytes != 1<<30 || r.Volume.AlertBelowBytes != AlertBelowBytes() {
		t.Fatalf("volume=%+v", r.Volume)
	}
	if len(r.Top) != 2 || r.Top[0].UserID != "8" || r.Top[0].TotalBytes != 28 || r.Top[0].Username != "kid8" {
		t.Fatalf("top=%+v", r.Top)
	}
	if r.Databases.LicensingBytes != 2<<20 || r.Totals.OwnersWithData != 2 {
		t.Fatalf("report=%+v", r)
	}
}

func TestStorageAlertSchedule(t *testing.T) {
	a := storageAlert{below: 10, repeat: 24 * time.Hour}
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	if a.due(20, t0) {
		t.Fatal("alert with enough space")
	}
	if !a.due(5, t0) {
		t.Fatal("no alert on dropping below")
	}
	a.sent(t0)
	if a.due(5, t0.Add(time.Hour)) {
		t.Fatal("repeated within a day")
	}
	if !a.due(5, t0.Add(25*time.Hour)) {
		t.Fatal("no repeat after a day of low space")
	}
	a.sent(t0.Add(25 * time.Hour))
	if a.due(20, t0.Add(26*time.Hour)) {
		t.Fatal("alert after space came back")
	}
	if !a.due(5, t0.Add(27*time.Hour)) {
		t.Fatal("no alert on dropping below again")
	}
}

func TestStorageAlerterNotifiesEverySuperAdmin(t *testing.T) {
	n := &fakeNotif{}
	a := NewStorageAlerter(fixedVolume{total: 100 << 30, free: 1 << 30}, n,
		func() ([]string, error) { return []string{"1", "2"}, nil }, 1<<30)
	now := time.Now()
	if sent, err := a.Check(now); err != nil || sent != 2 {
		t.Fatalf("sent=%d err=%v", sent, err)
	}
	if sent, _ := a.Check(now.Add(time.Minute)); sent != 0 {
		t.Fatal("alerted again a minute later")
	}
	if n.sent[0].Kind != "storage_low" || *n.sent[0].ActionURL != "/admin/storage" {
		t.Fatalf("notification=%+v", n.sent[0])
	}
}
