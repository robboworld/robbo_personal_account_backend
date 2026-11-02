package usecase

import (
	"errors"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/adminaudit"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	ppageusecase "github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage/usecase"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/storage"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/spf13/viper"
)

const (
	maxProjectsListed   = 1000
	defaultAuditLimit   = 50
	maxAuditLimit       = 200
	defaultTopUsers     = 20
	defaultAlertBelowMB = 10240
)

// UserName is how the ЛК shows an LMS account.
type UserName struct {
	Username string
	FullName string
}

// NameLookup resolves LMS user ids to names in one call; unknown ids are absent.
type NameLookup func(ids []string) map[string]UserName

// LMSNameLookup reads names from LMS MySQL; on error it returns what it has (none).
func LMSNameLookup(ids []string) map[string]UserName {
	out := map[string]UserName{}
	var nums []int64
	for _, id := range ids {
		if n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64); err == nil && n > 0 {
			nums = append(nums, n)
		}
	}
	if len(nums) == 0 {
		return out
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		log.Printf("admincontent: LMS reader: %v", err)
		return out
	}
	defer reader.Close()
	hits, err := reader.LookupAuthUsersByIDs(nums)
	if err != nil {
		log.Printf("admincontent: LMS names: %v", err)
		return out
	}
	for id, h := range hits {
		out[strconv.FormatInt(id, 10)] = UserName{Username: h.Username, FullName: strings.TrimSpace(h.FullName)}
	}
	return out
}

type Deps struct {
	Gateway       admincontent.Gateway
	Audit         adminaudit.Gateway
	Projects      projectPage.Gateway
	ProjectsUC    projectPage.UseCase
	Files         userfiles.UseCase
	Quota         *storage.Quota
	Licensing     licensing.Gateway
	Notifications notifications.Gateway
	// Volume is nil while user files are off.
	Volume admincontent.Volume
	Names  NameLookup
	// UploadsStopBelowBytes is userFiles.minFreeDiskMB in bytes.
	UploadsStopBelowBytes int64
}

type AdminContentUseCase struct {
	d Deps
}

func NewAdminContentUseCase(d Deps) *AdminContentUseCase {
	if d.Names == nil {
		d.Names = LMSNameLookup
	}
	return &AdminContentUseCase{d: d}
}

func (u *AdminContentUseCase) filesEnabled() bool {
	return u.d.Files != nil && u.d.Files.Enabled()
}

// UserContent is everything a user stores, private projects included.
type UserContent struct {
	Usage        storage.Usage
	QuotaBytes   int64
	FilesEnabled bool
	Projects     []admincontent.ProjectItem
	Files        []*models.UserFileDB
}

func (u *AdminContentUseCase) UserContent(target string) (*UserContent, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, admincontent.ErrNotFound
	}
	out := &UserContent{FilesEnabled: u.filesEnabled()}
	var err error
	if out.Projects, err = u.d.Gateway.ListProjectsForOwner(target, maxProjectsListed); err != nil {
		return nil, err
	}
	if out.FilesEnabled {
		if out.Files, err = u.d.Files.List(target); err != nil {
			return nil, err
		}
	}
	if u.d.Quota != nil {
		if out.Usage, err = u.d.Quota.Usage(target); err != nil {
			return nil, err
		}
		if out.QuotaBytes, err = u.d.Quota.LimitBytes(target); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// AuditItem is an audit record with the moderator's name.
type AuditItem struct {
	models.AdminContentAuditDB
	ActorName string
}

func (u *AdminContentUseCase) Audit(target string, limit int) ([]AuditItem, error) {
	if limit <= 0 {
		limit = defaultAuditLimit
	}
	limit = min(limit, maxAuditLimit)
	rows, err := u.d.Audit.ListForTarget(strings.TrimSpace(target), limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ActorUserID)
	}
	names := u.d.Names(ids)
	out := make([]AuditItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditItem{AdminContentAuditDB: r, ActorName: displayName(names[r.ActorUserID], r.ActorUserID)})
	}
	return out, nil
}

func displayName(n UserName, id string) string {
	if n.FullName != "" {
		return n.FullName
	}
	if n.Username != "" {
		return n.Username
	}
	return "User " + id
}

