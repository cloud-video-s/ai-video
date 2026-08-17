package main

import (
	"ai-video/internal/config"
	"ai-video/internal/generation"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/pkg/task"
	"ai-video/internal/router"
	"ai-video/internal/scheduledtask"
	"ai-video/internal/server/admin"
	"ai-video/internal/server/api"
	"context"
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var AdminDist embed.FS

func main() {
	cfgFile := flag.String("config", "", "config file path")
	flag.Parse()

	if err := config.Init(*cfgFile); err != nil {
		panic(fmt.Sprintf("init app failed: %v", err))
	}
	if err := setting.Init(context.Background()); err != nil {
		config.Log.Warnf("init settings: %v", err)
	}

	taskManager := task.NewManager(task.ManagerConfig{
		RedisAddr: config.Cfg.Redis.Addr(), RedisPassword: config.Cfg.Redis.Password, RedisDB: config.Cfg.Redis.DB,
		Concurrency: config.Cfg.Task.Concurrency, Queues: config.Cfg.Task.Queues,
		ErrorHandler: func(ctx context.Context, taskType string, err error) {
			config.Logger(ctx).Errorf("task failed: type=%s err=%v", taskType, err)
		},
	})
	subscriptionExpirationAt, subscriptionExpirationEnabled, err := config.Cfg.Task.SubscriptionExpirationTime(time.Local)
	if err != nil {
		taskManager.Close()
		config.Close()
		panic(fmt.Sprintf("parse subscription expiration execution time failed: %v", err))
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
		taskManager.Close()
		config.Close()
		panic(fmt.Sprintf("register scheduled tasks failed: %v", err))
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
	runtimeErr := make(chan error, 2)

	go func() {
		config.Log.Infof("server starting at %s", addr)
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			runtimeErr <- fmt.Errorf("server run failed: %w", err)
		}
	}()
	go func() {
		if err = taskManager.Worker.Start(); err != nil {
			runtimeErr <- fmt.Errorf("task worker stopped: %w", err)
		}
	}()
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
	case err = <-runtimeErr:
		config.Log.Errorf("service component stopped unexpectedly: %v", err)
	}

	config.Log.Info("shutting down server...")
	taskManager.Worker.Stop()
	taskManager.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		config.Log.Errorf("server forced to shutdown: %v", err)
	}
	config.Close()
	config.Log.Info("server stopped")
}
