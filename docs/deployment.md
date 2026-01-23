# デプロイガイド

## 概要

Devport のデプロイ方法は主に2つ:

1. **Docker**: 推奨。環境構築が簡単
2. **ネイティブビルド**: Claude CLI が既にインストールされている環境向け

## デプロイアーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────┐
│                          VPS (cloud.devport.app)                    │
│  ┌─────────────┐    ┌─────────────────────────────────────────┐    │
│  │   Caddy     │───▶│           Relay Server                  │    │
│  │ (HTTPS/WSS) │    │  - /api/relay/register                  │    │
│  │ Port 443    │    │  - /api/relay/refresh                   │    │
│  └─────────────┘    │  - /relay (Local PC WebSocket)          │    │
│        │            │  - /ws (Mobile Client WebSocket)         │    │
│        │            │  Port 8080                               │    │
│        ▼            └─────────────────────────────────────────┘    │
│   *.devport.app                                                     │
│   (ワイルドカード)                                                    │
└─────────────────────────────────────────────────────────────────────┘
         │
         │ WebSocket
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Local PC (ユーザー環境)                           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Devport Server                            │   │
│  │  - Relay Client (自動接続)                                    │   │
│  │  - Claude CLI Process                                        │   │
│  │  Port 9870                                                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## VPS へのリレーサーバーデプロイ

### 前提条件

- **VPS**: Ubuntu 22.04 LTS 推奨（2GB RAM 以上）
- **ドメイン**: `cloud.devport.app`（ワイルドカード DNS 設定済み）
- **DNS 設定**:
  - `A` レコード: `cloud.devport.app` → VPS IP
  - `A` レコード（ワイルドカード）: `*.cloud.devport.app` → VPS IP

### ステップ 1: VPS 初期設定

```bash
# パッケージ更新
sudo apt update && sudo apt upgrade -y

# Docker インストール
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Caddy インストール（リバースプロキシ + 自動 HTTPS）
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy

# ファイアウォール設定
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### ステップ 2: リレーサーバーのデプロイ

```bash
# デプロイディレクトリを作成
sudo mkdir -p /opt/devport-relay
cd /opt/devport-relay

# docker-compose.yml を作成
cat > docker-compose.yml << 'EOF'
services:
  relay:
    image: ghcr.io/noon-r/devport-relay:latest
    container_name: devport-relay
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    environment:
      - DOMAIN=cloud.devport.app
      - SERVER_HOST=0.0.0.0
      - DEV_MODE=false
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  relay-data:
EOF

# コンテナ起動
docker compose up -d

# ログ確認
docker compose logs -f
```

### ステップ 3: Caddy 設定（ワイルドカード HTTPS）

```bash
# Caddyfile を作成
sudo tee /etc/caddy/Caddyfile << 'EOF'
# ワイルドカード証明書の設定
*.cloud.devport.app, cloud.devport.app {
    # TLS 設定（Let's Encrypt DNS チャレンジ）
    tls {
        dns cloudflare {env.CF_API_TOKEN}
    }

    # WebSocket とHTTP のリバースプロキシ
    reverse_proxy localhost:8080 {
        # WebSocket サポート
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}

        # WebSocket タイムアウト設定
        transport http {
            keepalive 60s
            keepalive_idle_conns 10
        }
    }
}

# ヘルスチェックエンドポイント
cloud.devport.app {
    handle /health {
        reverse_proxy localhost:8080
    }
}
EOF

# Cloudflare API トークンを環境変数に設定
# (DNSプロバイダーに応じて変更)
sudo systemctl edit caddy
# [Service]
# Environment=CF_API_TOKEN=your_cloudflare_api_token

