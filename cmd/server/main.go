package main

import (
	"context"
	"errors"
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
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	// Wiring: repository -> service -> handler -> router
	userRepo := repository.NewUserMemory()
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.NewRouter(logger, userHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on Ctrl+C / SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
	logger.Info("server stopped")
}
