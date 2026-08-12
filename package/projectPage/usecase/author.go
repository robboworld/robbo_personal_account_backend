package usecase

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
)

type authorInfo struct {
	Name     string
	AvatarID string
}

func lookupAuthorInfo(ownerUserID string) authorInfo {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return authorInfo{}
	}
	fallback := fmt.Sprintf("User %s", ownerUserID)
	id, err := strconv.ParseInt(ownerUserID, 10, 64)
	if err != nil || id <= 0 {
		return authorInfo{Name: fallback}
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return authorInfo{Name: fallback}
	}
	defer reader.Close()

	profile, err := reader.LookupAuthUserProfileByID(id)
	if err != nil || profile == nil {
		return authorInfo{Name: fallback}
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
		name = fallback
	}
	return authorInfo{
		Name:     name,
		AvatarID: lmsdb.ParseAvatarIDFromMeta(profile.Meta),
	}
}

func lookupAuthorName(ownerUserID string) string {
	return lookupAuthorInfo(ownerUserID).Name
}
