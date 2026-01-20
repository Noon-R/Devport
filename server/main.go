package main

import (
	"log"
	"net/http"

	"github.com/Noon-R/Devport/server/config"
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

	// Static files (production mode)
	if !cfg.DevMode {
		mux.Handle("/", http.FileServer(http.Dir("./static")))
	}

	addr := ":" + cfg.ServerPort
	log.Printf("Devport server starting on %s", addr)
	log.Printf("  Health: http://localhost%s/health", addr)
	log.Printf("  WebSocket: ws://localhost%s/ws", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
