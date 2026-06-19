package usecase

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
)

func lookupAuthorName(ownerUserID string) string {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return ""
	}
	id, err := strconv.ParseInt(ownerUserID, 10, 64)
	if err != nil || id <= 0 {
		return fmt.Sprintf("User %s", ownerUserID)
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return fmt.Sprintf("User %s", ownerUserID)
	}
	defer reader.Close()

	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return fmt.Sprintf("User %s", ownerUserID)
	}
	name := strings.TrimSpace(profile.ProfileName)
	if name == "" {
		parts := []string{strings.TrimSpace(profile.FirstName), strings.TrimSpace(profile.LastName)}
		var nonEmpty []string
		for _, p := range parts {
			if p != "" {
				nonEmpty = append(nonEmpty, p)
			}
		}
		name = strings.Join(nonEmpty, " ")
	}
	if name == "" {
		name = strings.TrimSpace(profile.Username)
	}
	if name == "" {
		return fmt.Sprintf("User %s", ownerUserID)
	}
	return name
}
