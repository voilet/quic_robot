#!/bin/bash

echo "🚀 Setting up QUIC Robot development environment..."

# Create directories
mkdir -p internal/audit internal/ssh internal/quic web/dist
mkdir -p /var/log/quic-robot

# Generate certificates
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

# Initialize web project
echo "🌐 Initializing frontend..."
cd web
npm create vite@latest . -- --template react-ts
npm install xterm xterm-addon-fit
npm install -D @types/node

echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Generate certs: ./scripts/generate-certs.sh"
echo "  2. Start server: go run cmd/server/main.go"
echo "  3. Start agent (in another terminal):"
echo "     QUIC_SERVER=localhost:443 AGENT_ID=agent-1 go run cmd/agent/main.go"
echo "  4. Start frontend (in another terminal):"
echo "     cd web && npm run dev"