# Caddy を再起動
sudo systemctl restart caddy
sudo systemctl status caddy
```

**DNS プロバイダー別の設定:**

| プロバイダー | Caddy DNS プラグイン | 環境変数 |
|-------------|---------------------|----------|
| Cloudflare | `dns cloudflare` | `CF_API_TOKEN` |
| Route53 | `dns route53` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| Google Cloud | `dns googleclouddns` | `GCP_PROJECT` |

### ステップ 4: 動作確認

```bash
# ヘルスチェック
curl https://cloud.devport.app/health
# => {"status":"ok"}

# 登録テスト
curl -X POST https://cloud.devport.app/api/relay/register \
  -H "Content-Type: application/json" \
  -d '{"client_version": "1.0.0"}'
# => {"subdomain":"abc123","relay_token":"rt_...","relay_server":"cloud.devport.app"}

# WebSocket 接続テスト (wscat が必要)
npm i -g wscat
wscat -c wss://abc123.cloud.devport.app/ws
```

---

## Docker でのデプロイ

### 前提条件

- Docker 20.x 以上
- Docker Compose v2

### クイックスタート

```bash
# docker-compose.yml を作成
cat > docker-compose.yml << 'EOF'
services:
  devport:
    image: ghcr.io/sijiaoh/devport:latest
    ports:
      - "8080:8080"
    volumes:
      - workspace:/workspace
      - claude-config:/home/devport/.claude
    environment:
      - AUTH_TOKEN=your_secure_password
    restart: unless-stopped

volumes:
  workspace:
  claude-config:
EOF

# 起動
docker compose up -d
```

### 設定オプション

```yaml
services:
  devport:
    image: ghcr.io/sijiaoh/devport:latest
    ports:
      - "80:8080"                    # ポートマッピング
    volumes:
      - ./my-project:/workspace      # プロジェクトディレクトリ
      - claude-config:/home/devport/.claude
    environment:
      - AUTH_TOKEN=your_password     # 必須: 認証トークン
      - GIT_ENABLED=true             # Git 機能を有効化
      - REPOSITORY_URL=https://github.com/user/repo.git
      - REPOSITORY_TOKEN=ghp_xxx     # GitHub PAT
      - GIT_USER_NAME=Your Name
      - GIT_USER_EMAIL=you@example.com
      - RELAY_ENABLED=true           # リレー接続を有効化
    restart: unless-stopped
```

### ヘルスチェック

```bash
curl http://localhost:8080/health
```

レスポンス:
```json
{"status": "ok"}
```

### ログ確認

```bash
docker compose logs -f devport
```

---

## ネイティブビルドでのデプロイ

### 前提条件

- Go 1.25.x
- Node.js 22.x
- Claude CLI

### ビルド手順

```bash
# リポジトリをクローン
git clone https://github.com/sijiaoh/devport.git
cd devport

# フロントエンドをビルド
cd web
npm ci
npm run build

# バックエンドに静的ファイルをコピー
cp -r dist ../server/static

# バックエンドをビルド
cd ../server
go build -ldflags="-w -s" -o devport .
```

### 実行

```bash
AUTH_TOKEN=your_password ./devport
```

### systemd サービス化（Linux）

```ini
# /etc/systemd/system/devport.service
[Unit]
Description=Devport Server
After=network.target

