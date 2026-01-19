# JSON-RPC API リファレンス

## 概要

Devport の全ての通信は WebSocket 上の JSON-RPC 2.0 で行う。

- **エンドポイント**: `ws://localhost:9870/ws`（ローカル）または `wss://{subdomain}.cloud.devport.app/ws`（リレー経由）
- **プロトコル**: JSON-RPC 2.0
- **認証**: 接続後に `auth` メソッドを呼び出し

## 接続フロー

```
1. WebSocket 接続確立
2. auth メソッドでトークン認証
3. chat.attach でセッションに接続
4. 各種メソッドを呼び出し
```

---

## 認証

### auth

トークン認証を行う。接続後最初に呼び出す必要がある。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "auth",
  "params": {
    "token": "your_auth_token"
  },
  "id": 1
}
```

**レスポンス（成功）:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "success": true
  },
  "id": 1
}
```

**レスポンス（失敗）:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32001,
    "message": "Invalid token"
  },
  "id": 1
}
```

---

## チャット (chat.*)

### chat.attach

セッションに接続し、イベントの購読を開始する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "chat.attach",
  "params": {
    "session_id": "session_123"
  },
  "id": 2
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "session_id": "session_123",
    "status": "attached"
  },
  "id": 2
}
```

### chat.message

ユーザーメッセージを送信する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "chat.message",
  "params": {
    "session_id": "session_123",
    "content": "Hello, Claude!"
  },
  "id": 3
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "accepted": true
  },
  "id": 3
}
```

### chat.interrupt

AI の処理を中断する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "chat.interrupt",
  "params": {
    "session_id": "session_123"
  },
  "id": 4
}
```

### chat.permission_response

権限リクエストに応答する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "chat.permission_response",
  "params": {
    "session_id": "session_123",
    "permission_id": "perm_456",
    "allowed": true
  },
  "id": 5
}
```

### chat.question_response

ユーザーへの質問に応答する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "chat.question_response",
  "params": {
    "session_id": "session_123",
    "question_id": "q_789",
    "answer": "Option A"
  },
  "id": 6
}
```

---

## セッション (session.*)

### session.list

セッション一覧を取得する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "session.list",
  "params": {},
  "id": 10
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "sessions": [
      {
        "id": "session_123",
        "title": "My Chat",
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T11:00:00Z"
      }
    ]
  },
  "id": 10
}
```

### session.create

新規セッションを作成する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "session.create",
  "params": {
    "title": "New Chat"
  },
  "id": 11
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "session": {
      "id": "session_456",
      "title": "New Chat",
      "created_at": "2024-01-15T12:00:00Z"
    }
  },
  "id": 11
}
```

### session.delete

セッションを削除する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "session.delete",
  "params": {
    "session_id": "session_123"
  },
  "id": 12
}
```

### session.update_title

セッションのタイトルを更新する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "session.update_title",
  "params": {
    "session_id": "session_123",
    "title": "Updated Title"
  },
  "id": 13
}
```

### session.get_history

セッションの履歴を取得する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "session.get_history",
  "params": {
    "session_id": "session_123"
  },
  "id": 14
}
```

---

## ファイル (file.*)

### file.get

ファイルの内容を取得する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "file.get",
  "params": {
    "path": "src/main.ts"
  },
  "id": 20
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "path": "src/main.ts",
    "content": "console.log('Hello');",
    "language": "typescript"
  },
  "id": 20
}
```

---

## Git (git.*)

### git.status

Git ステータスを取得する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "git.status",
  "params": {},
  "id": 30
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "branch": "main",
    "staged": ["src/new.ts"],
    "modified": ["src/app.ts"],
    "untracked": ["temp.txt"]
  },
  "id": 30
}
```

### git.diff

Git diff を取得する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "git.diff",
  "params": {
    "path": "src/app.ts"
  },
  "id": 31
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "diff": "--- a/src/app.ts\n+++ b/src/app.ts\n@@ -1,3 +1,4 @@\n+// Added comment\n console.log('app');"
  },
  "id": 31
}
```

---

## ワークツリー (worktree.*)

### worktree.list

ワークツリー一覧を取得する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "worktree.list",
  "params": {},
  "id": 40
}
```

**レスポンス:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "worktrees": [
      {
        "name": "main",
        "path": "/workspace",
        "branch": "main",
        "is_main": true
      },
      {
        "name": "feature-x",
        "path": "/workspace/.worktrees/feature-x",
        "branch": "feature-x",
        "is_main": false
      }
    ]
  },
  "id": 40
}
```

### worktree.create

新規ワークツリーを作成する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "worktree.create",
  "params": {
    "name": "feature-y",
    "branch": "feature-y"
  },
  "id": 41
}
```

### worktree.delete

ワークツリーを削除する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "worktree.delete",
  "params": {
    "name": "feature-x"
  },
  "id": 42
}
```

### worktree.switch

アクティブなワークツリーを切り替える。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "worktree.switch",
  "params": {
    "name": "feature-x"
  },
  "id": 43
}
```

---

## 監視 (watch.*)

### watch.subscribe

ファイルシステムの変更を監視する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "watch.subscribe",
  "params": {
    "path": "src/"
  },
  "id": 50
}
```

### watch.unsubscribe

監視を解除する。

**リクエスト:**
```json
{
  "jsonrpc": "2.0",
  "method": "watch.unsubscribe",
  "params": {
    "path": "src/"
  },
  "id": 51
}
```

---

## サーバー → クライアント通知

サーバーからクライアントへの一方向通知（`id` フィールドなし）。

### chat.text

AI のテキスト出力（ストリーミング）。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.text",
  "params": {
    "session_id": "session_123",
    "content": "Here is my response..."
  }
}
```

### chat.tool_call

ツール呼び出しの開始。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.tool_call",
  "params": {
    "session_id": "session_123",
    "tool_use_id": "tool_001",
    "tool_name": "Read",
    "input": {
      "file_path": "/src/app.ts"
    }
  }
}
```

### chat.tool_result

ツール実行の結果。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.tool_result",
  "params": {
    "session_id": "session_123",
    "tool_use_id": "tool_001",
    "output": "File content here..."
  }
}
```

### chat.error

エラー発生。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.error",
  "params": {
    "session_id": "session_123",
    "error": "Process crashed unexpectedly"
  }
}
```

### chat.done

応答完了。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.done",
  "params": {
    "session_id": "session_123"
  }
}
```

### chat.permission_request

権限リクエスト。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.permission_request",
  "params": {
    "session_id": "session_123",
    "permission_id": "perm_456",
    "tool_name": "Bash",
    "description": "Run: npm install"
  }
}
```

### chat.ask_user_question

ユーザーへの質問。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.ask_user_question",
  "params": {
    "session_id": "session_123",
    "question_id": "q_789",
    "question": "Which option do you prefer?",
    "options": ["Option A", "Option B"]
  }
}
```

### chat.system

システムメッセージ。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.system",
  "params": {
    "session_id": "session_123",
    "message": "Session started"
  }
}
```

### chat.interrupted

処理中断完了。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.interrupted",
  "params": {
    "session_id": "session_123"
  }
}
```

### chat.process_ended

プロセス終了。

```json
{
  "jsonrpc": "2.0",
  "method": "chat.process_ended",
  "params": {
    "session_id": "session_123",
    "exit_code": 0
  }
}
```

---

## エラーコード

| コード | 意味 |
|--------|------|
| -32700 | Parse error（JSON パースエラー） |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |
| -32001 | Authentication failed |
| -32002 | Session not found |
| -32003 | Permission denied |
