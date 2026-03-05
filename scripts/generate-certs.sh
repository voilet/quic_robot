#!/bin/bash
# Generate self-signed certificates for QUIC development

CERT_DIR="certs"
mkdir -p "$CERT_DIR"

echo "🔐 Generating self-signed certificates..."

# Generate private key
openssl genrsa -out "$CERT_DIR/server.key" 2048

# Generate certificate
openssl req -new -x509 -sha256 -key "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.crt" -days 365 \
  -subj "/C=US/ST=CA/L=SF/O=QUICRobot/CN=localhost"

echo "✅ Certificates generated:"
echo "   - $CERT_DIR/server.crt"
echo "   - $CERT_DIR/server.key"
echo ""
echo "⚠️  These are self-signed certs for development only!"
echo "   For production, use Let's Encrypt or your CA."
