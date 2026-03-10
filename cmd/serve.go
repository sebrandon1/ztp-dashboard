package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sebrandon1/ztp-dashboard/internal/ai"
	"github.com/sebrandon1/ztp-dashboard/internal/api"
	"github.com/sebrandon1/ztp-dashboard/internal/hub"
	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
	"github.com/sebrandon1/ztp-dashboard/internal/spoke"
	"github.com/sebrandon1/ztp-dashboard/internal/ws"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the dashboard web server",
	Long:  `Starts the HTTP server with embedded React frontend and REST API for managing ZTP hub/spoke clusters.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(_ *cobra.Command, _ []string) error {
	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8sClient, err := k8s.NewClient(cfg.KubeConfig)
	if err != nil {
		slog.Warn("could not connect to Kubernetes cluster", "error", err)
		slog.Info("dashboard will start but cluster features will be unavailable")
	}

	hubManager := hub.NewManager(k8sClient)

	aiClient := ai.NewClient(cfg.OllamaEndpoint, cfg.OllamaModel)

	wsHub := ws.NewHub()
	go wsHub.Run(ctx)

	if k8sClient != nil {
		watcher := ws.NewWatcher(k8sClient, wsHub)
		watcher.OnEvent = func(event ws.WatchEvent) {
			api.RecordEvent(event)
		}
		go watcher.Start(ctx)
	}

	clientPool := k8s.NewClientPool(10*time.Minute, 20)
	spokeService := spoke.NewService(k8sClient, clientPool)

	srv := api.NewServer(cfg, k8sClient, hubManager, aiClient, wsHub, spokeService)

	const maxPortAttempts = 10
	var listener net.Listener
	port := cfg.Port
	for i := range maxPortAttempts {
		addr := fmt.Sprintf(":%d", port+i)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			if i > 0 {
				slog.Warn("configured port in use, using next available", "configured", cfg.Port, "actual", port+i)
			}
			port += i
			break
		}
		slog.Warn("port unavailable, trying next", "port", port+i, "error", err)
	}
	if listener == nil {
		return fmt.Errorf("could not find available port after %d attempts starting from %d", maxPortAttempts, cfg.Port)
	}

	httpServer := &http.Server{
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
		cancel()
	}()

	slog.Info("starting ZTP dashboard", "port", port)
	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
