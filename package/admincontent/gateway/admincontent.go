package gateway

import (
	"errors"
	"fmt"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type AdminContentGatewayImpl struct {
	projects  *gorm.DB
	licensing *gorm.DB
}

func SetupAdminContentGateway() (*AdminContentGatewayImpl, error) {
	projectsDSN := viper.GetString("projectsPostgres.postgresDsn")
	if projectsDSN == "" {
		return nil, errors.New("projectsPostgres.postgresDsn (or env PROJECTS_POSTGRES_DSN) is required")
	}
	projects, err := db_client.OpenByDSN(projectsDSN)
	if err != nil {
		return nil, fmt.Errorf("SetupAdminContentGateway: %w", err)
	}
	g := &AdminContentGatewayImpl{projects: projects}
	if dsn := viper.GetString("licensingPostgres.postgresDsn"); dsn != "" {
		if g.licensing, err = db_client.OpenByDSN(dsn); err != nil {
			return nil, fmt.Errorf("SetupAdminContentGateway: %w", err)
		}
	}
	return g, nil
}

func NewAdminContentGateway(projects, licensing *gorm.DB) *AdminContentGatewayImpl {
	return &AdminContentGatewayImpl{projects: projects, licensing: licensing}
}

func (g *AdminContentGatewayImpl) ListProjectsForOwner(ownerUserID string, limit int) ([]admincontent.ProjectItem, error) {
	var rows []admincontent.ProjectItem
	err := g.projects.Raw(`
		SELECT p.id, p.title, p.is_public, COALESCE(v.size_bytes, 0) AS size_bytes, p.updated_at
		FROM scratch_projects p
		LEFT JOIN scratch_project_versions v ON v.id = p.current_version_id
		WHERE p.owner_user_id = ? AND p.deleted_at IS NULL
		ORDER BY p.updated_at DESC
		LIMIT ?`, ownerUserID, limit).Scan(&rows).Error
	return rows, err
}

// ownersCTE sums each owner's current project versions and files, as the quota counts them.
const ownersCTE = `
	WITH p AS (
		SELECT p.owner_user_id AS owner, COUNT(*) AS n, COALESCE(SUM(v.size_bytes), 0) AS b
		FROM scratch_projects p
		LEFT JOIN scratch_project_versions v ON v.id = p.current_version_id
		WHERE p.deleted_at IS NULL
		GROUP BY p.owner_user_id
	), f AS (
		SELECT owner_user_id AS owner, COUNT(*) AS n, COALESCE(SUM(size_bytes), 0) AS b
		FROM user_files
		WHERE deleted_at IS NULL
		GROUP BY owner_user_id
	), o AS (
		SELECT COALESCE(p.owner, f.owner) AS owner,
			COALESCE(p.b, 0) AS projects_bytes, COALESCE(p.n, 0) AS project_count,
			COALESCE(f.b, 0) AS files_bytes, COALESCE(f.n, 0) AS file_count
		FROM p FULL OUTER JOIN f ON p.owner = f.owner
	)`

func (g *AdminContentGatewayImpl) StorageTotals() (admincontent.Totals, error) {
	var t admincontent.Totals
	err := g.projects.Raw(ownersCTE + `
		SELECT COALESCE(SUM(projects_bytes), 0) AS projects_bytes,
			COALESCE(SUM(project_count), 0) AS project_count,
			COALESCE(SUM(files_bytes), 0) AS files_bytes,
			COALESCE(SUM(file_count), 0) AS file_count,
			COUNT(*) FILTER (WHERE projects_bytes + files_bytes > 0) AS owners_with_data
		FROM o`).Scan(&t).Error
	return t, err
}

func (g *AdminContentGatewayImpl) TopOwners(limit int) ([]admincontent.OwnerUsage, error) {
	var rows []admincontent.OwnerUsage
	err := g.projects.Raw(ownersCTE+`
		SELECT owner, projects_bytes, files_bytes
		FROM o
		WHERE projects_bytes + files_bytes > 0
		ORDER BY projects_bytes + files_bytes DESC, owner
		LIMIT ?`, limit).Scan(&rows).Error
	return rows, err
}

func databaseBytes(db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, nil
	}
	var n int64
	err := db.Raw(`SELECT pg_database_size(current_database())`).Scan(&n).Error
	return n, err
}

func (g *AdminContentGatewayImpl) ProjectsDatabaseBytes() (int64, error) {
	return databaseBytes(g.projects)
}

func (g *AdminContentGatewayImpl) LicensingDatabaseBytes() (int64, error) {
	return databaseBytes(g.licensing)
}
