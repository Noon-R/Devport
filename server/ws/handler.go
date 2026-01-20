package ws

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/Noon-R/Devport/server/config"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Handler struct {
	cfg   *config.Config
	conns sync.Map // map[string]*ConnState
}

type ConnState struct {
	conn          *websocket.Conn
	authenticated bool
	sessionID     string
	mu            sync.Mutex
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "connection closed")

	ctx := r.Context()
	state := &ConnState{
		conn:          conn,
		authenticated: false,
	}

	log.Printf("New WebSocket connection established")

	// Message loop
	for {
		var req JSONRPCRequest
		if err := wsjson.Read(ctx, conn, &req); err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Printf("WebSocket closed: %v", websocket.CloseStatus(err))
			} else {
				log.Printf("Read error: %v", err)
			}
			return
		}

		resp := h.handleRequest(ctx, state, &req)
		if resp != nil {
			state.mu.Lock()
			err := wsjson.Write(ctx, conn, resp)
			state.mu.Unlock()
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}

// SendNotification sends a notification to the client (no id field)
func (h *Handler) SendNotification(ctx context.Context, state *ConnState, method string, params interface{}) error {
	notification := &JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return wsjson.Write(ctx, state.conn, notification)
}
