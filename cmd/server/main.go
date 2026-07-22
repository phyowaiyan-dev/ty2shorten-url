package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phyowaiyan-dev/ty2shorten-url/internal/bootstrap"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/config"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/database"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/server"
	"gorm.io/gorm"
)

const shutdownTimeout = 10 * time.Second

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	cfg, err := config.Load(logger)
	if err != nil {
		logger.Error("load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if version != "" && version != "dev" {
		cfg.Version = version
	}
	if commit != "" && commit != "unknown" {
		cfg.Commit = commit
	}
	if buildTime != "" && buildTime != "unknown" {
		cfg.BuildTime = buildTime
	}

	var dbClose func()
	var gormDB *gorm.DB
	bootstrapMode := !bootstrap.Exists(cfg.AppConfigPath)
	if !bootstrapMode {
		bootstrapConfig, err := bootstrap.Load(cfg.AppConfigPath)
		if err != nil {
			logger.Error("load bootstrap config", slog.String("error", err.Error()))
			os.Exit(1)
		}
		openedDB, err := bootstrap.NewConnector().Connect(context.Background(), bootstrapConfig.Database)
		if err != nil {
			logger.Error("open configured database", slog.String("error", err.Error()))
			os.Exit(1)
		}
		gormDB = openedDB
		dbClose = func() {
			if err := database.Close(openedDB); err != nil {
				logger.Error("close database", slog.String("error", err.Error()))
			}
		}
		if err := database.AutoMigrate(openedDB); err != nil {
			logger.Error("migrate database", slog.String("error", err.Error()))
			os.Exit(1)
		}
	} else {
		logger.Info("bootstrap config not found; starting database setup mode", slog.String("config_path", cfg.AppConfigPath))
	}
	defer func() {
		if dbClose != nil {
			dbClose()
		}
	}()

	httpServer, err := server.NewHTTPServer(server.Dependencies{
		Config:        cfg,
		DB:            gormDB,
		Logger:        logger,
		BootstrapMode: bootstrapMode,
	})
	if err != nil {
		logger.Error("build http server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			logger.Error("http address already in use",
				slog.String("addr", httpServer.Addr),
				slog.String("hint", "Stop the existing ty2shorten-url process or start with another APP_PORT, for example APP_PORT=8723."),
				slog.String("error", err.Error()),
			)
		} else {
			logger.Error("bind http address", slog.String("addr", httpServer.Addr), slog.String("error", err.Error()))
		}
		os.Exit(1)
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening",
			slog.String("addr", httpServer.Addr),
			slog.String("version", cfg.Version),
			slog.String("commit", cfg.Commit),
			slog.String("build_time", cfg.BuildTime),
			slog.String("env", cfg.AppEnv),
		)
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}
		serverErrors <- nil
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-shutdownSignals:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErrors:
		if err != nil {
			logger.Error("http server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx, httpServer); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := <-serverErrors; err != nil {
		logger.Error("http server stopped unexpectedly", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("http server stopped")
}
