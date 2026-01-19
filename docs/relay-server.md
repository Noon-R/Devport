# リレーサーバー詳細

## 概要

Devport は NAT 越えのために **cloud.devport.com** をリレーサーバーとして使用する。これにより、外出先のモバイル端末から自宅/オフィスの PC 上で動作する Claude CLI にアクセスできる。

## アーキテクチャ

```
┌──────────────────┐                      ┌──────────────────────────┐
│  モバイル端末      │                      │   cloud.devport.com      │
│                  │                      │   (リレーサーバー)         │
│  ┌────────────┐  │    WebSocket         │  ┌────────────────────┐  │
│  │ React SPA  │  │◄───────────────────►│  │   Multiplexer      │  │
│  └────────────┘  │  wss://{sub}.cloud   │  │   (接続多重化)       │  │
│                  │   .devport.com/ws    │  └─────────┬──────────┘  │
└──────────────────┘                      │            │             │
                                          │            │ WebSocket   │
                                          │            │ (常時接続)   │
                                          │            ▼             │
                                          │  ┌────────────────────┐  │
                                          │  │  Connection Pool   │  │
                                          │  └────────────────────┘  │
                                          └──────────────────────────┘
                                                       │
                                                       │ WebSocket
                                                       │ wss://{sub}.cloud
                                                       │  .devport.com/relay
                                                       ▼
                                          ┌──────────────────────────┐
                                          │   ローカル PC (NAT 内)     │
                                          │  ┌────────────────────┐  │
                                          │  │   Relay Manager    │  │
                                          │  │   (リレー管理)       │  │
                                          │  └─────────┬──────────┘  │
                                          │            │             │
                                          │            ▼             │
                                          │  ┌────────────────────┐  │
                                          │  │   Go Server        │  │
                                          │  │   + Claude CLI     │  │
                                          │  └────────────────────┘  │
                                          └──────────────────────────┘
```

## 接続フロー

### 1. 登録（初回起動時）

```
ローカル PC                           cloud.devport.com
    │                                        │
    │── POST /api/relay/register ──────────►│
    │   { client_version: "1.0.0" }          │
    │                                        │
    │◄── 201 Created ──────────────────────│
    │   {                                    │
    │     subdomain: "abc123",               │
    │     relay_token: "rt_xxx...",          │
    │     relay_server: "cloud.devport.com"  │
    │   }                                    │
    │                                        │
```

- サブドメインとトークンは `~/.devport/relay/` に保存
- 次回以降は保存された設定を使用

### 2. リフレッシュ（2回目以降）

```
ローカル PC                           cloud.devport.com
    │                                        │
    │── POST /api/relay/refresh ───────────►│
    │   { relay_token: "rt_xxx..." }         │
    │                                        │
    │◄── 200 OK ───────────────────────────│
    │   { subdomain: "abc123", ... }         │
    │                                        │
```

### 3. WebSocket 接続確立

```
ローカル PC                           cloud.devport.com
    │                                        │
    │── WebSocket Connect ─────────────────►│
    │   wss://abc123.cloud.devport.com/relay │
    │                                        │
    │── JSON-RPC: register ────────────────►│
    │   { relay_token: "rt_xxx..." }         │
    │                                        │
    │◄── result: { status: "ok" } ─────────│
    │                                        │
    │◄─────── 常時接続維持 ─────────────────│
    │                                        │
```

### 4. クライアント接続時

```
モバイル端末                  cloud.devport.com           ローカル PC
    │                              │                          │
    │── WebSocket Connect ────────►│                          │
    │   wss://abc123.cloud         │                          │
    │    .devport.com/ws           │                          │
    │                              │                          │
    │── JSON-RPC message ─────────►│── Envelope ─────────────►│
    │                              │   { connection_id: "c1", │
    │                              │     type: "message",     │
    │                              │     payload: {...} }     │
    │                              │                          │
    │◄── notification ────────────│◄── Envelope ─────────────│
    │                              │   { connection_id: "c1", │
    │                              │     type: "message",     │
    │                              │     payload: {...} }     │
    │                              │                          │
```

## 通信プロトコル

### Envelope 形式

リレーサーバーとローカル PC 間の通信は **Envelope** でラップされる:

