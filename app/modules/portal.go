package modules

import (
	"context"
	"log"
	"time"

	oidchttp "github.com/skinnykaen/robbo_student_personal_account.git/package/oidc/http"
	portalgateway "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/gateway"
	portalhttp "github.com/skinnykaen/robbo_student_personal_account.git/package/portal/http"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/portal/worker"
	"go.uber.org/fx"
)

type PortalModule struct {
	fx.Out
	Gateway                    portalgateway.Gateway
	PortalNotificationsHandler portalhttp.NotificationsHandler
	OIDCHandler                *oidchttp.Handler
}

func SetupPortalModule() PortalModule {
	gw := portalgateway.SetupPortalGateway()
	var oidcHandler *oidchttp.Handler
	if h, err := oidchttp.NewHandler(gw.Gateway); err != nil {
		log.Printf("[oidc] routes disabled: %v", err)
	} else {
		oidcHandler = &h
	}
	return PortalModule{
		Gateway:                    gw.Gateway,
		PortalNotificationsHandler: portalhttp.NewNotificationsHandler(gw.Gateway),
		OIDCHandler:                oidcHandler,
	}
}

func StartPortalOutboxWorker(portal portalgateway.Gateway, useCase UseCaseModule, lc fx.Lifecycle) {
	w := worker.NewOutboxWorker(portal, useCase.EdxUseCase)
	var cancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var workerCtx context.Context
			workerCtx, cancel = context.WithCancel(context.Background())
			go w.RunLoop(workerCtx, 30*time.Second)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}
			return nil
		},
	})
}
