// Package adminaudit records what a super admin does with other users' content. It has no
// dependencies on the content packages, so projectPage and admincontent can both write to it.
package adminaudit

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// MinReasonRunes: every moderator action on someone's content needs a reason.
const MinReasonRunes = 3

var ErrReasonRequired = errors.New("REASON_REQUIRED")

// NormalizeReason trims the reason and refuses one shorter than MinReasonRunes.
func NormalizeReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) < MinReasonRunes {
		return "", ErrReasonRequired
	}
	return reason, nil
}

type Gateway interface {
	Write(entry *models.AdminContentAuditDB) error
	ListForTarget(targetUserID string, limit int) ([]models.AdminContentAuditDB, error)
}
