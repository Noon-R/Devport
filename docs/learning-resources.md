# Devport 実装に必要な知識と参考リンク

## 目次

1. [フロントエンド](#フロントエンド)
2. [バックエンド](#バックエンド)
3. [通信プロトコル](#通信プロトコル)
4. [AI 統合](#ai-統合)
5. [インフラ・デプロイ](#インフラデプロイ)
6. [モバイルアプリ](#モバイルアプリ)

---

## フロントエンド

### React 19

モダンな UI 構築のためのライブラリ。

**必要な知識:**
- Hooks（useState, useEffect, useRef, useCallback, useMemo）
- コンポーネント設計パターン
- React 19 の新機能（Actions, use() hook, Server Components）

**参考リンク:**
- [React 公式ドキュメント](https://react.dev/)
- [React 19 リリースノート](https://react.dev/blog/2024/12/05/react-19)
- [React Hooks 完全ガイド](https://overreacted.io/a-complete-guide-to-useeffect/)

---

### TypeScript 5

型安全な JavaScript 開発。

**必要な知識:**
- 基本型（string, number, boolean, array, object）
- インターフェースと型エイリアス
- ジェネリクス
- ユニオン型、交差型
- 型ガード、型推論

**参考リンク:**
- [TypeScript 公式ドキュメント](https://www.typescriptlang.org/docs/)
- [TypeScript Deep Dive（日本語）](https://typescript-jp.gitbook.io/deep-dive)
- [Type Challenges](https://github.com/type-challenges/type-challenges) - 型パズルで学習

---

### Vite 7

高速なビルドツール・開発サーバー。

**必要な知識:**
- 設定ファイル（vite.config.ts）
- プラグインシステム
- 環境変数の扱い
- プロキシ設定（開発時のAPI接続）
- ビルド最適化

**参考リンク:**
- [Vite 公式ドキュメント](https://vite.dev/)
- [Vite プラグイン一覧](https://vite.dev/plugins/)

---

### Tailwind CSS 4

ユーティリティファーストの CSS フレームワーク。

**必要な知識:**
- ユーティリティクラスの基本（flex, grid, spacing, colors）
- レスポンシブデザイン（sm:, md:, lg:）
- ダークモード対応
- カスタマイズ（tailwind.config.js）

**参考リンク:**
- [Tailwind CSS 公式ドキュメント](https://tailwindcss.com/docs)
- [Tailwind CSS Cheat Sheet](https://nerdcave.com/tailwind-cheat-sheet)
- [Tailwind UI](https://tailwindui.com/) - コンポーネント例

---

### Zustand 5

軽量な状態管理ライブラリ。

**必要な知識:**
- ストアの作成と使用
- セレクター（パフォーマンス最適化）
- ミドルウェア（persist, devtools）
- 非同期アクション

**参考リンク:**
- [Zustand GitHub](https://github.com/pmndrs/zustand)
- [Zustand ドキュメント](https://docs.pmnd.rs/zustand/getting-started/introduction)
- [Zustand vs Redux 比較](https://blog.logrocket.com/zustand-vs-redux/)

---

### React Query (TanStack Query) 5

サーバー状態管理ライブラリ。

**必要な知識:**
- useQuery, useMutation
- キャッシュ戦略
- 楽観的更新
- 無限スクロール（useInfiniteQuery）
- WebSocket との組み合わせ

**参考リンク:**
- [TanStack Query 公式ドキュメント](https://tanstack.com/query/latest)
- [Practical React Query](https://tkdodo.eu/blog/practical-react-query) - ベストプラクティス

---

### TanStack Router

型安全なファイルベースルーティング。

**必要な知識:**
- ファイルベースルーティングの仕組み
- 動的ルート、ネストルート
- ローダーとアクション
- 型安全なパラメータ

**参考リンク:**
- [TanStack Router 公式ドキュメント](https://tanstack.com/router/latest)

---

### Biome 2

ESLint + Prettier の高速な代替。

**必要な知識:**
- 設定ファイル（biome.json）
- ルールのカスタマイズ
- VS Code 拡張との連携

**参考リンク:**
- [Biome 公式ドキュメント](https://biomejs.dev/)
- [Biome vs ESLint + Prettier](https://biomejs.dev/blog/biome-v1/)

---

## バックエンド

### Go 1.25

高性能なサーバーサイド言語。

**必要な知識:**
- 基本構文（変数、関数、構造体、インターフェース）
- goroutine と channel（並行処理）
- エラーハンドリング
- パッケージ管理（go mod）
- 標準ライブラリ（net/http, encoding/json, os/exec）
- Context によるキャンセル伝播

**参考リンク:**
- [Go 公式チュートリアル](https://go.dev/doc/tutorial/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Go 言語による並行処理](https://www.oreilly.co.jp/books/9784873118468/) - 書籍

---

### coder/websocket

Go の WebSocket ライブラリ。

**必要な知識:**
- WebSocket 接続の受け入れ
- メッセージの読み書き
- 接続のクローズ処理
- 複数接続の管理

**参考リンク:**
- [coder/websocket GitHub](https://github.com/coder/websocket)
- [WebSocket API 仕様](https://developer.mozilla.org/ja/docs/Web/API/WebSockets_API)

---

### sourcegraph/jsonrpc2

Go の JSON-RPC 2.0 実装。

**必要な知識:**
- JSON-RPC 2.0 の仕組み
- ハンドラーの実装
- 通知とリクエストの違い

**参考リンク:**
- [sourcegraph/jsonrpc2 GitHub](https://github.com/sourcegraph/jsonrpc2)
- [JSON-RPC 2.0 仕様](https://www.jsonrpc.org/specification)

---

### fsnotify

ファイルシステム監視ライブラリ。

**必要な知識:**
- ファイル変更イベントの種類
- 監視対象の追加・削除
- イベントのデバウンス処理

**参考リンク:**
- [fsnotify GitHub](https://github.com/fsnotify/fsnotify)

---

### プロセス管理（os/exec）

外部プロセス（Claude CLI）の管理。

**必要な知識:**
- exec.Command の使い方
- stdin/stdout/stderr パイプ
- プロセスのライフサイクル管理
- シグナル送信（中断処理）
- Context によるタイムアウト

**参考リンク:**
- [Go os/exec パッケージ](https://pkg.go.dev/os/exec)
- [Go でのプロセス管理](https://blog.gopheracademy.com/advent-2017/run-os-exec/)

---

## 通信プロトコル

### WebSocket

双方向リアルタイム通信。

**必要な知識:**
- WebSocket ハンドシェイク
- メッセージフレーミング
- 接続状態管理
- 再接続ロジック（指数バックオフ）
- ハートビート/Ping-Pong

**参考リンク:**
- [WebSocket API (MDN)](https://developer.mozilla.org/ja/docs/Web/API/WebSocket)
- [WebSocket プロトコル (RFC 6455)](https://datatracker.ietf.org/doc/html/rfc6455)
- [WebSocket 実践ガイド](https://javascript.info/websocket)

---

### JSON-RPC 2.0

軽量な RPC プロトコル。

**必要な知識:**
- リクエスト/レスポンス形式
- 通知（id なし）
- エラーオブジェクト
- バッチリクエスト

**参考リンク:**
- [JSON-RPC 2.0 仕様](https://www.jsonrpc.org/specification)
- [json-rpc-2.0 (npm)](https://www.npmjs.com/package/json-rpc-2.0)

---

## AI 統合

### Claude CLI

Anthropic の AI コーディングアシスタント。

**必要な知識:**
- インストールと認証
- コマンドラインオプション
- stream-json 入出力形式
- イベントタイプ（assistant, tool_use, tool_result, result）
- 権限プロンプト（permission-prompt-tool stdio）
- セッション管理

**参考リンク:**
- [Claude Code 公式ドキュメント](https://docs.anthropic.com/en/docs/claude-code)
- [Claude CLI GitHub](https://github.com/anthropics/claude-code)

---

### stream-json 形式

Claude CLI の入出力フォーマット。

**入力形式:**
```json
{"type": "user_message", "content": "Hello"}
{"type": "interrupt"}
{"type": "permission_response", "permission_id": "xxx", "allowed": true}
{"type": "question_response", "question_id": "xxx", "answer": "Option A"}
```

**出力イベント:**
| イベント | 説明 |
|---------|------|
| `assistant` | AI のテキスト応答 |
| `content_block_start` | ツール呼び出し開始 |
| `content_block_delta` | ストリーミングテキスト |
| `tool_result` | ツール実行結果 |
| `result` | 応答完了 |
| `permission_request` | 権限リクエスト |
| `ask_user_question` | ユーザーへの質問 |

---

## インフラ・デプロイ

### Docker

コンテナ化とデプロイ。

**必要な知識:**
- Dockerfile の書き方
- マルチステージビルド
- docker-compose
- ボリュームマウント
- 環境変数の扱い

**参考リンク:**
- [Docker 公式ドキュメント](https://docs.docker.com/)
- [Docker チュートリアル（日本語）](https://docs.docker.jp/)
- [Dockerfile ベストプラクティス](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)

---

### NAT 越えソリューション

外出先からローカル PC にアクセス。

#### Tailscale（推奨）

**必要な知識:**
- WireGuard の基本概念
- Tailscale のネットワーク構成
- MagicDNS

**参考リンク:**
- [Tailscale 公式ドキュメント](https://tailscale.com/kb/)
- [Tailscale ダウンロード](https://tailscale.com/download)

#### Cloudflare Tunnel

**必要な知識:**
- トンネルの作成と管理
- DNS 設定
- Zero Trust の概念

**参考リンク:**
- [Cloudflare Tunnel ドキュメント](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
- [cloudflared GitHub](https://github.com/cloudflare/cloudflared)

#### ngrok

**参考リンク:**
- [ngrok 公式](https://ngrok.com/)
- [ngrok ドキュメント](https://ngrok.com/docs)

---

### リバースプロキシ

本番環境での TLS 終端とルーティング。

#### Nginx

**参考リンク:**
- [Nginx 公式ドキュメント](https://nginx.org/en/docs/)
- [Nginx WebSocket プロキシ設定](https://nginx.org/en/docs/http/websocket.html)

#### Caddy

**参考リンク:**
- [Caddy 公式](https://caddyserver.com/)
- [Caddy ドキュメント](https://caddyserver.com/docs/)

---

## モバイルアプリ

### iOS (SwiftUI)

**必要な知識:**
- SwiftUI の基本（View, State, Binding）
- URLSession WebSocket API
- Combine フレームワーク
- App Store 申請プロセス

**参考リンク:**
- [SwiftUI チュートリアル](https://developer.apple.com/tutorials/swiftui)
- [URLSessionWebSocketTask](https://developer.apple.com/documentation/foundation/urlsessionwebsockettask)
- [Hacking with Swift](https://www.hackingwithswift.com/)

---

### Android (Kotlin + Jetpack Compose)

**必要な知識:**
- Kotlin 基本構文
- Jetpack Compose（宣言的 UI）
- OkHttp WebSocket
- Flow（リアクティブストリーム）
- Google Play 申請プロセス

**参考リンク:**
- [Jetpack Compose チュートリアル](https://developer.android.com/jetpack/compose/tutorial)
- [OkHttp WebSocket](https://square.github.io/okhttp/4.x/okhttp/okhttp3/-web-socket/)
- [Kotlin 公式ドキュメント](https://kotlinlang.org/docs/home.html)

---

## セキュリティ

### 認証・認可

**必要な知識:**
- トークンベース認証
- タイミング攻撃対策（constant-time compare）
- HTTPS/WSS の重要性

**参考リンク:**
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)

---

### 入力検証

**必要な知識:**
- パストラバーサル攻撃対策
- プロンプトインジェクション対策
- XSS 対策

**参考リンク:**
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)

---

## Git

### 基本操作

**必要な知識:**
- コミット、ブランチ、マージ
- リモートリポジトリ操作
- Git ワークツリー（複数ブランチの同時作業）

**参考リンク:**
- [Pro Git（日本語）](https://git-scm.com/book/ja/v2)
- [Git ワークツリー](https://git-scm.com/docs/git-worktree)

---

## 学習の順序（推奨）

### Phase 1: 基礎（1-2週間）
1. Go 言語の基本構文
2. TypeScript の基本
3. React Hooks

### Phase 2: 通信（1週間）
1. WebSocket の仕組み
2. JSON-RPC 2.0 仕様
3. Go での WebSocket 実装

### Phase 3: AI 統合（1週間）
1. Claude CLI のインストールと基本操作
2. stream-json 形式の理解
3. プロセス管理（stdin/stdout パイプ）

### Phase 4: フロントエンド（1-2週間）
1. Vite + React + TypeScript セットアップ
2. Zustand による状態管理
3. WebSocket クライアント実装

### Phase 5: 本番化（1週間）
1. Docker によるコンテナ化
2. リバースプロキシ設定
3. NAT 越えソリューション選択

### Phase 6: モバイル（オプション、2-4週間）
1. SwiftUI または Jetpack Compose の基本
2. WebSocket クライアント実装
3. アプリストア申請

---

## 書籍（推奨）

| タイトル | 対象 |
|---------|------|
| [Go 言語による並行処理](https://www.oreilly.co.jp/books/9784873118468/) | Go の並行処理 |
| [プログラミング TypeScript](https://www.oreilly.co.jp/books/9784873119045/) | TypeScript |
| [React ハンズオンラーニング 第2版](https://www.oreilly.co.jp/books/9784873119380/) | React |
| [Web API: The Good Parts](https://www.oreilly.co.jp/books/9784873116860/) | API 設計 |