```go
type Envelope struct {
    ConnectionID string          `json:"connection_id"`
    Type         EnvelopeType    `json:"type"`
    Payload      json.RawMessage `json:"payload,omitempty"`
    HTTPRequest  *HTTPRequest    `json:"http_request,omitempty"`
    HTTPResponse *HTTPResponse   `json:"http_response,omitempty"`
}
```

### EnvelopeType

| タイプ | 説明 |
|--------|------|
| `message` | JSON-RPC メッセージ |
| `disconnected` | クライアント切断通知 |
| `http_request` | HTTP リクエスト（静的ファイル等） |
| `http_response` | HTTP レスポンス |

### Multiplexer の役割

1. **接続の多重化**: 単一の WebSocket 上で複数クライアント接続を管理
2. **VirtualStream**: クライアントごとに仮想ストリームを作成
3. **メッセージルーティング**: `connection_id` でメッセージを振り分け
4. **HTTP プロキシ**: 静的ファイルリクエストもリレー経由で処理

## 再接続ロジック

```
接続失敗時:
  ├─ 1秒待機 → 再接続試行
  ├─ 2秒待機 → 再接続試行
  ├─ 4秒待機 → 再接続試行
  ├─ 8秒待機 → 再接続試行
  └─ 10秒待機 → 再接続試行（上限）

接続が1分以上安定した場合:
  └─ バックオフをリセット
```

## エラーハンドリング

### ErrUpgradeRequired

クライアントバージョンが古い場合:

```
Error: client version too old, please upgrade Devport
```

**解決策**: Devport を最新版に更新

### ErrInvalidToken

トークンが無効な場合:

```
Error: invalid relay token
```

**解決策**: `~/.devport/relay/` の設定を削除して再登録

## QR コード接続

起動時にターミナルに QR コードが表示される:

```
╭─────────────────────────────────────╮
│  Devport is running                 │
│                                     │
│  Local:  http://localhost:9870      │
│  Remote: https://abc123.cloud       │
│          .devport.com               │
│                                     │
│  █████████████████████████████████  │
│  █████████████████████████████████  │
│  ████ ▄▄▄▄▄ █ ▄█ █▄█ █ ▄▄▄▄▄ ████  │
│  ████ █   █ █ ▄▄▄█▄  █ █   █ ████  │
│  ...                                │
│                                     │
╰─────────────────────────────────────╯
```

モバイル端末でスキャンすると自動的にリモート URL に接続。

## 自前リレーサーバーの構築

cloud.devport.com を使わずに自前でリレーサーバーを構築する場合:

### 必要な機能

1. **登録 API**: `/api/relay/register`
2. **リフレッシュ API**: `/api/relay/refresh`
3. **WebSocket エンドポイント**: `/relay`（ローカル PC 用）
4. **WebSocket エンドポイント**: `/ws`（クライアント用）
5. **Multiplexer**: 接続の多重化
6. **サブドメイン管理**: 動的サブドメイン割り当て

### 代替案

| 方法 | メリット | デメリット |
|------|---------|-----------|
| **ngrok** | 簡単、無料枠あり | サブドメインが変わる |
| **Cloudflare Tunnel** | 無料、安定 | 設定がやや複雑 |
| **自前 VPS + WireGuard** | 完全制御 | 運用コスト |
| **Tailscale** | 簡単、安全 | Tailscale ネットワーク内のみ |

### 環境変数での切り替え

```bash
# 自前リレーサーバーを使用
CLOUD_URL=https://relay.example.com ./devport
```

## セキュリティ考慮事項

### ⚠️ 重要: リレーサーバー経由の通信リスク

**cloud.devport.com を使用する場合、以下のデータがリレーサーバーを通過する:**

```
モバイル ──WSS(暗号化)──► cloud.devport.com ──WSS(暗号化)──► ローカルPC
                              │
                              │ ← リレーサーバーで復号される
                              │
                         【通過するデータ】
                         ・チャットメッセージ（プロンプト）
                         ・Claude CLIの出力（AIの応答）
                         ・ファイル内容（ソースコード）
                         ・Git diff
                         ・AUTH_TOKEN（認証トークン）
```

### 暗号化の実態

