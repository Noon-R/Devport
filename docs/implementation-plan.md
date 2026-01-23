# Devport 実装計画

## 概要

**目標**: AI 編集をメインとしたモバイル向けプログラミングプラットフォームの MVP 構築

**インフラ**: さくらの VPS 1台（リレーサーバー + 将来的な本番環境）

---

## インフラ構成

```
┌─────────────────────────────────────────────────────────────────────┐
│                        さくらの VPS                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Nginx (リバースプロキシ + TLS終端)                           │    │
│  │  - cloud.devport.app → リレーサーバー                        │    │
│  │  - *.cloud.devport.app → リレーサーバー                      │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                       │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  リレーサーバー (Go)                                          │    │
│  │  - サブドメイン発行                                           │    │
│  │  - WebSocket 接続の多重化                                     │    │
│  │  - ローカル PC とモバイルの中継                                │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
                              │ WebSocket (常時接続)
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        ローカル PC                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Devport Server (Go)                                         │    │
│  │  - WebSocket RPC                                             │    │
│  │  - Claude CLI 管理                                           │    │
│  │  - ファイル/Git 操作                                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## フェーズ別実装計画

### Phase 0: 環境構築（1-2日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 0.1 | リポジトリ作成 | GitHub に devport リポジトリを作成 |
| 0.2 | 開発環境構築 | Node.js 22, Go 1.25, Claude CLI インストール |
| 0.3 | プロジェクト初期化 | web/, server/, relay/ ディレクトリ構成 |
| 0.4 | VPS 初期設定 | さくら VPS の OS セットアップ、SSH 設定 |

#### ディレクトリ構成

```
devport/
├── web/                    # フロントエンド (React)
├── server/                 # バックエンド (Go)
├── relay/                  # リレーサーバー (Go)
├── site/                   # ランディングページ (Hugo)
├── docs/                   # ドキュメント
├── docker/                 # Docker 設定
│   ├── relay/
│   └── server/
├── scripts/                # デプロイスクリプト
└── README.md
```

#### VPS 初期設定

```bash
# さくら VPS にログイン
ssh root@<vps-ip>

# ユーザー作成
adduser devport
usermod -aG sudo devport

# SSH 鍵設定
mkdir -p /home/devport/.ssh
cp ~/.ssh/authorized_keys /home/devport/.ssh/
chown -R devport:devport /home/devport/.ssh

# 必要なパッケージ
apt update && apt upgrade -y
apt install -y nginx certbot python3-certbot-nginx docker.io docker-compose git

# Docker グループに追加
usermod -aG docker devport
```

---

### Phase 1: バックエンド基盤（3-5日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 1.1 | Go プロジェクト初期化 | go mod init, 依存関係追加 |
| 1.2 | HTTP サーバー実装 | net/http, ヘルスチェック |
| 1.3 | WebSocket ハンドラ | coder/websocket 使用 |
| 1.4 | JSON-RPC 処理 | リクエスト/レスポンス/通知 |
| 1.5 | トークン認証 | AUTH_TOKEN による認証 |
| 1.6 | 単体テスト | 認証、RPC ハンドラのテスト |

#### ファイル構成

```
server/
├── main.go
├── go.mod
├── go.sum
├── config/
│   └── config.go           # 環境変数読み込み
├── ws/
│   ├── handler.go          # WebSocket ハンドラ
│   ├── rpc.go              # JSON-RPC 基盤
│   ├── rpc_auth.go         # 認証
│   ├── rpc_chat.go         # チャット
│   ├── rpc_session.go      # セッション
│   ├── rpc_file.go         # ファイル操作
│   └── rpc_git.go          # Git 操作
├── agent/
│   ├── agent.go            # インターフェース
│   └── claude/
│       └── claude.go       # Claude CLI 実装
├── session/
│   └── store.go            # セッション管理
└── middleware/
    └── auth.go             # 認証ミドルウェア
