package main

import (
	"log"
	"net/http"

	"github.com/Noon-R/Devport/relay/api"
	"github.com/Noon-R/Devport/relay/config"
	"github.com/Noon-R/Devport/relay/store"
	"github.com/Noon-R/Devport/relay/ws"
)

func main() {
	cfg := config.Load()
	connStore := store.NewStore()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Registration API
	registerHandler := api.NewRegisterHandler(cfg, connStore)
	mux.Handle("/api/relay/register", registerHandler)

	// Refresh API
	refreshHandler := api.NewRefreshHandler(cfg, connStore)
	mux.Handle("/api/relay/refresh", refreshHandler)

	// WebSocket handlers
	relayHandler := ws.NewRelayHandler(cfg, connStore)
	mux.Handle("/relay", relayHandler)

	clientHandler := ws.NewClientHandler(cfg, connStore, relayHandler)
	mux.Handle("/ws", clientHandler)

	// Start server
	addr := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("Relay server starting on %s", addr)
	log.Printf("  Health: http://%s/health", addr)
	log.Printf("  Register: http://%s/api/relay/register", addr)
	log.Printf("  Relay WS: ws://{subdomain}.%s/relay", cfg.Domain)
	log.Printf("  Client WS: ws://{subdomain}.%s/ws", cfg.Domain)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
