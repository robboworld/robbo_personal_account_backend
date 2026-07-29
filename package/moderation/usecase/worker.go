package usecase

import (
	"context"
	"log"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/moderation"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

const defaultExpiryPollInterval = 2 * time.Minute

// StartBanExpiryWorker periodically auto-unbans users whose temporary bans expired.
func StartBanExpiryWorker(uc moderation.UseCase, lc fx.Lifecycle) {
	interval := viper.GetDuration("moderation.expiryPollInterval")
	if interval < time.Minute {
		interval = defaultExpiryPollInterval
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer close(doneCh)
				run := func() {
					n, err := uc.ProcessExpiredBans()
					if err != nil {
						log.Printf("moderation: ProcessExpiredBans: %v", err)
					} else if n > 0 {
						log.Printf("moderation: auto-unbanned %d user(s)", n)
					}
				}
				run() // do not wait a full poll interval after restart
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
