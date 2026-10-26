package usecase

import (
	"context"
	"log"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/projectPage"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// Defaults keep only the latest .sb3 of a project (no history, like Scratch).
// Raise keepLast / keepDailyDays in config to bring version history back.
const (
	defaultRetentionKeepLast = 1
	defaultRetentionInterval = time.Hour
	pruneCandidatesBatch     = 100
)

// PruneSb3Versions deletes the versions the policy drops, across all projects.
// Every candidate has at least one such version, so the loop terminates.
func (p *ProjectPageUseCaseImpl) PruneSb3Versions(policy projectPage.Sb3RetentionPolicy) (int64, error) {
	var total int64
	for {
		ids, err := p.projectPageGateway.ListSb3PruneCandidates(policy, pruneCandidatesBatch)
		if err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}
		var batchDeleted int64
		for _, id := range ids {
			n, err := p.projectPageGateway.PruneSb3Versions(id, policy)
			if err != nil {
				return total, err
			}
			batchDeleted += n
		}
		total += batchDeleted
		if batchDeleted == 0 {
			// Only a concurrent save can make a candidate keep everything; retry next run.
			return total, nil
		}
	}
}

// pruneAfterSave drops the project's superseded versions right after an upload,
// so it does not wait for the next worker run. Failure only delays the cleanup.
func (p *ProjectPageUseCaseImpl) pruneAfterSave(storageProjectID string) {
	if !retentionEnabled() {
		return
	}
	if _, err := p.projectPageGateway.PruneSb3Versions(storageProjectID, retentionPolicyFromConfig()); err != nil {
		log.Printf("projectPage: prune after save %s: %v", storageProjectID, err)
	}
}

// retentionEnabled: on unless projectPage.versionRetention.enabled is set to false.
func retentionEnabled() bool {
	key := "projectPage.versionRetention.enabled"
	return !viper.IsSet(key) || viper.GetBool(key)
}

func retentionPolicyFromConfig() projectPage.Sb3RetentionPolicy {
	keepLast := viper.GetInt("projectPage.versionRetention.keepLast")
	if keepLast <= 0 {
		keepLast = defaultRetentionKeepLast
	}
	keepDaily := max(viper.GetInt("projectPage.versionRetention.keepDailyDays"), 0)
	return projectPage.Sb3RetentionPolicy{KeepLast: keepLast, KeepDailyDays: keepDaily}
}

// StartSb3RetentionWorker periodically deletes .sb3 versions the policy drops
// (irreversible). The first run after deploy also removes the accumulated history.
func StartSb3RetentionWorker(uc projectPage.UseCase, lc fx.Lifecycle) {
	if !retentionEnabled() {
		return
	}
	policy := retentionPolicyFromConfig()
	interval := viper.GetDuration("projectPage.versionRetention.interval")
	if interval < time.Minute {
		interval = defaultRetentionInterval
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("projectPage: sb3 retention on (keepLast=%d, keepDailyDays=%d, every %s)",
				policy.KeepLast, policy.KeepDailyDays, interval)
			go func() {
				defer close(doneCh)
				run := func() {
					n, err := uc.PruneSb3Versions(policy)
					if err != nil {
						log.Printf("projectPage: PruneSb3Versions: %v", err)
					}
					if n > 0 {
						log.Printf("projectPage: pruned %d old .sb3 version(s)", n)
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
