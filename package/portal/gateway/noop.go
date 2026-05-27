package gateway

import (
	"errors"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

var errPortalDisabled = errors.New("portal postgres disabled")

// NoopGateway satisfies Gateway when legacy/portal tables are not used (2-DB mode).
type NoopGateway struct{}

func (NoopGateway) UpsertUserLinkByOIDC(_, _, _, _ string) (*models.RobboPortalUserLinkDB, error) {
	return nil, errPortalDisabled
}

func (NoopGateway) FindUserLinkByEmail(_ string) (*models.RobboPortalUserLinkDB, error) {
	return nil, errPortalDisabled
}

func (NoopGateway) FindUserLinkByLegacyOrEdx(_ string) (*models.RobboPortalUserLinkDB, error) {
	return nil, errPortalDisabled
}

func (NoopGateway) FindPrimaryRoleByUserLinkID(_ int64) (models.Role, bool, error) {
	return models.Student, false, nil
}

func (NoopGateway) EnsureDefaultRole(_ int64, _ string) error {
	return nil
}

func (NoopGateway) EnqueueOutbox(_, _ string, _ interface{}) (bool, error) {
	return false, errPortalDisabled
}

func (NoopGateway) FetchPendingOutbox(_ int) ([]models.RobboPortalIntegrationOutboxDB, error) {
	return nil, nil
}

func (NoopGateway) MarkOutboxDone(_ int64) error {
	return nil
}

func (NoopGateway) MarkOutboxFailed(_ int64, _ string) error {
	return nil
}

func (NoopGateway) CreateNotification(_ *models.RobboPortalNotificationDB) (bool, error) {
	return false, errPortalDisabled
}
