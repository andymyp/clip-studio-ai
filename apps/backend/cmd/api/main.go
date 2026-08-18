package main

import (
	"context"
	"errors"
	"net/http"
	"os"
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

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	serverErrors := make(chan error, 1)
	go func() {
		deps.Logger.Info("backend server started on " + server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case received := <-signals:
		deps.Logger.Info(
			"shutdown signal received",
			zap.String("signal", received.String()),
			zap.Duration("timeout", cfg.ShutdownTimeout),
		)
	case serveErr := <-serverErrors:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			deps.Logger.Error("backend server stopped unexpectedly", zap.Error(serveErr))
		}
		return
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	shutdownResult := make(chan error, 1)
	go func() {
		shutdownResult <- server.Shutdown(shutdownContext)
	}()

	select {
	case err := <-shutdownResult:
		if err != nil {
			deps.Logger.Warn("HTTP drain timed out; force closing", zap.Error(err))
			forceClose(server, deps.Logger)
			return
		}
		deps.Logger.Info("graceful shutdown completed")
	case received := <-signals:
		deps.Logger.Warn(
			"second shutdown signal received; force closing",
			zap.String("signal", received.String()),
		)
		forceClose(server, deps.Logger)
		waitForShutdown(shutdownResult, deps.Logger)
	case <-shutdownContext.Done():
		deps.Logger.Warn("HTTP drain deadline reached; force closing")
		forceClose(server, deps.Logger)
		waitForShutdown(shutdownResult, deps.Logger)
	}
}

func forceClose(server *http.Server, logger *zap.Logger) {
	if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("force close server", zap.Error(err))
		return
	}
	logger.Info("backend server stopped")
}

func waitForShutdown(result <-chan error, logger *zap.Logger) {
	// Server.Close unblocks Shutdown. Wait for that goroutine before deferred
	// dependency cleanup so an in-flight shutdown cannot outlive its resources.
	if err := <-result; err != nil &&
		!errors.Is(err, http.ErrServerClosed) &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) {
		logger.Warn("HTTP shutdown ended with an error", zap.Error(err))
	}
}
