package resolvers

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// profileSelfOrAdminAccess allows a user to update their own profile (matching JWT user_id)
// or admins listed in adminRoles to update any profile.
func profileSelfOrAdminAccess(
	ctx context.Context,
	ginCtx *gin.Context,
	authDelegate auth.Delegate,
	targetUserID string,
	selfRole models.Role,
	adminRoles []models.Role,
) error {
	callerRole := ginCtx.Value("user_role").(models.Role)
	callerID, _ := ginCtx.Value("user_id").(string)
	if callerRole == selfRole && strings.TrimSpace(targetUserID) == strings.TrimSpace(callerID) {
		return nil
	}
	return authDelegate.UserAccess(callerRole, adminRoles, ctx)
}
