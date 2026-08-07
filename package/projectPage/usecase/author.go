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

// lookupAuthorIDsForQuery returns LMS user IDs matching q (username/email/name).
// On LMS errors returns nil so title/tag search still works.
func lookupAuthorIDsForQuery(q string) []string {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil
	}
	defer reader.Close()
	hits, err := reader.SearchAuthUsersPrefix(q, 50, false)
	if err != nil || len(hits) == 0 {
		return nil
	}
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		if h.ID > 0 {
			out = append(out, strconv.FormatInt(h.ID, 10))
		}
	}
	return out
}
