package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/operationeth/audiobookshelf-exporter/internal/api"
	"github.com/operationeth/audiobookshelf-exporter/internal/metrics"
)

// Default exporter port
const defaultPort = ":9860"

func main() {
	// Initialize zap logger
	var logger *zap.Logger
	if os.Getenv("LOG_FORMAT") == "console" || os.Getenv("ENVIRONMENT") == "development" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	absURL := os.Getenv("ABS_URL")
	if absURL == "" {
		logger.Fatal("ABS_URL required")
	}

	apiKey := os.Getenv("ABS_API_KEY")

	// Default scrape interval: 30 seconds
	interval := 30 * time.Second
	if v := os.Getenv("SCRAPE_INTERVAL_SECONDS"); v != "" {
		if s, err := strconv.Atoi(v); err == nil {
			interval = time.Duration(s) * time.Second
		}
	}

	// Default listening port: 9860
	listen := defaultPort
	if p := os.Getenv("EXPORTER_PORT"); p != "" {
		listen = ":" + p
	}

	logger.Info("starting Audiobookshelf exporter",
		zap.String("ABS_URL", absURL),
		zap.String("EXPORTER_PORT", listen),
		zap.String("SCRAPE_INTERVAL", interval.String()),
	)

	client := api.NewClient(absURL, apiKey)
	exp := metrics.NewExporter(client)

	go func() {
		logger.Info("starting scrape loop")
		exp.Run(interval)
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := http.Server{
		Addr:         listen,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting HTTP server", zap.String("address", listen))
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Warn("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown server gracefully", zap.Error(err))
	}

	logger.Info("exporter stopped cleanly")
}
