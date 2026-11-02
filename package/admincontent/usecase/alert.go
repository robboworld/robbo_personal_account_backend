package usecase

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/admincontent"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

const (
	defaultAlertInterval = 15 * time.Minute
	// While space stays low the alert repeats at most this often.
	alertRepeat = 24 * time.Hour
)

// SuperAdmins lists the LMS ids of the accounts that get the alert.
type SuperAdmins func() ([]string, error)

func LMSSuperAdmins() ([]string, error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	ids, err := reader.ListActiveSuperuserIDs()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	return out, nil
}

// storageAlert decides when to alert: on dropping below the threshold, then at most once per
// alertRepeat while it stays below; climbing back above re-arms it. The state lives in memory:
// CreateOrUpdateByDedupe marks the notification unread on every upsert, so it cannot rate-limit.
type storageAlert struct {
	below  int64
	repeat time.Duration
	low    bool
	last   time.Time
}

func (a *storageAlert) due(free int64, now time.Time) bool {
	if free >= a.below {
		a.low = false
		return false
	}
	return !a.low || now.Sub(a.last) >= a.repeat
}

func (a *storageAlert) sent(now time.Time) {
	a.low = true
	a.last = now
}

func gb(bytes int64) string {
	return fmt.Sprintf("%.1f ГБ", float64(bytes)/(1<<30))
}

// StorageAlerter sends the low-space alert to every super admin.
type StorageAlerter struct {
	Volume        admincontent.Volume
	Notifications notifications.Gateway
	SuperAdmins   SuperAdmins
	UploadsStop   int64
	state         storageAlert
}

func NewStorageAlerter(volume admincontent.Volume, notif notifications.Gateway, admins SuperAdmins, uploadsStop int64) *StorageAlerter {
	return &StorageAlerter{
		Volume: volume, Notifications: notif, SuperAdmins: admins, UploadsStop: uploadsStop,
		state: storageAlert{below: AlertBelowBytes(), repeat: alertRepeat},
	}
}

// Check runs once; it returns how many admins were notified.
func (a *StorageAlerter) Check(now time.Time) (int, error) {
	free, err := a.Volume.FreeBytes()
	if err != nil || free < 0 {
		return 0, err
	}
	if !a.state.due(free, now) {
		return 0, nil
	}
	ids, err := a.SuperAdmins()
	if err != nil {
		return 0, err
	}
	actionURL := "/admin/storage"
	body := fmt.Sprintf("На томе файлов ЛК свободно %s (порог оповещения %s). При %s загрузки файлов остановятся.",
		gb(free), gb(a.state.below), gb(a.UploadsStop))
	sentTo := 0
	for _, id := range ids {
		dedupe := "storage_low:" + id + ":" + now.UTC().Format("2006-01-02")
		if err := a.Notifications.CreateOrUpdateByDedupe(&models.UserNotificationDB{
			RecipientUserID: id,
			Title:           "Заканчивается место на сервере",
			Body:            body,
			Kind:            "storage_low",
			Severity:        "WARNING",
			Source:          "system",
			ActionURL:       &actionURL,
			DedupeKey:       &dedupe,
		}); err != nil {
			log.Printf("admincontent: storage alert to %s: %v", id, err)
			continue
		}
		sentTo++
	}
	if sentTo > 0 {
		a.state.sent(now)
	}
	return sentTo, nil
}

// StartStorageAlertWorker checks the volume every adminStorage.alert.interval. Off while user
// files are off (no volume).
func StartStorageAlertWorker(a *StorageAlerter, lc fx.Lifecycle) {
	if a == nil || a.Volume == nil || a.Notifications == nil {
		return
	}
	interval := viper.GetDuration("adminStorage.alert.interval")
	if interval < time.Minute {
		interval = defaultAlertInterval
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("admincontent: storage alert on (below %s, every %s)", gb(a.state.below), interval)
			go func() {
				defer close(doneCh)
				run := func() {
					if n, err := a.Check(time.Now()); err != nil {
						log.Printf("admincontent: storage alert: %v", err)
					} else if n > 0 {
						log.Printf("admincontent: low disk space, alerted %d super admin(s)", n)
					}
				}
				run()
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-stopCh:
						return
					case <-ticker.C:
						run()
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(stopCh)
			select {
			case <-doneCh:
			case <-ctx.Done():
			}
			return nil
		},
	})
}
