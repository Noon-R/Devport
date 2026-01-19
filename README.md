# Devport

Mobile-first AI programming platform where AI editing is the primary mode of interaction.

## Quick Start

### Prerequisites

- Node.js 22+
- Go 1.25+
- Claude CLI (`npm i -g @anthropic-ai/claude-code`)

### Development

```bash
# Install dependencies
npm install
cd web && npm install

# Run both frontend and backend
npm run dev
```

- Frontend: http://localhost:5173
- Backend: http://localhost:9870

### Frontend only

```bash
cd web
npm run dev
```

### Backend only

```bash
cd server
AUTH_TOKEN=your_password go run .
```

## Project Structure

```
devport/
├── web/          # React frontend (Vite + TypeScript + Tailwind)
├── server/       # Go backend (WebSocket + JSON-RPC)
├── relay/        # Relay server for NAT traversal
├── docs/         # Documentation
├── docker/       # Docker configurations
├── scripts/      # Utility scripts
└── site/         # Landing page (Hugo)
```

## Documentation

See [docs/](./docs/) for detailed documentation:

- [Architecture](./docs/architecture.md)
- [Setup Guide](./docs/setup-guide.md)
- [API Reference](./docs/api-reference.md)
- [Implementation Plan](./docs/implementation-plan.md)

## License

MIT
