package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arth/taskmanager/internal/handler"
	"github.com/arth/taskmanager/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))
	slog.SetDefault(logger)

	port := getEnv("PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "")

	if databaseURL == "" {
		logger.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	logger.Info("connecting to database...")
	taskStore, err := store.NewPostgresStore(databaseURL)
	if err != nil {
		logger.Error("failed to initialize store", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = taskStore.Close() //nolint:errcheck
	}()
	logger.Info("database connection established")

	taskHandler := handler.NewTaskHandler(taskStore, logger)

	mux := http.NewServeMux()
	taskHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(logger, mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := server.Shutdown(ctx); err != nil {
		cancel()
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1) //nolint:gocritic
	}
	cancel()

	logger.Info("server stopped gracefully")
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getLogLevel() slog.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func init() {
	fmt.Println("Task Manager API v1.0.0")
}
