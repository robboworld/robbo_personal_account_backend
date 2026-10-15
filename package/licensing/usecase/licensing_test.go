package usecase

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	licrypto "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/crypto"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

const testFingerprint = "fp-device-a"

// fakeGateway keeps one licence and its seats in memory; methods the tests do not reach panic
// through the embedded nil interface.
type fakeGateway struct {
	licensing.Gateway
	license *models.LicenseCore
	seats   map[string]*models.SeatCore // by fingerprint
}

func (g *fakeGateway) GetLicenseByID(id string) (*models.LicenseCore, error) {
	if g.license == nil || g.license.ID != id {
		return nil, licensing.ErrLicenseNotFound
	}
	copied := *g.license
	return &copied, nil
}

func (g *fakeGateway) UpdateLicense(license *models.LicenseCore) error {
	copied := *license
	g.license = &copied
	return nil
}

func (g *fakeGateway) GetProductTitle(string) (string, error) {
	return "", errors.New("no product")
}

func (g *fakeGateway) GetSeatByFingerprint(licenseID, fingerprint string) (*models.SeatCore, error) {
	seat, ok := g.seats[fingerprint]
	if !ok || seat.LicenseID != licenseID {
		return nil, licensing.ErrSeatNotFound
	}
	copied := *seat
	return &copied, nil
}

func (g *fakeGateway) TouchSeat(string, time.Time) error {
	return nil
}

func (g *fakeGateway) DeleteSeatByFingerprint(licenseID, fingerprint string) (*models.SeatCore, error) {
	seat, err := g.GetSeatByFingerprint(licenseID, fingerprint)
	if err != nil {
		return nil, err
	}
	delete(g.seats, fingerprint)
	return seat, nil
}

type fixture struct {
	gateway *fakeGateway
	usecase *LicensingUseCaseImpl
	signer  *licrypto.Signer
}

func newFixture(t *testing.T, expiresAt time.Time) *fixture {
	t.Helper()
	previous := ownerDisplayName
	ownerDisplayName = func(lmsUserID string) string { return "Owner of " + lmsUserID }
	t.Cleanup(func() { ownerDisplayName = previous })
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer := &licrypto.Signer{PrivateKey: key, PublicKey: &key.PublicKey, Kid: "test-1", Issuer: "test"}
	gateway := &fakeGateway{
		license: &models.LicenseCore{
			ID:           "lic-1",
			LmsUserID:    "42",
			LicenseKey:   "KEY-1",
			Status:       models.LicenseStatusActive,
			SeatLimit:    1,
			Capabilities: []string{"premium.auto_update"},
			ExpiresAt:    expiresAt,
		},
		seats: map[string]*models.SeatCore{
			testFingerprint: {LicenseID: "lic-1", SeatID: "seat-1", DeviceFingerprint: testFingerprint},
		},
	}
	return &fixture{
		gateway: gateway,
		usecase: &LicensingUseCaseImpl{gateway: gateway, signer: signer},
		signer:  signer,
	}
}

