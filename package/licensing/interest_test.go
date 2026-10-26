package licensing

import (
	"errors"
	"strings"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

func TestNormalizeInterestAcceptsKnownAnswer(t *testing.T) {
	got, err := NormalizeInterest(models.FeatureInterestCore{
		Role: " club_teacher ", Feature: "parent_progress", Source: "landing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != "club_teacher" || got.Source != "landing" {
		t.Fatalf("not normalized: %+v", got)
	}
}

func TestNormalizeInterestRefusesBadAnswers(t *testing.T) {
	bad := []models.FeatureInterestCore{
		{Role: "admin", Feature: "parent_progress"},
		{Role: "club_teacher", Feature: "free_robots"},
		{Role: "club_teacher", Feature: "other"}, // "other" needs a text
		{Role: "club_teacher", Feature: "other", Other: strings.Repeat("я", InterestOtherMaxRunes+1)},
	}
	for _, in := range bad {
		if _, err := NormalizeInterest(in); !errors.Is(err, ErrBadInterest) {
			t.Fatalf("accepted %+v", in)
		}
	}
}

func TestNormalizeInterestDropsUnknownSource(t *testing.T) {
	got, err := NormalizeInterest(models.FeatureInterestCore{Role: "parent", Feature: "club_showcase", Source: "evil"})
	if err != nil || got.Source != "" {
		t.Fatalf("source not dropped: %+v %v", got, err)
	}
}
