package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Noon-R/Devport/server/api"
	"github.com/Noon-R/Devport/server/config"
	"github.com/Noon-R/Devport/server/qr"
	"github.com/Noon-R/Devport/server/relay"
	"github.com/Noon-R/Devport/server/ws"
)

func main() {
	cfg := config.Load()

	if cfg.AuthToken == "" {
		log.Fatal("AUTH_TOKEN environment variable is required")
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebSocket endpoint
	wsHandler := ws.NewHandler(cfg)
	mux.Handle("/ws", wsHandler)

	// File system API
	fsHandler := api.NewFSHandler(cfg.WorkDir, cfg.AuthToken)
	mux.Handle("/api/fs/", fsHandler)
	mux.Handle("/api/fs", fsHandler)

	// Git API
	gitHandler := api.NewGitHandler(cfg.WorkDir, cfg.AuthToken)
	mux.Handle("/api/git/", gitHandler)

	// Chat REST API (for reliable message delivery)
	chatHandler := api.NewChatHandler(cfg.AuthToken, wsHandler.GetSessionStore(), wsHandler.GetProcessManager())
	mux.Handle("/api/sessions/", chatHandler)
	mux.Handle("/api/permissions/", chatHandler)
	mux.Handle("/api/questions/", chatHandler)

	// Static files (production mode)
	if !cfg.DevMode {
		mux.Handle("/", http.FileServer(http.Dir("./static")))
	}

	// Start relay client if enabled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var relayClient *relay.Client
	var remoteURL string

	if cfg.RelayEnabled {
		relayClient = relay.NewClient(cfg)
		if err := relayClient.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start relay client: %v", err)
		} else {
			// Wait briefly for relay to connect
			time.Sleep(500 * time.Millisecond)
			remoteURL = relayClient.GetRemoteURL()
		}
	}

	// Start HTTP server
	addr := ":" + cfg.ServerPort
	localURL := fmt.Sprintf("http://localhost%s", addr)

	// Print startup banner with QR code
	qr.PrintStartupBanner(localURL, remoteURL)

	// Setup graceful shutdown
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")

		if relayClient != nil {
			relayClient.Stop()
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
