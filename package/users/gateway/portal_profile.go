package gateway

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

func portalRoleCodeToModel(code string) models.Role {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "teacher":
		return models.Teacher
	case "parent":
		return models.Parent
	case "free_listener":
		return models.FreeListener
	case "unit_admin":
		return models.UnitAdmin
	case "super_admin":
		return models.SuperAdmin
	default:
		return models.Student
	}
}

func (r *UsersGatewayImpl) findPortalUserLinkByID(id string) (*models.RobboPortalUserLinkDB, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, auth.ErrUserNotFound
	}
	var link models.RobboPortalUserLinkDB
	err := r.PostgresClient.Db.
		Where("legacy_lk_user_id = ? OR edx_user_id = ?", id, id).
		First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *UsersGatewayImpl) portalPrimaryRoleForLink(linkID int64) models.Role {
	var row models.RobboPortalRoleDB
	err := r.PostgresClient.Db.Where("user_link_id = ?", linkID).Order("id ASC").First(&row).Error
	if err != nil {
		return models.Student
	}
	return portalRoleCodeToModel(row.RoleCode)
}

func (r *UsersGatewayImpl) lookupLMSUsername(edxUserID, email string) string {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return ""
	}
	defer reader.Close()

	if edxUserID != "" {
		if id, parseErr := strconv.ParseInt(strings.TrimSpace(edxUserID), 10, 64); parseErr == nil {
			if row, qErr := reader.LookupAuthUserByID(id); qErr == nil && row != nil {
				return strings.TrimSpace(row.Username)
			}
		}
	}
	if email != "" {
		if row, qErr := reader.LookupAuthUserByEmail(email); qErr == nil && row != nil {
			return strings.TrimSpace(row.Username)
		}
	}
	return ""
}

func portalFIOFromDisplayName(displayName string) (firstname, lastname, middlename string) {
	parts := strings.Fields(strings.TrimSpace(displayName))
	switch len(parts) {
	case 0:
		return "", "", ""
	case 1:
		return parts[0], "", ""
	case 2:
		return parts[1], parts[0], ""
	default:
		return parts[1], parts[0], strings.Join(parts[2:], " ")
	}
}

func (r *UsersGatewayImpl) userCoreFromPortalLink(link *models.RobboPortalUserLinkDB) models.UserCore {
	id := ""
	if link.EdxUserID != nil {
		id = strings.TrimSpace(*link.EdxUserID)
	} else if link.LegacyLKUserID != nil {
		id = strings.TrimSpace(*link.LegacyLKUserID)
	}
	username := r.lookupLMSUsername(id, link.Email)
	firstname, lastname, middlename := "", "", ""
	if link.DisplayName != nil {
		firstname, lastname, middlename = portalFIOFromDisplayName(*link.DisplayName)
	}
	if username == "" && link.DisplayName != nil {
		username = strings.TrimSpace(*link.DisplayName)
	}
	return models.UserCore{
		Id:         id,
		Email:      link.Email,
		Nickname:   username,
		Firstname:  firstname,
		Lastname:   lastname,
		Middlename: middlename,
		Role:       r.portalPrimaryRoleForLink(link.ID),
		CreatedAt:  link.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (r *UsersGatewayImpl) portalUserCoreByID(userID string) (models.UserCore, error) {
	link, err := r.findPortalUserLinkByID(userID)
	if err != nil {
		return models.UserCore{}, err
	}
	return r.userCoreFromPortalLink(link), nil
}

func portalDisplayNameFromFIO(core models.UserCore) string {
	var parts []string
	if s := strings.TrimSpace(core.Lastname); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(core.Firstname); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(core.Middlename); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

func (r *UsersGatewayImpl) updatePortalUserProfile(userID string, core models.UserCore) (models.UserCore, error) {
	link, err := r.findPortalUserLinkByID(userID)
	if err != nil {
		return models.UserCore{}, err
	}
	email := strings.TrimSpace(core.Email)
	if email == "" {
		return models.UserCore{}, auth.ErrUserNotFound
	}
	displayName := portalDisplayNameFromFIO(core)
	updates := map[string]interface{}{
		"email":      email,
		"updated_at": time.Now().UTC(),
	}
	if displayName != "" {
		updates["display_name"] = displayName
	}
	if err := r.PostgresClient.Db.Model(&models.RobboPortalUserLinkDB{}).
		Where("id = ?", link.ID).
		Updates(updates).Error; err != nil {
		return models.UserCore{}, err
	}
	var refreshed models.RobboPortalUserLinkDB
	if err := r.PostgresClient.Db.First(&refreshed, link.ID).Error; err != nil {
		return models.UserCore{}, err
	}
	return r.userCoreFromPortalLink(&refreshed), nil
}