// ownedProject loads the project and checks it belongs to target: a moderator acts on a
// user's content, so a project id from another account is "not found".
func (u *AdminContentUseCase) ownedProject(target, projectID string) (*models.ScratchProjectDB, error) {
	row, err := u.d.Projects.GetScratchProjectById(projectID)
	if err != nil || row == nil || row.DeletedAt != nil || strings.TrimSpace(row.OwnerUserID) != strings.TrimSpace(target) {
		return nil, admincontent.ErrNotFound
	}
	return row, nil
}

func projectTitle(row *models.ScratchProjectDB) string {
	if t := strings.TrimSpace(row.Title); t != "" {
		return t
	}
	return "Untitled"
}

// DownloadProject returns the current .sb3 of someone's project after the audit record is
// written (no record, no download).
func (u *AdminContentUseCase) DownloadProject(actor, target, projectID, reason string) ([]byte, string, error) {
	reason, err := adminaudit.NormalizeReason(reason)
	if err != nil {
		return nil, "", err
	}
	row, err := u.ownedProject(target, projectID)
	if err != nil {
		return nil, "", err
	}
	data, err := u.d.Projects.GetLatestSb3Archive(row.ID)
	if err != nil {
		if errors.Is(err, projectPage.ErrSb3ArchiveNotFound) {
			return nil, "", admincontent.ErrNotFound
		}
		return nil, "", err
	}
	if err := u.d.Audit.Write(&models.AdminContentAuditDB{
		ActorUserID: actor, TargetUserID: row.OwnerUserID, Action: models.AuditProjectDownload,
		ObjectType: "project", ObjectID: row.ID, ObjectName: projectTitle(row), SizeBytes: int64(len(data)), Reason: reason,
	}); err != nil {
		return nil, "", err
	}
	return data, ppageusecase.SB3DownloadFilename(row.Title, row.ID), nil
}

// DeleteProject deletes someone's project; ModerateDeleteProjectPage writes the audit record
// and tells the owner.
func (u *AdminContentUseCase) DeleteProject(actor, target, projectID, reason string) error {
	reason, err := adminaudit.NormalizeReason(reason)
	if err != nil {
		return err
	}
	row, err := u.ownedProject(target, projectID)
	if err != nil {
		return err
	}
	return u.d.ProjectsUC.ModerateDeleteProjectPage(row.ID, actor, reason)
}

func (u *AdminContentUseCase) openFile(target, fileID string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	if !u.filesEnabled() {
		return nil, nil, admincontent.ErrNotFound
	}
	row, rc, err := u.d.Files.Open(strings.TrimSpace(target), fileID)
	if errors.Is(err, userfiles.ErrFileNotFound) || errors.Is(err, userfiles.ErrDisabled) {
		return nil, nil, admincontent.ErrNotFound
	}
	return row, rc, err
}

// DownloadFile opens someone's file after the audit record is written.
func (u *AdminContentUseCase) DownloadFile(actor, target, fileID, reason string) (*models.UserFileDB, io.ReadSeekCloser, error) {
	reason, err := adminaudit.NormalizeReason(reason)
	if err != nil {
		return nil, nil, err
	}
	row, rc, err := u.openFile(target, fileID)
	if err != nil {
		return nil, nil, err
	}
	if err := u.d.Audit.Write(&models.AdminContentAuditDB{
		ActorUserID: actor, TargetUserID: row.OwnerUserID, Action: models.AuditFileDownload,
		ObjectType: "file", ObjectID: row.ID, ObjectName: row.OriginalName, SizeBytes: row.SizeBytes, Reason: reason,
	}); err != nil {
		rc.Close()
		return nil, nil, err
	}
	return row, rc, nil
}

// DeleteFile deletes someone's file, records it and tells the owner.
func (u *AdminContentUseCase) DeleteFile(actor, target, fileID, reason string) error {
	reason, err := adminaudit.NormalizeReason(reason)
	if err != nil {
		return err
	}
	row, rc, err := u.openFile(target, fileID)
	if err != nil {
		return err
	}
	rc.Close()
	if err := u.d.Audit.Write(&models.AdminContentAuditDB{
		ActorUserID: actor, TargetUserID: row.OwnerUserID, Action: models.AuditFileDelete,
		ObjectType: "file", ObjectID: row.ID, ObjectName: row.OriginalName, SizeBytes: row.SizeBytes, Reason: reason,
	}); err != nil {
		return err
	}
	if err := u.d.Files.Delete(row.OwnerUserID, row.ID); err != nil {
		if errors.Is(err, userfiles.ErrFileNotFound) {
			return admincontent.ErrNotFound
		}
		return err
	}
	if u.d.Notifications != nil {
		actionURL := "/myfiles"
		dedupeKey := "file_deleted:" + row.ID
		_ = u.d.Notifications.CreateOrUpdateByDedupe(&models.UserNotificationDB{
			RecipientUserID: row.OwnerUserID,
			Title:           "Файл удалён модератором",
			Body:            "Ваш файл «" + row.OriginalName + "» удалён модератором.\nПричина: " + reason,
			Kind:            "file_deleted",
			Severity:        "WARNING",
			Source:          "system",
			ActionURL:       &actionURL,
			DedupeKey:       &dedupeKey,
		})
	}
	return nil
}

