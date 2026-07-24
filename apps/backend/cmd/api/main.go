package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/api"
	"go.uber.org/zap"
)

func main() {
	cfg := api.LoadConfig()
	deps, err := api.Open(cfg)
	if err != nil {
		panic("initialize dependencies: " + err.Error())
	}
	defer deps.Close()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(cfg, deps),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		deps.Logger.Info("backend server started on " + server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-signalContext.Done():
		deps.Logger.Info(
			"shutdown signal received",
			zap.Duration("timeout", cfg.ShutdownTimeout),
		)
	case serveErr := <-serverErrors:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			deps.Logger.Fatal("backend server stopped unexpectedly", zap.Error(serveErr))
		}
		return
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		deps.Logger.Error("graceful shutdown timed out", zap.Error(err))
		if closeErr := server.Close(); closeErr != nil {
			deps.Logger.Error("force close server", zap.Error(closeErr))
		}
		return
	}
	deps.Logger.Info("graceful shutdown completed")
}
