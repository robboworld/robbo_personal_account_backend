package auth

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrKidAccount: a kid account may not write data about itself. Same text as
// teacherclass.ErrKidAccount, which the SPA already translates.
var ErrKidAccount = errors.New("kid_account")

// A kid account is made by a teacher for the class sign-in (architecture/ADR_CLASS_KID_LOGIN.md).
// Its address is a stand-in, <username>@kids.robbo.invalid, and no parent agreed to anything
// beyond what the teacher typed. So such an account writes no data about the child itself: its
// own profile and survey answers are refused (ErrKidAccount). Admins still edit it as before.

// KidEmailDomain is the stand-in domain of kid accounts. The class sign-in creates them with it.
const KidEmailDomain = "kids.robbo.invalid"

// kidAccountKey marks a request whose session belongs to a kid account; set by the auth middleware.
const kidAccountKey = "kid_account"

// IsKidAccountEmail reports whether email is a kid account's stand-in address.
func IsKidAccountEmail(email string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(email)), "@"+KidEmailDomain)
}

// MarkKidAccountSession is called by the auth middleware with the email of the signed session.
func MarkKidAccountSession(c *gin.Context, email string) {
	if IsKidAccountEmail(email) {
		c.Set(kidAccountKey, true)
	}
}

// IsKidAccountSession reports whether the request comes from a kid account's session.
func IsKidAccountSession(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(kidAccountKey)
	kid, _ := v.(bool)
	return ok && kid
}
