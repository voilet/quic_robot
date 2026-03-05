# Development Quick Start

## Prerequisites

- Go 1.21+
- Node.js 18+
- OpenSSL (for cert generation)

## Setup

1. Generate certificates:
```bash
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

2. Install Go dependencies:
```bash
go mod tidy
```

3. Install frontend dependencies:
```bash
cd web
npm install
```

## Run Development

### Terminal 1 - Start Server
```bash
go run cmd/server/main.go
```

### Terminal 2 - Start Agent
```bash
QUIC_SERVER=localhost:443 AGENT_ID=agent-1 go run cmd/agent/main.go
```

### Terminal 3 - Start Frontend
```bash
cd web
npm run dev
```

### Access

Open browser to: http://localhost:5173

## Testing

1. Enter connection details:
   - Agent ID: agent-1
   - Host: localhost
   - Port: 22
   - User: your_ssh_username
   - Password: your_ssh_password

2. Click "Connect"
3. You should see an SSH terminal!

## Architecture

```
Web Browser (React + xterm.js)
    ↓ WebSocket
Go Server (QUIC endpoint)
    ↓ QUIC
Go Agent (SSH client)
    ↓ SSH
Target Host
```

## TODO

- [ ] Implement bidirectional WebSocket ↔ QUIC streaming
- [ ] Add session recording (asciinema format)
- [ ] Add JWT authentication
- [ ] Add session list API
- [ ] Add audit log viewer
- [ ] Add agent status dashboard