```

#### 実装順序

```
1.1 → 1.2 → 1.3 → 1.4 → 1.5 → 1.6
```

---

### Phase 2: Claude CLI 統合（3-5日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 2.1 | Agent インターフェース定義 | イベント型、メソッド定義 |
| 2.2 | Claude CLI プロセス起動 | exec.Command, stdin/stdout パイプ |
| 2.3 | stream-json パース | イベント解析 |
| 2.4 | イベントストリーミング | チャンネルでイベント配信 |
| 2.5 | 中断処理 | Interrupt 実装 |
| 2.6 | 権限リクエスト処理 | permission_request/response |
| 2.7 | ユーザー質問処理 | ask_user_question/response |
| 2.8 | プロセス管理 | アイドルタイムアウト、参照カウント |

#### イベントフロー

```
クライアント                    サーバー                      Claude CLI
    │                              │                              │
    │─── chat.message ────────────►│                              │
    │                              │─── user_message ────────────►│
    │                              │                              │
    │                              │◄── assistant (text) ────────│
    │◄── chat.text ───────────────│                              │
    │                              │                              │
    │                              │◄── tool_use ────────────────│
    │◄── chat.tool_call ──────────│                              │
    │                              │                              │
    │                              │◄── tool_result ─────────────│
    │◄── chat.tool_result ────────│                              │
    │                              │                              │
    │                              │◄── result ──────────────────│
    │◄── chat.done ───────────────│                              │
```

---

### Phase 3: フロントエンド実装（5-7日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 3.1 | Vite + React + TS 初期化 | プロジェクトセットアップ |
| 3.2 | Tailwind CSS 設定 | スタイリング基盤 |
| 3.3 | Biome 設定 | Linter/Formatter |
| 3.4 | WebSocket 接続管理 | Zustand ストア |
| 3.5 | JSON-RPC クライアント | json-rpc-2.0 使用 |
| 3.6 | 認証画面 | トークン入力 UI |
| 3.7 | チャット UI | メッセージ一覧、入力エリア |
| 3.8 | ツール呼び出し表示 | 折りたたみ表示 |
| 3.9 | 権限リクエスト UI | 許可/拒否ダイアログ |
| 3.10 | ユーザー質問 UI | 選択肢表示 |
| 3.11 | セッション管理 UI | サイドバー |
| 3.12 | モバイル最適化 | レスポンシブデザイン |

#### コンポーネント構成

```
src/
├── main.tsx
├── App.tsx
├── index.css
├── lib/
│   ├── wsStore.ts          # WebSocket + 状態管理
│   └── rpc.ts              # JSON-RPC ヘルパー
├── components/
│   ├── Auth.tsx            # 認証画面
│   ├── Layout.tsx          # レイアウト
│   ├── Chat/
│   │   ├── ChatPanel.tsx   # チャットメイン
│   │   ├── MessageList.tsx # メッセージ一覧
│   │   ├── Message.tsx     # 単一メッセージ
│   │   ├── InputArea.tsx   # 入力エリア
│   │   ├── ToolCall.tsx    # ツール呼び出し表示
│   │   └── PermissionDialog.tsx
│   └── Session/
│       ├── SessionList.tsx # セッション一覧
│       └── SessionItem.tsx
├── hooks/
│   ├── useWebSocket.ts
│   └── useSession.ts
└── routes/
    ├── __root.tsx
    └── index.tsx
```

---

### Phase 4: セッション管理（2-3日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 4.1 | セッションストア実装 | メモリ + ファイル永続化 |
| 4.2 | セッション CRUD API | create, list, get, delete |
| 4.3 | セッション切り替え | attach/detach 処理 |
| 4.4 | 履歴保存 | .devport/sessions/ に JSON 保存 |
| 4.5 | 履歴復元 | セッション再開時に履歴読み込み |

#### データ構造

```go
// session/store.go
type Session struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    WorkDir   string    `json:"work_dir"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// .devport/sessions/{session_id}/
//   ├── meta.json      # セッションメタデータ
//   └── history.json   # メッセージ履歴
```

---

### Phase 5: リレーサーバー（5-7日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 5.1 | リレーサーバー基盤 | Go HTTP/WebSocket サーバー |
| 5.2 | 登録 API | POST /api/relay/register |
| 5.3 | リフレッシュ API | POST /api/relay/refresh |
| 5.4 | ローカル PC 接続 | /relay WebSocket エンドポイント |
| 5.5 | クライアント接続 | /ws WebSocket エンドポイント |
| 5.6 | Multiplexer 実装 | 接続の多重化 |
| 5.7 | Envelope プロトコル | メッセージラッピング |
| 5.8 | サブドメイン管理 | 動的サブドメイン割り当て |
| 5.9 | VPS デプロイ | Docker + Nginx |
| 5.10 | TLS 設定 | Let's Encrypt |
| 5.11 | ワイルドカード DNS | *.cloud.devport.app |

#### リレーサーバー構成

