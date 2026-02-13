package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"signal/internal/auth"
	"signal/internal/config"
	"signal/internal/httpapi"
	"signal/internal/logger"
	"signal/internal/room"
	"signal/internal/version"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		panic("config validation failed: " + err.Error())
	}
	log := logger.New(cfg.LogLevel)
	defer func() { _ = log.Sync() }()
	log.Info("starting signal server",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("buildTime", version.BuildTime),
	)

	mgr := room.NewManager(log)
	jwtAuth := auth.NewJWT(cfg.Security.JWTSecret)

	httpSrv := httpapi.NewServer(cfg, log, mgr, jwtAuth)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("http server exited unexpectedly", zap.Error(err))
		}
	}()

	<-ctx.Done()
	stop()
	log.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("server stopped")
}
