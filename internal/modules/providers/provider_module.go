package providers

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Options(
	fx.Provide(NewClient),
	fx.Provide(NewProviderManager),
	fx.Provide(NewRuntimeStatus),
	fx.Invoke(RegisterProvider),
)

func RegisterProvider(lc fx.Lifecycle, pm *ProviderManager, c *Client, status *RuntimeStatus, log *zap.Logger) {
	pm.Register("gemini", c)

	// Select Gemini as the active provider
	if err := pm.SelectProvider("gemini"); err != nil {
		log.Error("Failed to select Gemini provider", zap.Error(err))
	}

	initCtx, cancelInit := context.WithCancel(context.Background())
	done := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("Starting Gemini provider initialization in background")
			go func() {
				defer close(done)
				initialize := func(ctx context.Context) error {
					err := c.Init(ctx)
					if err != nil && ctx.Err() == nil {
						log.Warn("Gemini provider initialization failed; retrying")
					}
					return err
				}
				retryInitialConnection(initCtx, initialize, status, waitForInitialRetry)
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancelInit()
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
			status.setStopped()
			return c.Close()
		},
	})
}
