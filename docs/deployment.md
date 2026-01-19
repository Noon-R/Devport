# デプロイガイド

## 概要

Devport のデプロイ方法は主に2つ:

1. **Docker**: 推奨。環境構築が簡単
2. **ネイティブビルド**: Claude CLI が既にインストールされている環境向け

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