// VolumeStats is the disk under the user files store.
type VolumeStats struct {
	TotalBytes            int64 `json:"totalBytes"`
	FreeBytes             int64 `json:"freeBytes"`
	UploadsStopBelowBytes int64 `json:"uploadsStopBelowBytes"`
	AlertBelowBytes       int64 `json:"alertBelowBytes"`
}

type Databases struct {
	ProjectsBytes  int64 `json:"projectsBytes"`
	LicensingBytes int64 `json:"licensingBytes"`
}

type TopOwner struct {
	UserID        string `json:"userId"`
	Username      string `json:"username"`
	FullName      string `json:"fullName"`
	ProjectsBytes int64  `json:"projectsBytes"`
	FilesBytes    int64  `json:"filesBytes"`
	TotalBytes    int64  `json:"totalBytes"`
	// QuotaBytes 0 means unlimited.
	QuotaBytes int64 `json:"quotaBytes"`
}

type StorageReport struct {
	Volume    *VolumeStats        `json:"volume"`
	Totals    admincontent.Totals `json:"totals"`
	Databases Databases           `json:"databases"`
	Top       []TopOwner          `json:"top"`
}

// AlertBelowBytes is adminStorage.alertFreeBelowMB in bytes.
func AlertBelowBytes() int64 {
	mb := viper.GetInt64("adminStorage.alertFreeBelowMB")
	if mb <= 0 {
		mb = defaultAlertBelowMB
	}
	return mb << 20
}

func (u *AdminContentUseCase) volumeStats() (*VolumeStats, error) {
	if u.d.Volume == nil {
		return nil, nil
	}
	total, err := u.d.Volume.TotalBytes()
	if err != nil {
		return nil, err
	}
	free, err := u.d.Volume.FreeBytes()
	if err != nil {
		return nil, err
	}
	if total < 0 || free < 0 {
		return nil, nil
	}
	return &VolumeStats{
		TotalBytes:            total,
		FreeBytes:             free,
		UploadsStopBelowBytes: u.d.UploadsStopBelowBytes,
		AlertBelowBytes:       AlertBelowBytes(),
	}, nil
}

func (u *AdminContentUseCase) Storage() (*StorageReport, error) {
	r := &StorageReport{Top: []TopOwner{}}
	var err error
	if r.Volume, err = u.volumeStats(); err != nil {
		return nil, err
	}
	if r.Totals, err = u.d.Gateway.StorageTotals(); err != nil {
		return nil, err
	}
	if r.Databases.ProjectsBytes, err = u.d.Gateway.ProjectsDatabaseBytes(); err != nil {
		return nil, err
	}
	if r.Databases.LicensingBytes, err = u.d.Gateway.LicensingDatabaseBytes(); err != nil {
		return nil, err
	}
	limit := viper.GetInt("adminStorage.topUsers")
	if limit <= 0 {
		limit = defaultTopUsers
	}
	owners, err := u.d.Gateway.TopOwners(limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(owners))
	for _, o := range owners {
		ids = append(ids, o.OwnerUserID)
	}
	names := u.d.Names(ids)
	for _, o := range owners {
		t := TopOwner{
			UserID:        o.OwnerUserID,
			Username:      names[o.OwnerUserID].Username,
			FullName:      names[o.OwnerUserID].FullName,
			ProjectsBytes: o.ProjectsBytes,
			FilesBytes:    o.FilesBytes,
			TotalBytes:    o.ProjectsBytes + o.FilesBytes,
		}
		if u.d.Quota != nil {
			t.QuotaBytes, _ = u.d.Quota.LimitBytes(o.OwnerUserID)
		}
		r.Top = append(r.Top, t)
	}
	sort.SliceStable(r.Top, func(i, j int) bool { return r.Top[i].TotalBytes > r.Top[j].TotalBytes })
	return r, nil
}
