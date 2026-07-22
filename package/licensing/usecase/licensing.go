package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	licgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/gateway"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	licrypto "github.com/skinnykaen/robbo_student_personal_account.git/package/licensing/crypto"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type LicensingUseCaseImpl struct {
	gateway licensing.Gateway
	signer  *licrypto.Signer
}

type LicensingUseCaseModule struct {
	fx.Out
	licensing.UseCase
}

func SetupLicensingUseCase(gateway licensing.Gateway) LicensingUseCaseModule {
	keyPath := viper.GetString("licensing.privateKeyPath")
	kid := viper.GetString("licensing.jwtKid")
	issuer := viper.GetString("licensing.jwtIssuer")
	if keyPath == "" {
		panic("licensing.privateKeyPath is required")
	}
	signer, err := licrypto.LoadSigner(keyPath, kid, issuer)
	if err != nil {
		panic(err)
	}
	return LicensingUseCaseModule{
		UseCase: &LicensingUseCaseImpl{gateway: gateway, signer: signer},
	}
}

func (u *LicensingUseCaseImpl) IssueLicense(input models.IssueLicenseInput) (*models.LicenseCore, error) {
	if input.LmsUserID == "" {
		return nil, licensing.ErrBadRequest
	}
	seatLimit := input.SeatLimit
	if seatLimit <= 0 {
		seatLimit = viper.GetInt("licensing.defaultSeatLimit")
		if seatLimit <= 0 {
			seatLimit = 1
		}
	}
	caps := input.Capabilities
	if len(caps) == 0 {
		caps = []string{models.CapabilityPremiumAuto}
	}
	expiresAt := input.ExpiresAt
	if expiresAt.IsZero() {
		days := viper.GetInt("licensing.defaultExpiresDays")
		if days <= 0 {
			days = 365
		}
		expiresAt = time.Now().UTC().AddDate(0, 0, days)
	}
	key, err := licgateway.NewLicenseKey()
	if err != nil {
		return nil, err
	}
	source := input.Source
	if source == "" {
		source = models.LicenseSourceAdmin
	}
	license := &models.LicenseCore{
		LmsUserID:    input.LmsUserID,
		LicenseKey:   key,
		Status:       models.LicenseStatusActive,
		Source:       source,
		SeatLimit:    seatLimit,
		Capabilities: caps,
		ExpiresAt:    expiresAt.UTC(),
		IssuedBy:     input.IssuedBy,
		Note:         input.Note,
		ProductID:    input.ProductID,
	}
	return u.gateway.CreateLicense(license)
}

func (u *LicensingUseCaseImpl) ListMyLicenses(lmsUserID string) ([]*models.LicenseCore, error) {
	return u.gateway.ListLicensesByUser(lmsUserID)
}

func (u *LicensingUseCaseImpl) GetLicense(lmsUserID, licenseID string) (*models.LicenseCore, error) {
	lic, err := u.gateway.GetLicenseByID(licenseID)
	if err != nil {
		return nil, err
	}
	if lic.LmsUserID != lmsUserID {
		return nil, licensing.ErrForbidden
	}
	return lic, nil
}

func (u *LicensingUseCaseImpl) RevokeSeat(lmsUserID, licenseID, seatID string) error {
	lic, err := u.GetLicense(lmsUserID, licenseID)
	if err != nil {
		return err
	}
	for _, s := range lic.Seats {
		if s.SeatID == seatID {
			_, err := u.gateway.DeleteSeatByFingerprint(lic.ID, s.DeviceFingerprint)
			return err
		}
	}
	return licensing.ErrSeatNotFound
}

func (u *LicensingUseCaseImpl) Activate(licenseKey, fingerprint, publicBase string) (*models.ActivateResult, error) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return nil, licensing.ErrDeviceFingerprint
	}
	lic, err := u.gateway.GetLicenseByKey(strings.TrimSpace(licenseKey))
	if err != nil {
		return nil, err
	}
	if lic.Status != models.LicenseStatusActive {
		return nil, licensing.ErrSubscriptionExpired
	}
	now := time.Now().UTC()
	if !lic.ExpiresAt.After(now) {
		return nil, licensing.ErrSubscriptionExpired
	}

	seat, err := u.gateway.GetSeatByFingerprint(lic.ID, fingerprint)
	if err == nil {
		_ = u.gateway.TouchSeat(seat.SeatID, now)
		return u.buildActivateResult(lic, seat.SeatID, fingerprint, publicBase, now)
	}
	if err != licensing.ErrSeatNotFound {
		return nil, err
	}

	count, err := u.gateway.CountSeats(lic.ID)
	if err != nil {
		return nil, err
	}
	if int(count) >= lic.SeatLimit {
		return nil, licensing.ErrSeatLimitReached
	}

	cooldown := viper.GetInt("licensing.seatReactivationCooldownSec")
	if cooldown <= 0 {
		cooldown = 60
	}
	if lic.LastSeatChangeAt != nil {
		if now.Sub(*lic.LastSeatChangeAt) < time.Duration(cooldown)*time.Second {
			return nil, licensing.ErrReactivationCooldown
		}
	}

	created, err := u.gateway.CreateSeat(&models.SeatCore{
		LicenseID:         lic.ID,
		DeviceFingerprint: fingerprint,
		ActivatedAt:       now,
		LastSeenAt:        now,
	})
	if err != nil {
		return nil, err
	}
	lic.LastSeatChangeAt = &now
	_ = u.gateway.UpdateLicense(lic)
	return u.buildActivateResult(lic, created.SeatID, fingerprint, publicBase, now)
}

