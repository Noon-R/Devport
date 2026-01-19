# 開発環境構築ガイド

## 前提条件

### 必須

| ツール | バージョン | 確認コマンド |
|--------|-----------|-------------|
| Node.js | 22.x 以上 | `node -v` |
| npm | 10.x 以上 | `npm -v` |
| Go | 1.25.x 以上 | `go version` |
| Claude CLI | 最新 | `claude --version` |

### オプション

| ツール | 用途 |
|--------|------|
| Docker | コンテナ実行 |
| Git | バージョン管理 |

## インストール

### 1. リポジトリのクローン

```bash
git clone https://github.com/sijiaoh/devport.git
cd devport
```

### 2. Claude CLI のインストール

```bash
npm install -g @anthropic-ai/claude-code
```

認証設定:
```bash
claude auth
```

### 3. 依存関係のインストール

```bash
# フロントエンド
cd web
npm install

# バックエンド（自動で依存解決）
cd ../server
go mod download
```

## 開発サーバーの起動

### 方法1: 一括起動（推奨）

プロジェクトルートで:

```bash
npm run dev
```

これにより以下が並行起動:
- Go サーバー: `http://localhost:9870`
- Vite 開発サーバー: `http://localhost:5173`

### 方法2: 個別起動

**ターミナル1（バックエンド）:**
```bash
cd server
AUTH_TOKEN=your_password DEV_MODE=true go run .
```

**ターミナル2（フロントエンド）:**
```bash
cd web
npm run dev
```

### 接続確認

ブラウザで `http://localhost:5173` を開き、設定した `AUTH_TOKEN` で認証。

## OS 別の注意点

### Windows

| 項目 | 状況 |
|------|------|
| ビルド | ✅ 動作確認済み |
| fsnotify | ✅ Windows 対応 |
| Claude CLI | ✅ Windows 版あり |

**PowerShell での環境変数設定:**
```powershell
$env:AUTH_TOKEN="your_password"
$env:DEV_MODE="true"
go run .
```

**cmd での環境変数設定:**
```cmd
set AUTH_TOKEN=your_password
set DEV_MODE=true
go run .
```

### macOS

特別な設定は不要。

```bash
AUTH_TOKEN=your_password DEV_MODE=true go run .
```

### Linux

特別な設定は不要。

```bash
AUTH_TOKEN=your_password DEV_MODE=true go run .
```

## ビルド

### フロントエンドのビルド

```bash
cd web
npm run build
```

出力先: `web/dist/`（Brotli 圧縮済み）

### バックエンドのビルド

```bash
cd server
go build -o devport .
```

### 統合ビルド（本番用）

```bash
# フロントエンドをビルドしてサーバーに埋め込み
cd web && npm run build
cp -r dist ../server/static
cd ../server
go build -ldflags="-w -s" -o devport .
```

## コマンド一覧

### ルートディレクトリ

| コマンド | 説明 |
|---------|------|
| `npm run dev` | 開発サーバー一括起動 |
| `npm run build` | フロントエンド + バックエンドビルド |

### フロントエンド (`web/`)

| コマンド | 説明 |
|---------|------|
| `npm run dev` | Vite 開発サーバー |
| `npm run build` | 本番ビルド |
| `npm run lint` | Biome lint |
| `npm run format` | Biome format |
| `npm run test` | Vitest テスト |

### バックエンド (`server/`)

| コマンド | 説明 |
|---------|------|
| `go run .` | 開発サーバー起動 |
| `go build -o devport .` | バイナリビルド |
| `go test ./...` | テスト実行 |
| `gofmt -w .` | コード整形 |
| `go vet ./...` | 静的解析 |

## 環境変数

| 変数 | 必須 | デフォルト | 説明 |
|------|:----:|-----------|------|
| `AUTH_TOKEN` | ✓ | — | API 認証トークン |
| `SERVER_PORT` | | `9870` | サーバーポート |
| `WORK_DIR` | | `.` | 作業ディレクトリ |
| `DATA_DIR` | | `.devport/` | データ保存先 |
| `DEV_MODE` | | `false` | 開発モード（静的ファイル配信無効） |
| `RELAY_ENABLED` | | `true` | リレー接続を有効化 |
| `CLOUD_URL` | | `cloud.devport.com` | リレーサーバーURL |
| `LOG_LEVEL` | | `info` | ログレベル |
| `LOG_FORMAT` | | `text` | ログ形式（`text` / `json`） |

## トラブルシューティング

### Claude CLI が見つからない

```
Error: exec: "claude": executable file not found in $PATH
```

**解決策:**
```bash
npm install -g @anthropic-ai/claude-code
```

### WebSocket 接続エラー

```
WebSocket connection failed
```

**確認事項:**
1. バックエンドが起動しているか
2. `AUTH_TOKEN` が一致しているか
3. ポートが正しいか（開発時: 9870）

### ファイル監視が動作しない（Windows）

**解決策:**
- アンチウイルスソフトの除外設定を確認
- WSL2 ではなく PowerShell/cmd で実行

### Vite HMR が動作しない

**解決策:**
```bash
# node_modules を削除して再インストール
rm -rf node_modules
npm install
```
