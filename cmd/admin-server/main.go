package main

import (
	"ai-video/internal/adjustevent"
	"ai-video/internal/config"
	"ai-video/internal/generation"
	"ai-video/internal/pkg/adjust"
	"ai-video/internal/pkg/monitor"
	"ai-video/internal/pkg/task"
	"ai-video/internal/router"
	"ai-video/internal/scheduledtask"
	"ai-video/internal/server/admin"
	"ai-video/internal/server/api"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

var AdminDist embed.FS

func main() {
	os.Exit(realMain())
}

func realMain() (exitCode int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			monitor.ReportPanic(config.Log, "admin-server", recovered, debug.Stack())
			exitCode = 1
		}
		config.Close()
	}()

	if err := run(); err != nil {
		monitor.Report(config.Log, monitor.KindProcessExit, "admin-server", err)
		return 1
	}
	return 0
}

func run() error {
	cfgFile := flag.String("config", "", "config file path")
	flag.Parse()

	if err := config.Init(*cfgFile); err != nil {
		return fmt.Errorf("init app failed: %w", err)
	}

	workerRestartDelay, workerRestartMaxDelay := config.Cfg.Task.WorkerRestartBackoff()
	taskManager := task.NewManager(task.ManagerConfig{
		RedisAddr:       config.Cfg.Redis.Addr(),
		RedisUsername:   config.Cfg.Redis.Username,
		RedisPassword:   config.Cfg.Redis.Password,
		RedisDB:         config.Cfg.Redis.DB,
		Concurrency:     config.Cfg.Task.Concurrency,
		Queues:          config.Cfg.Task.Queues,
		RestartDelay:    workerRestartDelay,
		RestartMaxDelay: workerRestartMaxDelay,
		ErrorHandler: func(ctx context.Context, taskType string, err error) {
			monitor.Report(config.Logger(ctx), monitor.KindTaskFailure, "task_worker", err, "task_type", taskType)
		},
		RestartHandler: func(err error, nextDelay time.Duration) {
			monitor.Report(
				config.Log,
				monitor.KindError,
				"task_worker_supervisor",
				err,
				"restart_in", nextDelay.String(),
			)
		},
	})
	defer taskManager.Close()
	if config.Cfg.Adjust.TrackerSyncEnabled {
		trackerClient, trackerErr := adjust.NewClient(adjust.ClientConfig{
			APIToken: config.Cfg.Adjust.CampaignAppToken,
			BaseURL:  config.Cfg.Adjust.CampaignBaseURL,
		})
		if trackerErr != nil {
			return fmt.Errorf("init Adjust tracker sync client failed: %w", trackerErr)
		}
		syncAdjustTrackers, trackerErr := scheduledtask.NewAdjustTrackerSync(
			trackerClient,
			config.Cfg.Adjust.CampaignAppToken,
			func(ctx context.Context, result scheduledtask.AdjustTrackerSyncResult) {
				config.Logger(ctx).Infof(
					"Adjust tracker sync completed: root_token=%s requests=%d trackers=%d",
					result.RootToken,
					result.Requests,
					result.Trackers,
				)
			},
		)
		if trackerErr != nil {
			return fmt.Errorf("init Adjust tracker sync task failed: %w", trackerErr)
		}
		if trackerErr = taskManager.RegisterPeriodic(task.PeriodicTasks{
			scheduledtask.TypeSyncAdjustTrackers: {
				Every: scheduledtask.AdjustTrackerSyncInterval,
				Run:   syncAdjustTrackers,
			},
		}); trackerErr != nil {
			return fmt.Errorf("register Adjust tracker sync task failed: %w", trackerErr)
		}
		config.Log.Infof("Adjust tracker sync scheduled: interval=%s", scheduledtask.AdjustTrackerSyncInterval)
	} else {
		config.Log.Info("Adjust tracker sync is disabled by adjust.tracker_sync_enabled")
	}

	eventContext, stopAdjustEvents := context.WithCancel(context.Background())
	defer stopAdjustEvents()
	var adjustEvents *adjustevent.Runtime
	if config.Cfg.Adjust.EventEnabled {
		createdAdjustEvents, eventErr := adjustevent.NewRuntime(adjustevent.RuntimeConfig{
			Client: taskManager.Client, Worker: taskManager.Worker,
			AuthToken: config.Cfg.Adjust.CampaignAppToken, BaseURL: config.Cfg.Adjust.EventBaseURL,
			Environment: adjust.Environment(strings.ToLower(strings.TrimSpace(config.Cfg.Adjust.EventEnvironment))),
		})
		if eventErr != nil {
			return fmt.Errorf("init Adjust event runtime failed: %w", eventErr)
		}
		adjustEvents = createdAdjustEvents
		if eventErr = adjustEvents.Start(eventContext); eventErr != nil {
			return fmt.Errorf("start Adjust event runtime failed: %w", eventErr)
		}
		adjustevent.SetDefault(adjustEvents)
		defer adjustevent.SetDefault(nil)
		config.Log.Infof("Adjust event Redis runtime started: environment=%s", config.Cfg.Adjust.EventEnvironment)
	} else {
		config.Log.Warn("Adjust event reporting is disabled by adjust.event_enabled")
	}

	subscriptionExpirationAt, subscriptionExpirationEnabled, err := config.Cfg.Task.SubscriptionExpirationTime(time.Local)
	if err != nil {
		return fmt.Errorf("parse subscription expiration execution time failed: %w", err)
	}
	if err = scheduledtask.RegisterSubscriptionExpiration(
		taskManager,
		subscriptionExpirationAt,
		func(ctx context.Context, expiredUsers int64) {
			if expiredUsers > 0 {
				config.Logger(ctx).Infof("subscription expiration task completed: expired_users=%d", expiredUsers)
			}
		},
	); err != nil {
		return fmt.Errorf("register scheduled tasks failed: %w", err)
	}
	if !subscriptionExpirationEnabled {
		config.Log.Info("subscription expiration task is not scheduled: task.subscription_expiration_at is empty")
	} else if subscriptionExpirationAt.After(time.Now()) {
		config.Log.Infof("subscription expiration task scheduled once at %s", subscriptionExpirationAt.Format(time.RFC3339))
	} else {
		config.Log.Infof("subscription expiration task is not scheduled: configured time %s has passed", subscriptionExpirationAt.Format(time.RFC3339))
	}

	engine := router.NewRouter(
		AdminDist,
		admin.New(),
		api.New(),
	)

	addr := fmt.Sprintf(":%d", config.Cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: engine}
	generation.Start()
	runtimeErr := make(chan error, 3)
	if adjustEvents != nil {
		go func() {
			for {
				select {
				case eventErr, ok := <-adjustEvents.Errors():
					if !ok {
						return
					}
					monitor.Report(config.Log, monitor.KindError, "adjust_event_runtime", eventErr)
				case <-eventContext.Done():
					return
				}
			}
		}()
	}

	go func() {
		config.Log.Infof("server starting at %s", addr)
		defer reportComponentPanic("http_server", runtimeErr)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			runtimeErr <- fmt.Errorf("server run failed: %w", serveErr)
		}
	}()
	go func() {
		defer reportComponentPanic("task_runtime", runtimeErr)
		if taskErr := taskManager.Start(); taskErr != nil {
			runtimeErr <- fmt.Errorf("task runtime stopped: %w", taskErr)
		}
	}()
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)
	var runtimeFailure error
	select {
	case sig := <-quit:
		config.Log.Infow("shutdown signal received", "signal", sig.String())
	case runtimeFailure = <-runtimeErr:
		monitor.Report(config.Log, monitor.KindComponentExit, "admin-server", runtimeFailure)
	}

	config.Log.Info("shutting down server...")
	stopAdjustEvents()
	adjustevent.SetDefault(nil)
	generation.Stop()
	taskManager.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var shutdownErr error
	if err = srv.Shutdown(ctx); err != nil {
		shutdownErr = fmt.Errorf("server forced to shutdown: %w", err)
		monitor.Report(config.Log, monitor.KindComponentExit, "http_server_shutdown", shutdownErr)
	}
	config.Log.Info("server stopped")
	return errors.Join(runtimeFailure, shutdownErr)
}

func reportComponentPanic(source string, runtimeErr chan<- error) {
	if recovered := recover(); recovered != nil {
		stack := debug.Stack()
		monitor.ReportPanic(config.Log, source, recovered, stack)
		runtimeErr <- fmt.Errorf("%s panicked: %v", source, recovered)
	}
}
