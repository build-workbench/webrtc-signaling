package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"singal/internal/auth"
	"singal/internal/config"
	"singal/internal/httpapi"
	"singal/internal/logger"
	"singal/internal/room"
)

func main() {
	cfg := config.Load()
	log := logger.New()
	defer func() { _ = log.Sync() }()

	mgr := room.NewManager(log)
	jwtAuth := auth.NewJWT(cfg.Security.JWTSecret)

	httpSrv := httpapi.NewServer(cfg, log, mgr, jwtAuth)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.Start(); err != nil {
			log.Fatal("http server exited", zapError(err))
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zapError(err))
	}
}

// zapError is a small helper to avoid importing zap in main directly
func zapError(err error) zap.Field { return zap.Error(err) }
