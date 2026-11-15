package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"gorm.io/gorm"
)

// interestGateway records what the usecase passes down; other methods panic through the nil interface.
type interestGateway struct {
	licensing.Gateway
	saved      []models.FeatureInterestCore
	getErr     error
	listLimit  int
	listOffset int
	listRows   []models.FeatureInterestCore
	listTotal  int64
	listErr    error
}

func (g *interestGateway) SaveFeatureInterest(a *models.FeatureInterestCore) error {
	g.saved = append(g.saved, *a)
	return nil
}

func (g *interestGateway) GetFeatureInterest(string) (*models.FeatureInterestCore, error) {
	return nil, g.getErr
}

func (g *interestGateway) ListFeatureInterest(limit, offset int) ([]models.FeatureInterestCore, int64, error) {
	g.listLimit, g.listOffset = limit, offset
	return g.listRows, g.listTotal, g.listErr
}

func TestListFeatureInterestClampsPaging(t *testing.T) {
	cases := []struct {
		name                  string
		limit, offset         int
		wantLimit, wantOffset int
	}{
		{"zero limit falls back to default", 0, 0, 50, 0},
		{"negative limit falls back to default", -5, 0, 50, 0},
		{"limit over the cap falls back to default", 201, 0, 50, 0},
		{"cap itself is allowed", 200, 0, 200, 0},
		{"normal page is kept", 20, 40, 20, 40},
		{"negative offset becomes zero", 20, -1, 20, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := &interestGateway{}
			u := &LicensingUseCaseImpl{gateway: g}
			if _, _, err := u.ListFeatureInterest(tc.limit, tc.offset); err != nil {
				t.Fatal(err)
			}
			if g.listLimit != tc.wantLimit || g.listOffset != tc.wantOffset {
				t.Fatalf("gateway got limit=%d offset=%d, want %d/%d", g.listLimit, g.listOffset, tc.wantLimit, tc.wantOffset)
			}
		})
	}
}

func TestListFeatureInterestPassesRowsTotalAndError(t *testing.T) {
	rows := []models.FeatureInterestCore{{Role: "parent", Feature: "club_showcase"}}
	g := &interestGateway{listRows: rows, listTotal: 7}
	u := &LicensingUseCaseImpl{gateway: g}
	got, total, err := u.ListFeatureInterest(10, 0)
	if err != nil || total != 7 || len(got) != 1 || got[0].Role != "parent" {
		t.Fatalf("got %+v total=%d err=%v", got, total, err)
	}

	boom := errors.New("db down")
	u = &LicensingUseCaseImpl{gateway: &interestGateway{listErr: boom}}
	if _, _, err := u.ListFeatureInterest(10, 0); !errors.Is(err, boom) {
		t.Fatalf("want the gateway error, got %v", err)
	}
}

func TestSaveFeatureInterestRefusesBadAnswerWithoutTouchingGateway(t *testing.T) {
	g := &interestGateway{}
	u := &LicensingUseCaseImpl{gateway: g}
	err := u.SaveFeatureInterest(models.FeatureInterestCore{Role: "hacker", Feature: "club_showcase"})
	if !errors.Is(err, licensing.ErrBadInterest) {
		t.Fatalf("want ErrBadInterest, got %v", err)
	}
	if len(g.saved) != 0 {
		t.Fatal("a bad answer reached the gateway")
	}
}

func TestSaveFeatureInterestStoresTrimmedAnswer(t *testing.T) {
	g := &interestGateway{}
	u := &LicensingUseCaseImpl{gateway: g}
	if err := u.SaveFeatureInterest(models.FeatureInterestCore{
		LmsUserID: "42", Role: " parent ", Feature: "club_showcase", Org: "  Club, Kazan ", Source: "evil",
	}); err != nil {
		t.Fatal(err)
	}
	got := g.saved[0]
	if got.Role != "parent" || got.Org != "Club, Kazan" || got.Source != "" || got.LmsUserID != "42" {
		t.Fatalf("unexpected stored answer: %+v", got)
	}
}

func TestGetFeatureInterestNotFoundIsNilNil(t *testing.T) {
	u := &LicensingUseCaseImpl{gateway: &interestGateway{getErr: gorm.ErrRecordNotFound}}
	got, err := u.GetFeatureInterest("42")
	if got != nil || err != nil {
		t.Fatalf("want nil, nil; got %+v, %v", got, err)
	}
	boom := errors.New("db down")
	u = &LicensingUseCaseImpl{gateway: &interestGateway{getErr: boom}}
	if _, err := u.GetFeatureInterest("42"); !errors.Is(err, boom) {
		t.Fatalf("want the gateway error, got %v", err)
	}
}
