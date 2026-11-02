package usecase

import (
	"context"
	"log"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/userfiles"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

const defaultGCInterval = time.Hour

// StartGCWorker periodically removes deleted files for good (after userFiles.retainDeletedDays)
// and cut-off uploads. Off while the feature is off.
func StartGCWorker(uc userfiles.UseCase, lc fx.Lifecycle) {
	if uc == nil || !uc.Enabled() {
		return
	}
	interval := viper.GetDuration("userFiles.gc.interval")
	if interval < time.Minute {
		interval = defaultGCInterval
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("userfiles: gc on (every %s)", interval)
			go func() {
				defer close(doneCh)
				run := func() {
					n, err := uc.CollectGarbage(time.Now().UTC())
					if err != nil {
						log.Printf("userfiles: CollectGarbage: %v", err)
					}
					if n > 0 {
						log.Printf("userfiles: removed %d deleted file(s) for good", n)
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