```
relay/
├── main.go
├── go.mod
├── config/
│   └── config.go
├── api/
│   ├── register.go         # 登録 API
│   └── refresh.go          # リフレッシュ API
├── ws/
│   ├── relay_handler.go    # ローカル PC 接続
│   ├── client_handler.go   # クライアント接続
│   └── multiplexer.go      # 接続多重化
├── store/
│   └── connection.go       # 接続管理
└── Dockerfile
```

#### Nginx 設定（VPS）

```nginx
# /etc/nginx/sites-available/devport-relay
server {
    listen 80;
    server_name cloud.devport.app *.cloud.devport.app;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name cloud.devport.app *.cloud.devport.app;

    ssl_certificate /etc/letsencrypt/live/cloud.devport.app/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cloud.devport.app/privkey.pem;

    location /relay {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

#### DNS 設定（ドメインレジストラ）

```
cloud.devport.app      A     <VPS_IP>
*.cloud.devport.app    A     <VPS_IP>
```

---

### Phase 6: ファイル・Git 操作（2-3日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 6.1 | ファイル取得 API | file.get |
| 6.2 | ディレクトリ一覧 API | file.list |
| 6.3 | Git status API | git.status |
| 6.4 | Git diff API | git.diff |
| 6.5 | ファイルブラウザ UI | ツリー表示 |
| 6.6 | Git diff ビューア UI | 差分表示 |

---

### Phase 7: 統合テスト・仕上げ（3-5日）

#### タスク

| # | タスク | 詳細 |
|---|--------|------|
| 7.1 | E2E テスト | 認証 → チャット → ツール呼び出し |
| 7.2 | エラーハンドリング | 接続断、タイムアウト |
| 7.3 | 再接続ロジック | 指数バックオフ |
| 7.4 | QR コード表示 | 起動時にリモート URL 表示 |
| 7.5 | ドキュメント更新 | README, 使い方ガイド |
| 7.6 | Docker イメージ作成 | server, relay |
| 7.7 | CI/CD 設定 | GitHub Actions |

---

## タイムライン

| フェーズ | 期間 | 累計 |
|---------|------|------|
| Phase 0: 環境構築 | 1-2日 | 2日 |
| Phase 1: バックエンド基盤 | 3-5日 | 7日 |
| Phase 2: Claude CLI 統合 | 3-5日 | 12日 |
| Phase 3: フロントエンド | 5-7日 | 19日 |
| Phase 4: セッション管理 | 2-3日 | 22日 |
| Phase 5: リレーサーバー | 5-7日 | 29日 |
| Phase 6: ファイル・Git | 2-3日 | 32日 |
| Phase 7: 統合テスト | 3-5日 | 37日 |

**合計: 約5-6週間**

---

## MVP スコープ

### 含める機能

- [x] トークン認証
- [x] チャット UI（メッセージ送受信）
- [x] ストリーミング表示
- [x] ツール呼び出し表示
- [x] 権限リクエスト UI
- [x] セッション管理（作成、一覧、切り替え）
- [x] リレーサーバー経由のリモートアクセス
- [x] QR コード接続
- [x] ファイルブラウザ
- [x] Git diff ビューア
- [x] セッション履歴永続化
- [x] 自動再接続（指数バックオフ）
- [x] REST API フォールバック
- [x] Docker 対応
- [x] CI/CD (GitHub Actions)
- [x] E2E テスト

### 後回しにする機能

- [ ] コードエディタ（インライン編集）
- [ ] iOS/Android ネイティブアプリ
- [ ] マルチ AI エージェント対応
- [ ] VPS デプロイ自動化

---

## VPS コスト見積もり

| 項目 | 仕様 | 月額 |
|------|------|------|
| さくら VPS | 1GB メモリ | ¥880 |
| ドメイン | devport.app | ¥1,500/年 ≈ ¥125/月 |
| **合計** | | **約 ¥1,000/月** |

※ 2GB メモリ（¥1,738/月）推奨（リレーサーバーの安定運用のため）

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| Claude CLI の仕様変更 | バージョン固定、変更検知テスト |
| VPS 障害 | ローカルのみモードを用意 |
| セキュリティ | TLS 必須、トークン認証、レート制限 |
| メモリリーク | プロセス監視、定期再起動 |

---

## 次のアクション

1. **今日**: GitHub リポジトリ作成、ディレクトリ構成
2. **明日**: VPS 初期設定、ドメイン取得
3. **今週**: Phase 1 完了（バックエンド基盤）