func (u *LicensingUseCaseImpl) DeactivateSeat(licenseKey, fingerprint string) (string, error) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" || strings.TrimSpace(licenseKey) == "" {
		return "", licensing.ErrBadRequest
	}
	lic, err := u.gateway.GetLicenseByKey(strings.TrimSpace(licenseKey))
	if err != nil {
		return "", err
	}
	seat, err := u.gateway.DeleteSeatByFingerprint(lic.ID, fingerprint)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	lic.LastSeatChangeAt = &now
	_ = u.gateway.UpdateLicense(lic)
	return seat.SeatID, nil
}

func (u *LicensingUseCaseImpl) buildActivateResult(
	lic *models.LicenseCore,
	seatID, fingerprint, publicBase string,
	now time.Time,
) (*models.ActivateResult, error) {
	publicBase = strings.TrimRight(strings.TrimSpace(publicBase), "/")
	manifestURL := publicBase + "/addon/manifest.json"
	licenseIDClaim := fmt.Sprintf("lic-%s-%s", lic.ID, seatID)
	claims := licrypto.TokenClaims{
		Iss:              u.signer.Issuer,
		Sub:              licenseIDClaim,
		LicenseID:        licenseIDClaim,
		EntitlementID:    lic.ID,
		SeatID:           seatID,
		Iat:              now.Unix(),
		Nbf:              now.Unix(),
		Exp:              lic.ExpiresAt.Unix(),
		Capabilities:     append([]string{}, lic.Capabilities...),
		DeviceBinding:    fingerprint,
		AddonManifestUrl: manifestURL,
	}
	token, err := u.signer.SignJWT(claims)
	if err != nil {
		return nil, err
	}
	return &models.ActivateResult{
		SignedOfflineToken: token,
		AddonManifestUrl:   manifestURL,
		ServerTime:         now,
		RefreshAfter:       now.Add(7 * 24 * time.Hour),
		SeatID:             seatID,
		LicenseID:          licenseIDClaim,
		EntitlementID:      lic.ID,
	}, nil
}

func (u *LicensingUseCaseImpl) StartDeviceLink(fingerprint string) (*models.DeviceLinkSessionCore, error) {
	ttlMin := viper.GetInt("licensing.deviceLinkTTLMinutes")
	if ttlMin <= 0 {
		ttlMin = 10
	}
	deviceCode, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	userCode, err := randomUserCode()
	if err != nil {
		return nil, err
	}
	session := &models.DeviceLinkSessionCore{
		DeviceCode:        deviceCode,
		UserCode:          userCode,
		Status:            models.DeviceLinkPending,
		DeviceFingerprint: strings.TrimSpace(fingerprint),
		ExpiresAt:         time.Now().UTC().Add(time.Duration(ttlMin) * time.Minute),
	}
	return u.gateway.CreateDeviceLink(session)
}