| 区間 | 暗号化 | 説明 |
|------|:------:|------|
| モバイル → リレー | ✅ TLS | 経路は暗号化 |
| リレー内部 | ❌ 平文 | **リレー運営者は閲覧可能** |
| リレー → ローカルPC | ✅ TLS | 経路は暗号化 |

**End-to-End 暗号化ではない** ため、リレーサーバーの運営者は理論上すべての通信内容を見ることができる。

### 用途別の推奨設定

| 用途 | 推奨方法 | セキュリティ |
|------|---------|:-----------:|
| 機密性の高い業務コード | Tailscale / Cloudflare Tunnel | ★★★ |
| 個人の趣味プロジェクト | cloud.devport.com でも可 | ★★☆ |
| 学習・実験用 | cloud.devport.com | ★☆☆ |
| 社内ネットワーク | 自前リレーサーバー | ★★★ |

### 基本的なセキュリティ対策

1. **トークン管理**: `relay_token` は機密情報として扱う
2. **HTTPS/WSS**: 本番環境では必ず暗号化通信を使用
3. **認証の多層化**: リレー認証 + アプリ認証の2段階
4. **レート制限**: cloud.devport.com にはレート制限あり

---

## 自前リレーサーバーの構築（詳細）

cloud.devport.com を使わず、自分でリレーサーバーを構築する方法。

### 方法1: Tailscale（最も簡単・安全）

Tailscale は WireGuard ベースの VPN サービス。リレーサーバー不要で直接接続できる。

**セットアップ:**

```bash
# ローカル PC にインストール
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up

# モバイル端末にも Tailscale アプリをインストール
# 同じアカウントでログイン
```

**Devport の起動:**

```bash
# リレーを無効化してローカルのみで起動
RELAY_ENABLED=false AUTH_TOKEN=your_password ./devport
```

**接続:**
- モバイルから `http://<tailscale-ip>:9870` でアクセス
- Tailscale ネットワーク内は暗号化済み

**メリット:**
- 第三者サーバーを経由しない
- WireGuard による強力な暗号化
- 設定が簡単

**デメリット:**
- 両端末に Tailscale のインストールが必要

---

### 方法2: Cloudflare Tunnel（無料・安定）

Cloudflare の Zero Trust ネットワークを利用。カスタムドメインで安定したアクセスが可能。

**セットアップ:**

```bash
# cloudflared のインストール
# macOS
brew install cloudflared

# Linux
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o cloudflared
chmod +x cloudflared

# Windows
# https://github.com/cloudflare/cloudflared/releases からダウンロード
```

**トンネルの作成:**

```bash
# Cloudflare にログイン
cloudflared tunnel login

# トンネル作成
cloudflared tunnel create devport

# DNS 設定（要: Cloudflare で管理しているドメイン）
cloudflared tunnel route dns devport devport.example.com
```

**設定ファイル作成:**

```yaml
# ~/.cloudflared/config.yml
tunnel: <TUNNEL_ID>
credentials-file: ~/.cloudflared/<TUNNEL_ID>.json

ingress:
  - hostname: devport.example.com
    service: http://localhost:9870
  - service: http_status:404
```

**起動:**

```bash
# Devport（リレー無効）
RELAY_ENABLED=false AUTH_TOKEN=your_password ./devport

# Cloudflare Tunnel
cloudflared tunnel run devport
```

**systemd でサービス化:**

```bash
cloudflared service install
systemctl enable cloudflared
systemctl start cloudflared
```

**メリット:**
- 無料
- Cloudflare のグローバルネットワークで高速
- DDoS 保護付き

**デメリット:**
- Cloudflare でドメインを管理する必要がある
- 初期設定がやや複雑

---

### 方法3: ngrok（最も手軽）

一時的な公開 URL を即座に取得できる。

**セットアップ:**

```bash
# インストール
# macOS
brew install ngrok

# その他
# https://ngrok.com/download からダウンロード

# アカウント登録後、認証トークンを設定
ngrok config add-authtoken <YOUR_TOKEN>
```

**起動:**

```bash
# Devport
RELAY_ENABLED=false AUTH_TOKEN=your_password ./devport

# ngrok（別ターミナル）
ngrok http 9870
```

**出力例:**