// token issued earlier for seat-1 of lic-1, valid until exp.
func (f *fixture) token(t *testing.T, exp time.Time) string {
	t.Helper()
	token, err := f.signer.SignJWT(licrypto.TokenClaims{
		EntitlementID: "lic-1",
		SeatID:        "seat-1",
		Exp:           exp.Unix(),
		DeviceBinding: testFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestRefreshTokenCarriesRenewedExpiryAndTariff(t *testing.T) {
	renewed := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second)
	f := newFixture(t, renewed)
	// The old token expired yesterday: a renewal in the ЛК must still reach the device.
	result, err := f.usecase.RefreshToken(f.token(t, time.Now().Add(-24*time.Hour)), testFingerprint, "https://lk.example")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	claims, err := f.signer.VerifyJWT(result.SignedOfflineToken)
	if err != nil {
		t.Fatalf("new token does not verify: %v", err)
	}
	if claims.Exp != renewed.Unix() {
		t.Errorf("exp = %d, want the licence expiry %d", claims.Exp, renewed.Unix())
	}
	if claims.SeatID != "seat-1" || claims.DeviceBinding != testFingerprint {
		t.Errorf("seat/binding = %q/%q", claims.SeatID, claims.DeviceBinding)
	}
	if claims.Tariff != "Individual" {
		t.Errorf("tariff = %q, want Individual", claims.Tariff)
	}
	if claims.OwnerName != "Owner of 42" {
		t.Errorf("ownerName = %q, want the account name for a personal licence", claims.OwnerName)
	}
	if !strings.HasPrefix(result.AddonManifestUrl, "https://lk.example/") {
		t.Errorf("manifest url = %q", result.AddonManifestUrl)
	}
}

func TestClassLicenceTokenHasNoOwnerName(t *testing.T) {
	f := newFixture(t, time.Now().Add(30*24*time.Hour))
	f.gateway.license.SeatLimit = 20
	result, err := f.usecase.RefreshToken(f.token(t, time.Now().Add(time.Hour)), testFingerprint, "")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	claims, err := f.signer.VerifyJWT(result.SignedOfflineToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Tariff != "Class" || claims.OwnerName != "" {
		t.Errorf("tariff/owner = %q/%q, want Class and no owner", claims.Tariff, claims.OwnerName)
	}
}

func TestRefreshTokenReasons(t *testing.T) {
	future := time.Now().Add(30 * 24 * time.Hour)
	cases := []struct {
		name   string
		mutate func(f *fixture)
		fp     string
		want   error
	}{
		{"licence expired", func(f *fixture) { f.gateway.license.ExpiresAt = time.Now().Add(-time.Hour) }, testFingerprint, licensing.ErrSubscriptionExpired},
		{"licence revoked", func(f *fixture) { f.gateway.license.Status = models.LicenseStatusRevoked }, testFingerprint, licensing.ErrLicenseRevoked},
		{"licence deleted", func(f *fixture) { f.gateway.license.ID = "other" }, testFingerprint, licensing.ErrLicenseRevoked},
		{"seat removed in the ЛК", func(f *fixture) { delete(f.gateway.seats, testFingerprint) }, testFingerprint, licensing.ErrSeatRemoved},
		{"device activated again", func(f *fixture) { f.gateway.seats[testFingerprint].SeatID = "seat-2" }, testFingerprint, licensing.ErrSeatRemoved},
		{"another computer", func(*fixture) {}, "fp-device-b", licensing.ErrDeviceBinding},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, future)
			token := f.token(t, future)
			tc.mutate(f)
			if _, err := f.usecase.RefreshToken(token, tc.fp, ""); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestRefreshTokenRejectsForgedToken(t *testing.T) {
	future := time.Now().Add(30 * 24 * time.Hour)
	f := newFixture(t, future)
	token := f.token(t, future)
	forged := token[:len(token)-4] + "AAAA"
	if _, err := f.usecase.RefreshToken(forged, testFingerprint, ""); !errors.Is(err, licensing.ErrInvalidSignature) {
		t.Fatalf("err = %v, want ErrInvalidSignature", err)
	}
	if _, err := f.usecase.RefreshToken("", testFingerprint, ""); !errors.Is(err, licensing.ErrAddonAuthRequired) {
		t.Fatalf("empty token: err = %v", err)
	}
}

func TestReleaseSeatFreesTheSeatWithoutKey(t *testing.T) {
	f := newFixture(t, time.Now().Add(30*24*time.Hour))
	// Expired token: an expired licence can still give its seat back.
	token := f.token(t, time.Now().Add(-time.Hour))
	seatID, err := f.usecase.ReleaseSeat(token, testFingerprint)
	if err != nil {
		t.Fatalf("ReleaseSeat: %v", err)
	}
	if seatID != "seat-1" {
		t.Errorf("seatID = %q", seatID)
	}
	if _, ok := f.gateway.seats[testFingerprint]; ok {
		t.Error("seat still present")
	}
	if f.gateway.license.LastSeatChangeAt == nil {
		t.Error("LastSeatChangeAt not set (reactivation cooldown)")
	}
	if _, err := f.usecase.ReleaseSeat(token, testFingerprint); !errors.Is(err, licensing.ErrSeatNotFound) {
		t.Fatalf("second release: err = %v, want ErrSeatNotFound", err)
	}
}

func TestReleaseSeatKeepsANewerActivation(t *testing.T) {
	f := newFixture(t, time.Now().Add(30*24*time.Hour))
	token := f.token(t, time.Now().Add(time.Hour))
	f.gateway.seats[testFingerprint].SeatID = "seat-2"
	if _, err := f.usecase.ReleaseSeat(token, testFingerprint); !errors.Is(err, licensing.ErrSeatNotFound) {
		t.Fatalf("err = %v, want ErrSeatNotFound", err)
	}
	if _, ok := f.gateway.seats[testFingerprint]; !ok {
		t.Error("the newer seat was deleted")
	}
}

func TestReleaseSeatOtherDevice(t *testing.T) {
	f := newFixture(t, time.Now().Add(30*24*time.Hour))
	token := f.token(t, time.Now().Add(time.Hour))
	if _, err := f.usecase.ReleaseSeat(token, "fp-device-b"); !errors.Is(err, licensing.ErrDeviceBinding) {
		t.Fatalf("err = %v, want ErrDeviceBinding", err)
	}
}
