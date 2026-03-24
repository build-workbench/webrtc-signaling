package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/LessUp/aurora-signal/internal/auth"
	"github.com/LessUp/aurora-signal/internal/config"
	"github.com/LessUp/aurora-signal/internal/httpapi"
	"github.com/LessUp/aurora-signal/internal/logger"
	"github.com/LessUp/aurora-signal/internal/room"
	"github.com/LessUp/aurora-signal/internal/version"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	cfg := config.Load()
	warnings, err := cfg.Validate()
	if err != nil {
		panic("config validation failed: " + err.Error())
	}
	log := logger.New(cfg.LogLevel)
	defer func() { _ = log.Sync() }()
	for _, w := range warnings {
		log.Warn(w)
	}
	log.Info("starting signal server",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("buildTime", version.BuildTime),
	)

	mgr := room.NewManager(log)
	mgr.StartCleanup(30*time.Second, 5*time.Minute)
	jwtAuth := auth.NewJWT(cfg.Security.JWTSecret)

	httpSrv, err := httpapi.NewServer(cfg, log, mgr, jwtAuth)
	if err != nil {
		log.Fatal("http server initialization failed", zap.Error(err))
	}

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
	mgr.Stop()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("server stopped")
}

func runHealthcheck() int {
	cfg := config.Load()
	addr := cfg.Server.Addr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	url := "http://" + addr + "/healthz"
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