```
Session Status                online
Forwarding                    https://abc123.ngrok.io -> http://localhost:9870
```

**メリット:**
- 最も簡単（1コマンドで公開）
- 無料枠あり

**デメリット:**
- 無料版は URL が毎回変わる
- 有料プランでないと固定ドメインが使えない

---

### 方法4: 自前 VPS でリレーサーバー構築

完全に自分で管理したい場合。Devport 互換のリレーサーバーを実装する。

**必要な API エンドポイント:**

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/api/relay/register` | POST | 新規登録、サブドメイン発行 |
| `/api/relay/refresh` | POST | トークンリフレッシュ |
| `/relay` | WebSocket | ローカル PC からの常時接続 |
| `/ws` | WebSocket | クライアントからの接続 |

**リレーサーバーの最小実装（Go）:**

```go
package main

import (
    "encoding/json"
    "net/http"
    "sync"

    "github.com/coder/websocket"
    "github.com/google/uuid"
)

type RelayServer struct {
    connections sync.Map // subdomain -> *websocket.Conn
}

func (s *RelayServer) handleRegister(w http.ResponseWriter, r *http.Request) {
    subdomain := uuid.New().String()[:8]
    token := uuid.New().String()

    json.NewEncoder(w).Encode(map[string]string{
        "subdomain":    subdomain,
        "relay_token":  token,
        "relay_server": "relay.example.com",
    })
}

func (s *RelayServer) handleRelay(w http.ResponseWriter, r *http.Request) {
    // ローカル PC からの WebSocket 接続
    conn, _ := websocket.Accept(w, r, nil)
    subdomain := extractSubdomain(r.Host)
    s.connections.Store(subdomain, conn)

    // メッセージをクライアントに転送
    for {
        _, data, err := conn.Read(r.Context())
        if err != nil {
            break
        }
        // クライアントへ転送...
    }
}

func (s *RelayServer) handleWS(w http.ResponseWriter, r *http.Request) {
    // クライアントからの WebSocket 接続
    conn, _ := websocket.Accept(w, r, nil)
    subdomain := extractSubdomain(r.Host)

    // ローカル PC への接続を取得
    if localConn, ok := s.connections.Load(subdomain); ok {
        // メッセージを双方向に転送...
    }
}

func main() {
    server := &RelayServer{}
    http.HandleFunc("/api/relay/register", server.handleRegister)
    http.HandleFunc("/relay", server.handleRelay)
    http.HandleFunc("/ws", server.handleWS)
    http.ListenAndServe(":8080", nil)
}
```

**ワイルドカード DNS 設定:**

```
*.relay.example.com  A  <VPS_IP>
```

**Nginx 設定（TLS 終端）:**

```nginx
server {
    listen 443 ssl;
    server_name *.relay.example.com;

    ssl_certificate /etc/letsencrypt/live/relay.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/relay.example.com/privkey.pem;

    location /relay {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }

    location / {
        proxy_pass http://localhost:8080;
    }
}
```

**Devport での使用:**

```bash
CLOUD_URL=https://relay.example.com AUTH_TOKEN=your_password ./devport
```

**メリット:**
- 完全な制御
- データが自分のサーバーのみを通過

**デメリット:**
- VPS の運用コスト（月額 $5〜）
- 実装・運用の手間

---

### 方法比較まとめ

| 方法 | コスト | 設定難易度 | セキュリティ | 安定性 |
|------|:------:|:----------:|:------------:|:------:|
| **Tailscale** | 無料 | ★☆☆ | ★★★ | ★★★ |
| **Cloudflare Tunnel** | 無料 | ★★☆ | ★★★ | ★★★ |
| **ngrok** | 無料〜 | ★☆☆ | ★★☆ | ★★☆ |
| **自前 VPS** | 有料 | ★★★ | ★★★ | ★★★ |
| **cloud.devport.com** | 無料 | ★☆☆ | ★☆☆ | ★★★ |

**推奨:**
- 個人利用: **Tailscale**（最も簡単で安全）
- チーム利用: **Cloudflare Tunnel**（カスタムドメイン、管理しやすい）
- 一時的な利用: **ngrok**（すぐ使える）
- 完全な制御が必要: **自前 VPS**