func (u *LicensingUseCaseImpl) ConfirmDeviceLink(lmsUserID, userCode, licenseID string) (*models.DeviceLinkSessionCore, error) {
	session, err := u.gateway.GetDeviceLinkByUserCode(userCode)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if session.ExpiresAt.Before(now) || session.Status == models.DeviceLinkExpired {
		session.Status = models.DeviceLinkExpired
		_ = u.gateway.UpdateDeviceLink(session)
		return nil, licensing.ErrDeviceLinkExpired
	}
	if session.Status != models.DeviceLinkPending {
		return nil, licensing.ErrBadRequest
	}
	licenses, err := u.gateway.ListLicensesByUser(lmsUserID)
	if err != nil {
		return nil, err
	}
	var chosen *models.LicenseCore
	for _, lic := range licenses {
		if lic.Status != models.LicenseStatusActive || !lic.ExpiresAt.After(now) {
			continue
		}
		if licenseID != "" && lic.ID != licenseID {
			continue
		}
		chosen = lic
		break
	}
	if chosen == nil {
		return nil, licensing.ErrLicenseNotFound
	}
	session.Status = models.DeviceLinkConfirmed
	session.LmsUserID = lmsUserID
	session.LicenseID = chosen.ID
	session.ConfirmedAt = &now
	if err := u.gateway.UpdateDeviceLink(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (u *LicensingUseCaseImpl) PollDeviceLink(deviceCode, fingerprint, publicBase string) (*models.ActivateResult, string, error) {
	session, err := u.gateway.GetDeviceLinkByDeviceCode(deviceCode)
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	if session.ExpiresAt.Before(now) {
		session.Status = models.DeviceLinkExpired
		_ = u.gateway.UpdateDeviceLink(session)
		return nil, "", licensing.ErrDeviceLinkExpired
	}
	if session.Status == models.DeviceLinkPending {
		return nil, models.DeviceLinkPending, licensing.ErrDeviceLinkPending
	}
	if session.Status != models.DeviceLinkConfirmed && session.Status != models.DeviceLinkConsumed {
		return nil, "", licensing.ErrBadRequest
	}
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		fp = session.DeviceFingerprint
	}
	if fp == "" {
		return nil, "", licensing.ErrDeviceFingerprint
	}
	lic, err := u.gateway.GetLicenseByID(session.LicenseID)
	if err != nil {
		return nil, "", err
	}
	result, err := u.Activate(lic.LicenseKey, fp, publicBase)
	if err != nil {
		return nil, "", err
	}
	session.Status = models.DeviceLinkConsumed
	session.SeatID = result.SeatID
	session.DeviceFingerprint = fp
	_ = u.gateway.UpdateDeviceLink(session)
	return result, models.DeviceLinkConsumed, nil
}

func (u *LicensingUseCaseImpl) authAddon(token, fingerprint string) (*licrypto.TokenClaims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, licensing.ErrAddonAuthRequired
	}
	claims, err := u.signer.VerifyJWT(token)
	if err != nil {
		switch err.Error() {
		case "invalid_token_shape":
			return nil, licensing.ErrInvalidToken
		case "unsupported_alg":
			return nil, licensing.ErrUnsupportedAlg
		case "invalid_signature":
			return nil, licensing.ErrInvalidSignature
		case "license_expired":
			return nil, licensing.ErrLicenseExpiredToken
		default:
			return nil, licensing.ErrInvalidSignature
		}
	}
	fp := strings.TrimSpace(fingerprint)
	if claims.DeviceBinding == "" || claims.DeviceBinding != fp {
		return nil, licensing.ErrDeviceBinding
	}
	lic, err := u.gateway.GetLicenseByID(claims.EntitlementID)
	if err != nil {
		return nil, licensing.ErrSeatNotActive
	}
	if lic.Status != models.LicenseStatusActive {
		return nil, licensing.ErrSeatNotActive
	}
	seat, err := u.gateway.GetSeatByFingerprint(lic.ID, fp)
	if err != nil || seat.SeatID != claims.SeatID {
		return nil, licensing.ErrSeatNotActive
	}
	return claims, nil
}

func (u *LicensingUseCaseImpl) BuildAddonManifest(token, fingerprint string) (map[string]interface{}, error) {
	claims, err := u.authAddon(token, fingerprint)
	if err != nil {
		return nil, err
	}
	plain, err := readAddonPlaintext()
	if err != nil {
		return nil, err
	}
	manifest := map[string]interface{}{
		"id":           "rs3-paid-addon",
		"version":      viper.GetString("licensing.addonVersion"),
		"bundle":       "paid-addon.js.enc",
		"bundleSha256": licrypto.Sha256Hex(plain),
		"signatureKid": u.signer.Kid,
	}
	if manifest["version"] == "" {
		manifest["version"] = "0.2.0"
	}
	sig, err := u.signer.SignManifest(manifest)
	if err != nil {
		return nil, err
	}
	manifest["manifestSignature"] = sig
	_ = claims
	return manifest, nil
}

func (u *LicensingUseCaseImpl) EncryptAddonBundle(token, fingerprint string) (string, error) {
	claims, err := u.authAddon(token, fingerprint)
	if err != nil {
		return "", err
	}
	plain, err := readAddonPlaintext()
	if err != nil {
		return "", err
	}
	licenseID := claims.LicenseID
	if licenseID == "" {
		licenseID = claims.Sub
	}
	return licrypto.EncryptAddonAESGCM(plain, fingerprint, licenseID)
}

func readAddonPlaintext() ([]byte, error) {
	dir := viper.GetString("licensing.addonStaticDir")
	if dir == "" {
		return nil, fmt.Errorf("manifest_build_failed")
	}
	// Prefer local override from private rs3-paid-addon (gitignored); else placeholder.
	candidates := []string{
		filepath.Join(dir, "paid-addon.local.js"),
		filepath.Join(dir, "paid-addon.js"),
	}
	var lastErr error
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("manifest_build_failed")
	}
	return nil, fmt.Errorf("manifest_build_failed")
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomUserCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, 8)
	for i := range out {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out[0:4]) + "-" + string(out[4:8]), nil
}
