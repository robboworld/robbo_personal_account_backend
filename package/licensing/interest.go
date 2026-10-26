package licensing

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// Interest survey for the paid editions while tariffs are not approved: role and the one
// feature needed first. The lists match the ЛК form (frontend/src/helpers/featureInterest.js).
var (
	InterestRoles    = []string{"club_teacher", "club_owner", "school_teacher", "parent", "student", "other"}
	InterestFeatures = []string{
		"parent_progress", // progress and portfolio of the child for parents
		"club_showcase",   // showcase of the club's projects
		"club_scenarios",  // lesson scenarios following the club programme
		"work_review",     // assignments and review of students' work
		"competitions",    // competition and festival mode
		"lesson_mode",     // one starting project for the group, fewer menus
		"other",
	}
	InterestSources = []string{"landing", "licenses", "rs-desktop", "rs-web"}
)

// InterestOtherMaxRunes caps the free-text answer.
const InterestOtherMaxRunes = 300

var ErrBadInterest = errors.New("BAD_INTEREST")

func oneOf(value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// NormalizeInterest checks an answer and trims it; an unknown source is dropped, not refused.
func NormalizeInterest(in models.FeatureInterestCore) (models.FeatureInterestCore, error) {
	in.Role = strings.TrimSpace(in.Role)
	in.Feature = strings.TrimSpace(in.Feature)
	in.Other = strings.TrimSpace(in.Other)
	in.Source = strings.TrimSpace(in.Source)
	if !oneOf(in.Role, InterestRoles) || !oneOf(in.Feature, InterestFeatures) {
		return in, ErrBadInterest
	}
	if utf8.RuneCountInString(in.Other) > InterestOtherMaxRunes {
		return in, ErrBadInterest
	}
	if in.Feature == "other" && in.Other == "" {
		return in, ErrBadInterest
	}
	if !oneOf(in.Source, InterestSources) {
		in.Source = ""
	}
	return in, nil
}