[Service]
Type=simple
User=devport
WorkingDirectory=/opt/devport
Environment=AUTH_TOKEN=your_password
Environment=WORK_DIR=/home/devport/workspace
ExecStart=/opt/devport/devport
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable devport
sudo systemctl start devport
```

---

## 環境変数一覧

### 必須

| 変数 | 説明 |
|------|------|
| `AUTH_TOKEN` | API 認証トークン |

### サーバー設定

| 変数 | デフォルト | 説明 |
|------|-----------|------|
| `SERVER_PORT` | `8080` | HTTP サーバーポート |
| `WORK_DIR` | `/workspace` | 作業ディレクトリ |
| `DATA_DIR` | `.devport/` | Devport データ保存先 |
| `IDLE_TIMEOUT` | `10m` | Claude プロセスのアイドルタイムアウト |

### リレー設定

| 変数 | デフォルト | 説明 |
|------|-----------|------|
| `RELAY_ENABLED` | `true` | リレー接続を有効化 |
| `CLOUD_URL` | `cloud.devport.com` | リレーサーバーURL |
| `RELAY_PORT` | `SERVER_PORT` | リレー転送先ポート |

### Git 設定

| 変数 | 説明 |
|------|------|
| `GIT_ENABLED` | Git 機能を有効化（`true`/`false`） |
| `REPOSITORY_URL` | リポジトリ URL |
| `REPOSITORY_TOKEN` | アクセストークン（PAT） |
| `GIT_USER_NAME` | コミット時のユーザー名 |
| `GIT_USER_EMAIL` | コミット時のメールアドレス |

### ログ設定

| 変数 | デフォルト | 説明 |
|------|-----------|------|
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `text` | `text` または `json` |
| `LOG_FILE` | stdout | ログ出力先ファイル |

---

## リバースプロキシ設定

### Nginx

```nginx
server {
    listen 80;
    server_name devport.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }
}
```

### Caddy

```caddyfile
devport.example.com {
    reverse_proxy localhost:8080
}
```

---

## セキュリティ考慮事項

### 推奨設定

1. **HTTPS を使用**: リバースプロキシで TLS 終端
2. **強力なトークン**: `AUTH_TOKEN` は十分な長さと複雑さを持たせる
3. **ファイアウォール**: 8080 ポートを直接公開しない
4. **定期更新**: Docker イメージを定期的に更新

### 本番環境チェックリスト

- [ ] HTTPS が有効
- [ ] AUTH_TOKEN が強力
- [ ] ログが適切に設定
- [ ] バックアップが設定
- [ ] 監視が設定

---

## トラブルシューティング

### Docker コンテナが起動しない

```bash
docker compose logs devport
```

よくある原因:
- `AUTH_TOKEN` が未設定
- ポートが既に使用中
- ボリュームの権限問題

### Claude CLI が動作しない

```bash
# コンテナ内で確認
docker compose exec devport claude --version
docker compose exec devport claude auth status
```

### WebSocket 接続エラー

リバースプロキシを使用している場合:
- `Upgrade` ヘッダーが転送されているか確認
- `proxy_read_timeout` が十分長いか確認

---

## 本番デプロイチェックリスト

### リレーサーバー（VPS）

#### デプロイ前

- [ ] **VPS 選定**
  - [ ] リージョンがユーザーに近い
  - [ ] 2GB RAM 以上
  - [ ] SSD ストレージ
- [ ] **DNS 設定**
  - [ ] `A` レコード: `cloud.devport.app` → VPS IP
  - [ ] `A` レコード: `*.cloud.devport.app` → VPS IP
  - [ ] DNS 伝播完了を確認（`dig cloud.devport.app`）
- [ ] **ドメイン準備**
  - [ ] ドメイン取得済み
  - [ ] DNS プロバイダーの API トークン取得済み

#### デプロイ中

- [ ] **セキュリティ設定**
  - [ ] SSH 鍵認証のみ許可
  - [ ] ファイアウォール設定（22, 80, 443 のみ開放）
  - [ ] fail2ban インストール済み
- [ ] **Docker 設定**
  - [ ] Docker インストール済み
  - [ ] docker-compose.yml 作成済み
  - [ ] コンテナ正常起動確認
- [ ] **リバースプロキシ設定**
  - [ ] Caddy/Nginx インストール済み
  - [ ] HTTPS 証明書取得済み
  - [ ] ワイルドカード証明書設定済み
  - [ ] WebSocket プロキシ設定済み

#### デプロイ後

- [ ] **動作確認**
  - [ ] `/health` エンドポイント応答確認
  - [ ] `/api/relay/register` API 動作確認
  - [ ] WebSocket 接続確認（wss://）
  - [ ] サブドメインルーティング確認
- [ ] **監視設定**
  - [ ] ヘルスチェック監視設定
  - [ ] ログ監視設定
  - [ ] ディスク使用量アラート
  - [ ] メモリ使用量アラート
- [ ] **バックアップ設定**
  - [ ] 設定ファイルのバックアップ
  - [ ] 復旧手順の文書化

### ローカルサーバー（ユーザー環境）

#### デプロイ前

- [ ] **環境確認**
  - [ ] Go 1.25+ インストール済み
  - [ ] Node.js 22+ インストール済み
  - [ ] Claude CLI インストール済み（`claude --version`）
  - [ ] Claude CLI 認証済み（`claude auth status`）

#### デプロイ中

- [ ] **ビルド**
  - [ ] フロントエンドビルド成功
  - [ ] バックエンドビルド成功
  - [ ] 静的ファイルコピー済み
- [ ] **環境変数設定**
  - [ ] `AUTH_TOKEN` 設定済み（強力なパスワード）
  - [ ] `RELAY_ENABLED=true` 設定済み
  - [ ] `RELAY_URL` 正しく設定

#### デプロイ後

- [ ] **動作確認**
  - [ ] サーバー起動確認
  - [ ] リレーサーバーへの接続確認
  - [ ] QR コード表示確認
  - [ ] モバイルからの接続テスト
  - [ ] チャット送受信テスト

---

## 監視とアラート

### 推奨監視項目

| 項目 | 閾値 | 重要度 |
|------|------|--------|
| ヘルスチェック | 3回連続失敗 | Critical |
| CPU 使用率 | > 80% | Warning |
| メモリ使用率 | > 85% | Warning |
| ディスク使用率 | > 90% | Critical |
| WebSocket 接続数 | > 1000 | Warning |
| 応答時間 | > 500ms | Warning |

### Prometheus + Grafana での監視（オプション）

```yaml
# docker-compose.monitoring.yml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

