# Devport 類似システム実装ガイド

Devport と同等の機能を持つ「モバイル向け AI プログラミング環境」を自作するための詳細な実装手順。

## 目次

1. [システム概要](#システム概要)
2. [Phase 1: バックエンド基盤](#phase-1-バックエンド基盤)
3. [Phase 2: Claude CLI 統合](#phase-2-claude-cli-統合)
4. [Phase 3: フロントエンド実装](#phase-3-フロントエンド実装)
5. [Phase 4: セッション管理](#phase-4-セッション管理)
6. [Phase 5: ファイル・Git 操作](#phase-5-ファイルgit-操作)
7. [Phase 6: リモートアクセス](#phase-6-リモートアクセス)
8. [Phase 7: iOS/Android アプリ化](#phase-7-iosandroid-アプリ化)

---

## システム概要

### 最終的なアーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────┐
│                        クライアント                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│  │  Web (React)    │  │  iOS (SwiftUI)  │  │ Android (Compose)│      │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘      │
│           │                    │                    │               │
│           └────────────────────┼────────────────────┘               │
│                                │                                    │
│                    WebSocket (JSON-RPC 2.0)                         │
└────────────────────────────────┼────────────────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │    (オプション)          │
                    │    リレーサーバー         │
                    │    NAT越え用            │
                    └────────────┬────────────┘
                                 │
┌────────────────────────────────┼────────────────────────────────────┐
│                        ローカル PC                                   │
│  ┌─────────────────────────────┴─────────────────────────────────┐  │
│  │                      Go Server                                 │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │  │
│  │  │  WebSocket   │  │   Session    │  │    File/Git          │ │  │
│  │  │  RPC Handler │  │   Manager    │  │    Operations        │ │  │
│  │  └──────┬───────┘  └──────────────┘  └──────────────────────┘ │  │
│  │         │                                                      │  │
│  │         │ stdin/stdout (stream-json)                           │  │
│  │         ▼                                                      │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │                   Claude CLI                              │ │  │
│  │  │  claude --output-format stream-json                       │ │  │
│  │  │         --input-format stream-json                        │ │  │
│  │  │         --permission-prompt-tool stdio                    │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 技術スタック選定

| コンポーネント | 推奨技術 | 代替案 |
|---------------|---------|--------|
| バックエンド | Go | Node.js, Rust |
| フロントエンド | React + Vite | Vue, Svelte |
| 状態管理 | Zustand | Redux, Jotai |
| 通信 | WebSocket + JSON-RPC 2.0 | Socket.IO |
| AI | Claude CLI | OpenAI API, Gemini |

---

## Phase 1: バックエンド基盤

### 1.1 プロジェクト初期化

```bash
mkdir my-devport
cd my-devport

# Go モジュール初期化
mkdir server && cd server
go mod init github.com/yourname/my-devport/server
```

### 1.2 ディレクトリ構造

```
server/
├── main.go              # エントリーポイント
├── ws/
│   ├── handler.go       # WebSocket ハンドラ
│   └── rpc.go           # JSON-RPC 処理
├── agent/
│   ├── agent.go         # Agent インターフェース
│   └── claude/
│       └── claude.go    # Claude CLI 実装
├── session/
│   └── store.go         # セッション管理
└── middleware/
    └── auth.go          # 認証
```

### 1.3 HTTP サーバー + WebSocket（main.go）

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yourname/my-devport/server/middleware"
	"github.com/yourname/my-devport/server/ws"
)

func main() {
	authToken := os.Getenv("AUTH_TOKEN")
	if authToken == "" {
		log.Fatal("AUTH_TOKEN is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9870"
	}

	// ルーティング
	mux := http.NewServeMux()

	// ヘルスチェック
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebSocket エンドポイント（認証付き）
	wsHandler := ws.NewHandler(authToken)
	mux.Handle("/ws", wsHandler)

	// 静的ファイル（本番用）
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
```

### 1.4 WebSocket ハンドラ（ws/handler.go）

```go
package ws

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Handler struct {
	authToken string
	conns     sync.Map
}

func NewHandler(authToken string) *Handler {
	return &Handler{authToken: authToken}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := r.Context()
	connState := &ConnState{
		conn:          conn,
		authenticated: false,
	}

	// メッセージループ
	for {
		var req JSONRPCRequest
		if err := wsjson.Read(ctx, conn, &req); err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		resp := h.handleRequest(ctx, connState, &req)
		if resp != nil {
			if err := wsjson.Write(ctx, conn, resp); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}

type ConnState struct {
	conn          *websocket.Conn
	authenticated bool
	sessionID     string
}
```

### 1.5 JSON-RPC 処理（ws/rpc.go）

```go
package ws

import (
	"context"
	"crypto/subtle"
	"encoding/json"
)

// JSON-RPC 2.0 構造体
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// エラーコード
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeAuthFailed     = -32001
	ErrCodeUnauthorized   = -32002
)

func (h *Handler) handleRequest(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	// 認証チェック（auth メソッド以外）
	if req.Method != "auth" && !state.authenticated {
		return errorResponse(req.ID, ErrCodeUnauthorized, "Not authenticated")
	}

	switch req.Method {
	case "auth":
		return h.handleAuth(ctx, state, req)
	case "chat.attach":
		return h.handleChatAttach(ctx, state, req)
	case "chat.message":
		return h.handleChatMessage(ctx, state, req)
	case "chat.interrupt":
		return h.handleChatInterrupt(ctx, state, req)
	case "session.list":
		return h.handleSessionList(ctx, state, req)
	case "session.create":
		return h.handleSessionCreate(ctx, state, req)
	default:
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found")
	}
}

// 認証ハンドラ
func (h *Handler) handleAuth(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	// タイミング攻撃対策
	if subtle.ConstantTimeCompare([]byte(params.Token), []byte(h.authToken)) != 1 {
		return errorResponse(req.ID, ErrCodeAuthFailed, "Invalid token")
	}

	state.authenticated = true
	return successResponse(req.ID, map[string]bool{"success": true})
}

// ヘルパー関数
func successResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

func errorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}
```

### 1.6 依存関係インストール

```bash
cd server
go get github.com/coder/websocket
go get github.com/google/uuid
```

---

## Phase 2: Claude CLI 統合

### 2.1 Agent インターフェース（agent/agent.go）

```go
package agent

import "context"

// イベントタイプ
type EventType string

const (
	EventTypeText             EventType = "text"
	EventTypeToolCall         EventType = "tool_call"
	EventTypeToolResult       EventType = "tool_result"
	EventTypeError            EventType = "error"
	EventTypeDone             EventType = "done"
	EventTypePermissionRequest EventType = "permission_request"
	EventTypeAskUserQuestion  EventType = "ask_user_question"
	EventTypeSystem           EventType = "system"
)

// イベント構造体
type Event struct {
	Type             EventType              `json:"type"`
	Content          string                 `json:"content,omitempty"`
	ToolUseID        string                 `json:"tool_use_id,omitempty"`
	ToolName         string                 `json:"tool_name,omitempty"`
	ToolInput        map[string]interface{} `json:"tool_input,omitempty"`
	ToolOutput       string                 `json:"tool_output,omitempty"`
	PermissionID     string                 `json:"permission_id,omitempty"`
	QuestionID       string                 `json:"question_id,omitempty"`
	Question         string                 `json:"question,omitempty"`
	Options          []string               `json:"options,omitempty"`
}

// Agent インターフェース
type Agent interface {
	// メッセージ送信、イベントをチャンネルで返す
	SendMessage(ctx context.Context, message string) (<-chan Event, error)

	// 処理中断
	Interrupt(ctx context.Context) error

	// 権限リクエストへの応答
	RespondToPermission(ctx context.Context, permissionID string, allowed bool) error

	// 質問への応答
	RespondToQuestion(ctx context.Context, questionID string, answer string) error

	// プロセス終了
	Close() error
}
```

### 2.2 Claude CLI 実装（agent/claude/claude.go）

```go
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/yourname/my-devport/server/agent"
)

type Claude struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	sessionID string
	workDir   string
	mu        sync.Mutex
	cancel    context.CancelFunc
}

func New(sessionID, workDir string) (*Claude, error) {
	return &Claude{
		sessionID: sessionID,
		workDir:   workDir,
	}, nil
}

func (c *Claude) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.cmd = exec.CommandContext(ctx, "claude",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
		"--session-id", c.sessionID,
	)
	c.cmd.Dir = c.workDir

	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// stderr をログに出力
	go func() {
		scanner := bufio.NewScanner(c.stderr)
		for scanner.Scan() {
			// ログに記録（デバッグ用）
		}
	}()

	return nil
}

func (c *Claude) SendMessage(ctx context.Context, message string) (<-chan agent.Event, error) {
	events := make(chan agent.Event, 100)

	// メッセージを stdin に書き込み
	input := map[string]interface{}{
		"type":    "user_message",
		"content": message,
	}
	data, _ := json.Marshal(input)

	c.mu.Lock()
	_, err := c.stdin.Write(append(data, '\n'))
	c.mu.Unlock()

	if err != nil {
		close(events)
		return nil, fmt.Errorf("write stdin: %w", err)
	}

	// stdout からイベントを読み取り
	go c.readEvents(ctx, events)

	return events, nil
}

func (c *Claude) readEvents(ctx context.Context, events chan<- agent.Event) {
	defer close(events)

	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB バッファ

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		event := c.parseEvent(line)
		if event != nil {
			events <- *event
			if event.Type == agent.EventTypeDone {
				return
			}
		}
	}
}

func (c *Claude) parseEvent(data []byte) *agent.Event {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return &agent.Event{Type: agent.EventTypeText, Content: string(data)}
	}

	eventType, _ := raw["type"].(string)

	switch eventType {
	case "assistant":
		// テキスト出力
		if msg, ok := raw["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].([]interface{}); ok {
				for _, c := range content {
					if block, ok := c.(map[string]interface{}); ok {
						if block["type"] == "text" {
							return &agent.Event{
								Type:    agent.EventTypeText,
								Content: block["text"].(string),
							}
						}
					}
				}
			}
		}

	case "content_block_start":
		if cb, ok := raw["content_block"].(map[string]interface{}); ok {
			if cb["type"] == "tool_use" {
				return &agent.Event{
					Type:      agent.EventTypeToolCall,
					ToolUseID: cb["id"].(string),
					ToolName:  cb["name"].(string),
				}
			}
		}

	case "tool_result":
		return &agent.Event{
			Type:       agent.EventTypeToolResult,
			ToolUseID:  raw["tool_use_id"].(string),
			ToolOutput: raw["content"].(string),
		}

	case "result":
		return &agent.Event{Type: agent.EventTypeDone}

	case "permission_request":
		return &agent.Event{
			Type:         agent.EventTypePermissionRequest,
			PermissionID: raw["permission_id"].(string),
			ToolName:     raw["tool_name"].(string),
			Content:      raw["description"].(string),
		}

	case "ask_user_question":
		options := []string{}
		if opts, ok := raw["options"].([]interface{}); ok {
			for _, o := range opts {
				options = append(options, o.(string))
			}
		}
		return &agent.Event{
			Type:       agent.EventTypeAskUserQuestion,
			QuestionID: raw["question_id"].(string),
			Question:   raw["question"].(string),
			Options:    options,
		}
	}

	return nil
}

func (c *Claude) Interrupt(ctx context.Context) error {
	input := map[string]string{"type": "interrupt"}
	data, _ := json.Marshal(input)

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.stdin.Write(append(data, '\n'))
	return err
}

func (c *Claude) RespondToPermission(ctx context.Context, permissionID string, allowed bool) error {
	input := map[string]interface{}{
		"type":          "permission_response",
		"permission_id": permissionID,
		"allowed":       allowed,
	}
	data, _ := json.Marshal(input)

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.stdin.Write(append(data, '\n'))
	return err
}

func (c *Claude) RespondToQuestion(ctx context.Context, questionID, answer string) error {
	input := map[string]interface{}{
		"type":        "question_response",
		"question_id": questionID,
		"answer":      answer,
	}
	data, _ := json.Marshal(input)

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.stdin.Write(append(data, '\n'))
	return err
}

func (c *Claude) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}
```

### 2.3 チャットハンドラの実装（ws/rpc.go に追加）

```go
// ws/rpc.go に追加

import (
	"github.com/yourname/my-devport/server/agent/claude"
)

// セッションごとの Agent を保持
var agents = sync.Map{}

func (h *Handler) handleChatAttach(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	state.sessionID = params.SessionID

	// Agent がなければ作成
	if _, ok := agents.Load(params.SessionID); !ok {
		agent, err := claude.New(params.SessionID, ".")
		if err != nil {
			return errorResponse(req.ID, ErrCodeInternal, err.Error())
		}
		if err := agent.Start(context.Background()); err != nil {
			return errorResponse(req.ID, ErrCodeInternal, err.Error())
		}
		agents.Store(params.SessionID, agent)
	}

	return successResponse(req.ID, map[string]string{
		"session_id": params.SessionID,
		"status":     "attached",
	})
}

func (h *Handler) handleChatMessage(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	agentVal, ok := agents.Load(params.SessionID)
	if !ok {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Session not found")
	}
	agent := agentVal.(*claude.Claude)

	// メッセージ送信、イベントをストリーミング
	events, err := agent.SendMessage(ctx, params.Content)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	// イベントを非同期で送信
	go func() {
		for event := range events {
			h.sendNotification(state.conn, "chat."+string(event.Type), map[string]interface{}{
				"session_id": params.SessionID,
				"event":      event,
			})
		}
	}()

	return successResponse(req.ID, map[string]bool{"accepted": true})
}

func (h *Handler) handleChatInterrupt(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(req.Params, &params)

	if agentVal, ok := agents.Load(params.SessionID); ok {
		agent := agentVal.(*claude.Claude)
		agent.Interrupt(ctx)
	}

	return successResponse(req.ID, map[string]bool{"success": true})
}

// 通知送信（id なし）
func (h *Handler) sendNotification(conn *websocket.Conn, method string, params interface{}) {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	wsjson.Write(context.Background(), conn, notification)
}
```

---

## Phase 3: フロントエンド実装

### 3.1 プロジェクト初期化

```bash
cd my-devport
npm create vite@latest web -- --template react-ts
cd web
npm install
```

### 3.2 依存関係インストール

```bash
npm install zustand json-rpc-2.0 react-markdown
npm install -D tailwindcss postcss autoprefixer @tailwindcss/vite
npx tailwindcss init -p
```

### 3.3 ディレクトリ構造

```
web/src/
├── main.tsx
├── App.tsx
├── index.css
├── lib/
│   ├── wsStore.ts       # WebSocket + 状態管理
│   └── rpc.ts           # JSON-RPC クライアント
├── components/
│   ├── Auth.tsx         # 認証画面
│   ├── Chat/
│   │   ├── ChatPanel.tsx
│   │   ├── MessageList.tsx
│   │   └── InputArea.tsx
│   └── Session/
│       └── SessionList.tsx
└── hooks/
    └── useSession.ts
```

### 3.4 WebSocket + 状態管理（lib/wsStore.ts）

```typescript
import { create } from 'zustand';
import { JSONRPCClient } from 'json-rpc-2.0';

interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  toolCalls?: ToolCall[];
}

interface ToolCall {
  id: string;
  name: string;
  input?: Record<string, unknown>;
  output?: string;
}

interface WsState {
  // 接続状態
  connected: boolean;
  authenticated: boolean;

  // セッション
  currentSessionId: string | null;
  sessions: { id: string; title: string }[];

  // メッセージ
  messages: Message[];
  isGenerating: boolean;

  // アクション
  connect: (url: string, token: string) => Promise<void>;
  disconnect: () => void;
  sendMessage: (content: string) => Promise<void>;
  interrupt: () => Promise<void>;
  attachSession: (sessionId: string) => Promise<void>;
  createSession: () => Promise<string>;
}

export const useWsStore = create<WsState>((set, get) => {
  let ws: WebSocket | null = null;
  let rpcClient: JSONRPCClient | null = null;
  let currentAssistantMessage: Message | null = null;

  // 通知ハンドラ
  const handleNotification = (method: string, params: any) => {
    switch (method) {
      case 'chat.text':
        // テキストを追加
        if (!currentAssistantMessage) {
          currentAssistantMessage = {
            id: crypto.randomUUID(),
            role: 'assistant',
            content: '',
          };
          set(state => ({
            messages: [...state.messages, currentAssistantMessage!],
          }));
        }
        currentAssistantMessage.content += params.event.content;
        set(state => ({
          messages: state.messages.map(m =>
            m.id === currentAssistantMessage!.id
              ? { ...m, content: currentAssistantMessage!.content }
              : m
          ),
        }));
        break;

      case 'chat.tool_call':
        if (currentAssistantMessage) {
          const toolCall: ToolCall = {
            id: params.event.tool_use_id,
            name: params.event.tool_name,
            input: params.event.tool_input,
          };
          currentAssistantMessage.toolCalls = [
            ...(currentAssistantMessage.toolCalls || []),
            toolCall,
          ];
          set(state => ({
            messages: state.messages.map(m =>
              m.id === currentAssistantMessage!.id
                ? { ...m, toolCalls: currentAssistantMessage!.toolCalls }
                : m
            ),
          }));
        }
        break;

      case 'chat.done':
        currentAssistantMessage = null;
        set({ isGenerating: false });
        break;

      case 'chat.error':
        set({ isGenerating: false });
        break;
    }
  };

  return {
    connected: false,
    authenticated: false,
    currentSessionId: null,
    sessions: [],
    messages: [],
    isGenerating: false,

    connect: async (url: string, token: string) => {
      return new Promise((resolve, reject) => {
        ws = new WebSocket(url);

        ws.onopen = async () => {
          set({ connected: true });

          // JSON-RPC クライアント設定
          rpcClient = new JSONRPCClient((request) => {
            ws!.send(JSON.stringify(request));
            return Promise.resolve();
          });

          // 認証
          try {
            await rpcClient.request('auth', { token });
            set({ authenticated: true });
            resolve();
          } catch (e) {
            reject(e);
          }
        };

        ws.onmessage = (event) => {
          const data = JSON.parse(event.data);

          // 通知（id がない）
          if (!('id' in data) && data.method) {
            handleNotification(data.method, data.params);
            return;
          }

          // レスポンス
          rpcClient?.receive(data);
        };

        ws.onclose = () => {
          set({ connected: false, authenticated: false });
        };

        ws.onerror = (error) => {
          reject(error);
        };
      });
    },

    disconnect: () => {
      ws?.close();
      set({ connected: false, authenticated: false });
    },

    sendMessage: async (content: string) => {
      if (!rpcClient || !get().currentSessionId) return;

      // ユーザーメッセージを追加
      const userMessage: Message = {
        id: crypto.randomUUID(),
        role: 'user',
        content,
      };
      set(state => ({
        messages: [...state.messages, userMessage],
        isGenerating: true,
      }));

      await rpcClient.request('chat.message', {
        session_id: get().currentSessionId,
        content,
      });
    },

    interrupt: async () => {
      if (!rpcClient || !get().currentSessionId) return;
      await rpcClient.request('chat.interrupt', {
        session_id: get().currentSessionId,
      });
    },

    attachSession: async (sessionId: string) => {
      if (!rpcClient) return;
      await rpcClient.request('chat.attach', { session_id: sessionId });
      set({ currentSessionId: sessionId, messages: [] });
    },

    createSession: async () => {
      if (!rpcClient) return '';
      const result = await rpcClient.request('session.create', {});
      const sessionId = (result as any).session.id;
      set(state => ({
        sessions: [...state.sessions, { id: sessionId, title: 'New Chat' }],
      }));
      return sessionId;
    },
  };
});
```

### 3.5 チャットコンポーネント（components/Chat/ChatPanel.tsx）

```tsx
import { useState, useRef, useEffect } from 'react';
import { useWsStore } from '../../lib/wsStore';
import ReactMarkdown from 'react-markdown';

export function ChatPanel() {
  const {
    messages,
    isGenerating,
    sendMessage,
    interrupt,
    currentSessionId,
  } = useWsStore();

  const [input, setInput] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isGenerating) return;

    const content = input;
    setInput('');
    await sendMessage(content);
  };

  if (!currentSessionId) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500">
        セッションを選択してください
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* メッセージ一覧 */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.map((message) => (
          <div
            key={message.id}
            className={`flex ${
              message.role === 'user' ? 'justify-end' : 'justify-start'
            }`}
          >
            <div
              className={`max-w-[80%] rounded-lg p-3 ${
                message.role === 'user'
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-100 text-gray-900'
              }`}
            >
              <ReactMarkdown>{message.content}</ReactMarkdown>

              {/* ツール呼び出し表示 */}
              {message.toolCalls?.map((tool) => (
                <div
                  key={tool.id}
                  className="mt-2 p-2 bg-gray-200 rounded text-sm"
                >
                  <div className="font-mono text-xs text-gray-600">
                    {tool.name}
                  </div>
                  {tool.output && (
                    <pre className="mt-1 text-xs overflow-x-auto">
                      {tool.output}
                    </pre>
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* 入力エリア */}
      <form onSubmit={handleSubmit} className="p-4 border-t">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="メッセージを入力..."
            className="flex-1 px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            disabled={isGenerating}
          />
          {isGenerating ? (
            <button
              type="button"
              onClick={interrupt}
              className="px-4 py-2 bg-red-500 text-white rounded-lg"
            >
              停止
            </button>
          ) : (
            <button
              type="submit"
              className="px-4 py-2 bg-blue-500 text-white rounded-lg disabled:opacity-50"
              disabled={!input.trim()}
            >
              送信
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
```

### 3.6 認証画面（components/Auth.tsx）

```tsx
import { useState } from 'react';
import { useWsStore } from '../lib/wsStore';

export function Auth() {
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { connect } = useWsStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const wsUrl = `ws://${window.location.hostname}:9870/ws`;
      await connect(wsUrl, token);
    } catch (e) {
      setError('認証に失敗しました');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <form
        onSubmit={handleSubmit}
        className="bg-white p-8 rounded-lg shadow-md w-96"
      >
        <h1 className="text-2xl font-bold mb-6 text-center">Devport</h1>

        {error && (
          <div className="mb-4 p-3 bg-red-100 text-red-700 rounded">
            {error}
          </div>
        )}

        <input
          type="password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="認証トークン"
          className="w-full px-4 py-2 border rounded-lg mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />

        <button
          type="submit"
          disabled={loading || !token}
          className="w-full py-2 bg-blue-500 text-white rounded-lg disabled:opacity-50"
        >
          {loading ? '接続中...' : '接続'}
        </button>
      </form>
    </div>
  );
}
```

### 3.7 App.tsx

```tsx
import { useWsStore } from './lib/wsStore';
import { Auth } from './components/Auth';
import { ChatPanel } from './components/Chat/ChatPanel';
import { useEffect } from 'react';

function App() {
  const { authenticated, currentSessionId, createSession, attachSession } =
    useWsStore();

  useEffect(() => {
    // 認証後、自動でセッション作成
    if (authenticated && !currentSessionId) {
      createSession().then((id) => {
        if (id) attachSession(id);
      });
    }
  }, [authenticated, currentSessionId]);

  if (!authenticated) {
    return <Auth />;
  }

  return (
    <div className="h-screen flex flex-col">
      <header className="h-12 bg-gray-800 text-white flex items-center px-4">
        <h1 className="font-bold">Devport</h1>
      </header>
      <main className="flex-1 overflow-hidden">
        <ChatPanel />
      </main>
    </div>
  );
}

export default App;
```

---

## Phase 4: セッション管理

### 4.1 セッションストア（server/session/store.go）

```go
package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	sessions sync.Map
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Create(title string) *Session {
	session := &Session{
		ID:        uuid.New().String(),
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.sessions.Store(session.ID, session)
	return session
}

func (s *Store) Get(id string) *Session {
	if val, ok := s.sessions.Load(id); ok {
		return val.(*Session)
	}
	return nil
}

func (s *Store) List() []*Session {
	var sessions []*Session
	s.sessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
	})
	return sessions
}

func (s *Store) Delete(id string) {
	s.sessions.Delete(id)
}

func (s *Store) UpdateTitle(id, title string) {
	if val, ok := s.sessions.Load(id); ok {
		session := val.(*Session)
		session.Title = title
		session.UpdatedAt = time.Now()
	}
}
```

### 4.2 セッション RPC ハンドラ

```go
// ws/rpc.go に追加

var sessionStore = session.NewStore()

func (h *Handler) handleSessionList(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	sessions := sessionStore.List()
	return successResponse(req.ID, map[string]interface{}{
		"sessions": sessions,
	})
}

func (h *Handler) handleSessionCreate(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Title string `json:"title"`
	}
	json.Unmarshal(req.Params, &params)

	if params.Title == "" {
		params.Title = "New Chat"
	}

	session := sessionStore.Create(params.Title)
	return successResponse(req.ID, map[string]interface{}{
		"session": session,
	})
}
```

---

## Phase 5: ファイル・Git 操作

### 5.1 ファイル操作 RPC

```go
// ws/rpc_file.go

package ws

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func (h *Handler) handleFileGet(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	// パストラバーサル対策
	cleanPath := filepath.Clean(params.Path)
	if strings.HasPrefix(cleanPath, "..") {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid path")
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	return successResponse(req.ID, map[string]interface{}{
		"path":    cleanPath,
		"content": string(content),
	})
}
```

### 5.2 Git 操作 RPC

```go
// ws/rpc_git.go

package ws

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

func (h *Handler) handleGitStatus(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	var staged, modified, untracked []string
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])

		switch {
		case status[0] != ' ' && status[0] != '?':
			staged = append(staged, file)
		case status[1] == 'M':
			modified = append(modified, file)
		case status[0] == '?':
			untracked = append(untracked, file)
		}
	}

	// ブランチ名取得
	branchCmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	branchOutput, _ := branchCmd.Output()
	branch := strings.TrimSpace(string(branchOutput))

	return successResponse(req.ID, map[string]interface{}{
		"branch":    branch,
		"staged":    staged,
		"modified":  modified,
		"untracked": untracked,
	})
}

func (h *Handler) handleGitDiff(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(req.Params, &params)

	args := []string{"diff"}
	if params.Path != "" {
		args = append(args, "--", params.Path)
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	return successResponse(req.ID, map[string]interface{}{
		"diff": string(output),
	})
}
```

---

## Phase 6: リモートアクセス

外出先からアクセスするための設定。[リレーサーバー詳細](./relay-server.md) を参照。

### 6.1 Tailscale（推奨）

```bash
# 両端末に Tailscale をインストール
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up

# Devport 起動（リレー無効）
RELAY_ENABLED=false AUTH_TOKEN=your_password ./server
```

モバイルから `http://<tailscale-ip>:9870` でアクセス。

### 6.2 Cloudflare Tunnel

```bash
# トンネル作成
cloudflared tunnel create my-devport
cloudflared tunnel route dns my-devport devport.example.com

# config.yml 作成
cat > ~/.cloudflared/config.yml << EOF
tunnel: <TUNNEL_ID>
credentials-file: ~/.cloudflared/<TUNNEL_ID>.json
ingress:
  - hostname: devport.example.com
    service: http://localhost:9870
  - service: http_status:404
EOF

# 起動
cloudflared tunnel run my-devport
```

---

## Phase 7: iOS/Android アプリ化

### 7.1 iOS（SwiftUI）

```swift
// WebSocketManager.swift
import Foundation

class WebSocketManager: ObservableObject {
    private var webSocket: URLSessionWebSocketTask?
    @Published var messages: [ChatMessage] = []
    @Published var isConnected = false

    func connect(url: URL, token: String) {
        webSocket = URLSession.shared.webSocketTask(with: url)
        webSocket?.resume()

        // 認証
        let authRequest: [String: Any] = [
            "jsonrpc": "2.0",
            "method": "auth",
            "params": ["token": token],
            "id": 1
        ]
        send(authRequest)
        receiveMessages()
    }

    private func send(_ object: [String: Any]) {
        guard let data = try? JSONSerialization.data(withJSONObject: object) else { return }
        webSocket?.send(.data(data)) { _ in }
    }

    private func receiveMessages() {
        webSocket?.receive { [weak self] result in
            switch result {
            case .success(let message):
                if case .data(let data) = message {
                    self?.handleMessage(data)
                }
                self?.receiveMessages()
            case .failure:
                self?.isConnected = false
            }
        }
    }

    private func handleMessage(_ data: Data) {
        // JSON-RPC レスポンス/通知を処理
    }

    func sendMessage(_ content: String, sessionId: String) {
        let request: [String: Any] = [
            "jsonrpc": "2.0",
            "method": "chat.message",
            "params": ["session_id": sessionId, "content": content],
            "id": Int.random(in: 1...10000)
        ]
        send(request)
    }
}
```

### 7.2 Android（Kotlin + Compose）

```kotlin
// WebSocketManager.kt
class WebSocketManager {
    private val client = OkHttpClient()
    private var webSocket: WebSocket? = null

    val messages = MutableStateFlow<List<ChatMessage>>(emptyList())
    val isConnected = MutableStateFlow(false)

    fun connect(url: String, token: String) {
        val request = Request.Builder().url(url).build()
        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                isConnected.value = true
                // 認証
                val auth = JSONObject().apply {
                    put("jsonrpc", "2.0")
                    put("method", "auth")
                    put("params", JSONObject().put("token", token))
                    put("id", 1)
                }
                webSocket.send(auth.toString())
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                isConnected.value = false
            }
        })
    }

    fun sendMessage(content: String, sessionId: String) {
        val request = JSONObject().apply {
            put("jsonrpc", "2.0")
            put("method", "chat.message")
            put("params", JSONObject().apply {
                put("session_id", sessionId)
                put("content", content)
            })
            put("id", (1..10000).random())
        }
        webSocket?.send(request.toString())
    }
}
```

---

## チェックリスト

### MVP（最小構成）

- [ ] Go サーバー起動
- [ ] WebSocket 接続
- [ ] トークン認証
- [ ] Claude CLI 起動・通信
- [ ] メッセージ送受信
- [ ] ストリーミング表示
- [ ] React フロントエンド

### フル機能

- [ ] セッション管理（作成、切り替え、削除）
- [ ] ファイル閲覧
- [ ] Git status/diff 表示
- [ ] 権限リクエスト UI
- [ ] ユーザー質問 UI
- [ ] 処理中断
- [ ] リモートアクセス（Tailscale/Cloudflare）
- [ ] iOS/Android アプリ

---

## 参考リソース

- [Devport リポジトリ](https://github.com/sijiaoh/devport) - 実装の参考
- [Claude CLI ドキュメント](https://docs.anthropic.com/claude-code) - stream-json 形式
- [JSON-RPC 2.0 仕様](https://www.jsonrpc.org/specification)
- [coder/websocket](https://github.com/coder/websocket) - Go WebSocket ライブラリ
