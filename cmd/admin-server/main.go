package main

import (
	"ai-video/internal/app"
	"ai-video/internal/config"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/repository"
	"ai-video/internal/router"
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
	if err := app.MigrateUserCenterColumns(config.DB); err != nil {
		panic(fmt.Sprintf("migrate user center failed: %v", err))
	}
	if err := app.SeedAppUserAdmin(); err != nil {
		panic(fmt.Sprintf("seed user center permissions failed: %v", err))
	}
	// Seed default config values into DB and warm the Redis cache.
	if err := setting.Init(context.Background()); err != nil {
		config.Log.Warnf("init settings: %v", err)
	}
	if count, err := repository.NewUserAttributionRepo().SyncUsers(context.Background()); err != nil {
		config.Log.Warnf("sync user attributions: %v", err)
	} else if count > 0 {
		config.Log.Infof("created %d missing user attribution records", count)
	}

	engine := router.NewRouter(
		AdminDist,
		admin.New(),
		api.New(),
	)

	addr := fmt.Sprintf(":%d", config.Cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: engine}

	go func() {
		config.Log.Infof("server starting at %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Log.Fatalf("server run failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	config.Log.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		config.Log.Errorf("server forced to shutdown: %v", err)
	}
	config.Close()
	config.Log.Info("server stopped")
}
