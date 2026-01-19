# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Devport is a mobile-first programming platform where AI editing is the primary mode of interaction. Users interact with Claude CLI via natural language rather than manually editing code on small screens.

## Architecture

```
Mobile (React SPA) ──WebSocket JSON-RPC 2.0──▶ cloud.devport.app (Relay) ──▶ Local PC (Go Server) ──stdin/stdout──▶ Claude CLI
```

**Key Components:**
- `web/` - React frontend with Zustand (state), React Query (server state), TanStack Router (file-based routing)
- `server/` - Go backend with WebSocket JSON-RPC, process management, Git worktree management
- `site/` - devport.app marketing site (Hugo)

**Communication:** All client-server communication uses WebSocket JSON-RPC 2.0 (no REST API). See `docs/api-reference.md` for method details.

**AI Integration:** Claude CLI runs as subprocess with `--output-format stream-json --input-format stream-json --permission-prompt-tool stdio`

## Development Commands

### Quick Start (both servers)
```bash
npm run dev  # Runs Go server (port 9870) + Vite dev server (port 5173) concurrently
```

### Frontend (`web/`)
```bash
npm run dev      # Vite dev server
npm run build    # Production build (outputs to dist/)
npm run lint     # Biome lint
npm run format   # Biome format
npm run test     # Vitest
```

### Backend (`server/`)
```bash
# Windows PowerShell
$env:AUTH_TOKEN="password"; $env:DEV_MODE="true"; go run .

# Unix/macOS
AUTH_TOKEN=password DEV_MODE=true go run .

go build -o pockode .    # Build binary
go test ./...            # Run tests
go vet ./...             # Static analysis
```

## Required Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `AUTH_TOKEN` | Yes | API authentication token |
| `DEV_MODE` | No | Set "true" for dev (disables static file serving) |
| `SERVER_PORT` | No | Default 9870 (dev) or 8080 (prod) |

## Tech Stack

- **Frontend:** React 19, Vite 7, TypeScript 5, Tailwind 4, Biome 2
- **Backend:** Go 1.25, coder/websocket, sourcegraph/jsonrpc2, fsnotify
- **Prerequisites:** Node.js 22+, Go 1.25+, Claude CLI (`npm i -g @anthropic-ai/claude-code`)

## Key Design Patterns

- **Lazy process creation:** Claude CLI processes spawn per-session on demand
- **Reference counting:** Multiple WebSocket connections share the same Claude process
- **Idle timeout:** Processes auto-cleanup after 10 minutes of inactivity
- **File-based storage:** Session data in `.devport/sessions/`, no external database