### UptimeRobot / Better Uptime での外部監視

1. **HTTP モニター**
   - URL: `https://cloud.devport.app/health`
   - 間隔: 1分
   - アラート: Slack / Email

2. **キーワードモニター**
   - 期待値: `"status":"ok"`

---

## ロールバック手順

### Docker の場合

```bash
# 現在のイメージタグを確認
docker images | grep devport

# 前バージョンにロールバック
docker compose down
docker compose pull devport-relay:v1.0.0  # 特定バージョン指定
docker compose up -d

# または特定のダイジェストを指定
docker pull ghcr.io/noon-r/devport-relay@sha256:abc123...
```

### ネイティブビルドの場合

```bash
# バックアップからリストア
cp /opt/devport/devport.backup /opt/devport/devport
sudo systemctl restart devport
```

---

## 障害時の連絡フロー

1. **検知**: 監視システムがアラート発報
2. **一次対応**: ログ確認、サービス再起動試行
3. **エスカレーション**: 再起動で復旧しない場合
4. **復旧確認**: ヘルスチェックとユーザー動作確認
5. **事後報告**: 障害報告書作成

---

## セキュリティアップデート

### 定期更新スケジュール

| 項目 | 頻度 |
|------|------|
| OS パッケージ | 週1回 |
| Docker イメージ | 新バージョンリリース時 |
| SSL 証明書 | 自動更新（Let's Encrypt）|
| 依存パッケージ | 月1回（脆弱性チェック）|

### 自動更新スクリプト（オプション）

```bash
#!/bin/bash
# /opt/devport-relay/update.sh

cd /opt/devport-relay

# 最新イメージを取得
docker compose pull

# 変更があればコンテナを再作成
docker compose up -d

# 古いイメージを削除
docker image prune -f

# ログ出力
echo "$(date): Update completed" >> /var/log/devport-update.log
```

```bash
# cron で週1回実行
echo "0 3 * * 0 /opt/devport-relay/update.sh" | sudo crontab -
```
