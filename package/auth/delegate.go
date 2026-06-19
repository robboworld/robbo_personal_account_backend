package auth

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type Delegate interface {
	SignIn(email, password string, role uint) (accessToken string, refreshToken string, err error)
	SignUp(userHttp *models.UserHTTP) (accessToken string, refreshToken string, err error)
	SignUpCore(userCore *models.UserCore) (accessToken string, refreshToken string, err error)
	UserIdentity(c *gin.Context) (id string, role models.Role, err error)
	UserAccess(currentRole models.Role, roles []models.Role, ctx context.Context) (err error)
	RefreshToken(refreshToken string) (newAccessToken string, err error)
}
