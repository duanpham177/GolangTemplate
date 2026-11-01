package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"servergo/internal/config"
	"servergo/internal/handler"
	"servergo/internal/repository"
	"servergo/internal/server"
	"servergo/internal/service"
	tcptransport "servergo/internal/transport/tcp"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Wire the HTTP and TCP transports to the same device service.
	userRepo := repository.NewUserMemory()
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

	telemetryRepo := repository.NewTelemetryMemory()
	tcpManager := tcptransport.NewManager()
	deviceSvc := service.NewDeviceService(telemetryRepo, tcpManager)
	deviceHandler := handler.NewDeviceHandler(deviceSvc, cfg.APIToken)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.NewRouter(logger, userHandler, deviceHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}
	tcpServer := tcptransport.NewServer(
		cfg.TCPAddr,
		cfg.TCPToken,
		cfg.TLSCertFile,
		cfg.TLSKeyFile,
		deviceSvc,
		tcpManager,
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	httpErr := make(chan error, 1)
	tcpDone := make(chan error, 1)
	go func() {
		logger.Info("server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpErr <- fmt.Errorf("HTTP server: %w", err)
		}
	}()
	go func() {
		logger.Info("starting TCP server", "addr", cfg.TCPAddr)
		tcpDone <- tcpServer.Serve(ctx)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	var runErr error
	tcpFinished := false
	select {
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-httpErr:
		runErr = err
	case err := <-tcpDone:
		tcpFinished = true
		if err != nil {
			runErr = err
		}
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		if runErr == nil {
			runErr = fmt.Errorf("shutdown HTTP server: %w", err)
		}
	}
	if !tcpFinished {
		select {
		case err := <-tcpDone:
			if err != nil && runErr == nil {
				runErr = err
			}
		case <-shutdownCtx.Done():
			if runErr == nil {
				runErr = errors.New("TCP server did not stop before shutdown deadline")
			}
		}
	}
	logger.Info("server stopped")
	return runErr
}
