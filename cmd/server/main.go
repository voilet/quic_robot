package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rs/zerolog"
)

type Server struct {
	quicListener  quic.Listener
	agents        map[string]*AgentConnection
	auditLogger   *AuditLogger
	websocketPort string
	logger        zerolog.Logger
}

type AgentConnection struct {
	AgentID    string
	Conn       quic.Connection
	LastSeen   time.Time
	Streams    map[int64]quic.Stream
}

func main() {
	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger

	// Create logs directory
	logDir := "/var/log/quic-robot"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// Initialize audit logger
	auditLogger, err := NewAuditLogger(filepath.Join(logDir, "audit.log"))
	if err != nil {
		log.Fatalf("Failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	// Load TLS certificates
	cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		log.Fatalf("Failed to load certificates: %v. Run 'generate-certs.sh' first.", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"quic-robot"},
	}

	// Create QUIC listener
	quicListener, err := quic.ListenAddr("0.0.0.0:443", tlsConfig, &quic.Config{
		MaxIdleTimeout:  time.Minute * 30,
		KeepAlivePeriod: time.Second * 15,
	})
	if err != nil {
		log.Fatalf("Failed to listen on QUIC: %v", err)
	}
	defer quicListener.Close()

	server := &Server{
		quicListener:  quicListener,
		agents:        make(map[string]*AgentConnection),
		auditLogger:   auditLogger,
		websocketPort: "8080",
		logger:        logger,
	}

	// Start WebSocket server
	go server.startWebSocketServer()

	logger.Info().Msg("🚀 QUIC Robot Server started on :443")
	logger.Info().Msg("📡 WebSocket server started on :8080")
	logger.Info().Msg("📝 Audit logs: /var/log/quic-robot/audit.log")

	// Accept incoming connections
	for {
		conn, err := quicListener.Accept(context.Background())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to accept connection")
			continue
		}

		go server.handleAgent(conn)
	}
}

func (s *Server) handleAgent(conn quic.Connection) {
	// Get remote address
	remoteAddr := conn.RemoteAddr().String()
	s.logger.Info().Str("addr", remoteAddr).Msg("📡 New agent connection")

	// Wait for agent to register
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to accept stream")
		return
	}

	// Read agent registration
	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to read agent ID")
		return
	}

	agentID := string(buf[:n])
	s.logger.Info().Str("agent_id", agentID).Msg("✅ Agent registered")

	// Store agent connection
	agentConn := &AgentConnection{
		AgentID:  agentID,
		Conn:     conn,
		LastSeen: time.Now(),
		Streams:  make(map[int64]quic.Stream),
	}
	s.agents[agentID] = agentConn

	// Log registration
	s.auditLogger.LogEvent(agentID, "agent_connected", map[string]interface{}{
		"remote_addr": remoteAddr,
		"timestamp":   time.Now().Unix(),
	})

	// Send confirmation
	stream.Write([]byte("OK"))

	// Handle agent streams
	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			s.logger.Error().Err(err).Str("agent_id", agentID).Msg("Stream error")
			break
		}

		go s.handleAgentStream(agentConn, stream)
	}

	// Cleanup
	delete(s.agents, agentID)
	s.logger.Info().Str("agent_id", agentID).Msg("❌ Agent disconnected")
}

func (s *Server) handleAgentStream(agent *AgentConnection, stream quic.Stream) {
	// Read stream type
	buf := make([]byte, 1)
	_, err := stream.Read(buf)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to read stream type")
		return
	}

	streamType := buf[0]

	switch streamType {
	case 1: // SSH data stream
		s.handleSSHStream(agent, stream)
	default:
		s.logger.Warn().Uint8("type", streamType).Msg("Unknown stream type")
	}
}

func (s *Server) handleSSHStream(agent *AgentConnection, stream quic.Stream) {
	s.logger.Info().Str("agent_id", agent.AgentID).Msg("🔌 New SSH stream")

	// TODO: Implement SSH data forwarding
	// For now, just echo data
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			s.logger.Error().Err(err).Msg("SSH stream read error")
			break
		}

		// Echo back (for testing)
		stream.Write(buf[:n])
	}
}

func (s *Server) startWebSocketServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	mux.Handle("/", http.FileServer(http.Dir("web/dist")))

	s.logger.Info().Str("port", s.websocketPort).Msg("🌐 WebSocket server listening")
	if err := http.ListenAndServe(":"+s.websocketPort, mux); err != nil {
		s.logger.Error().Err(err).Msg("WebSocket server error")
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement WebSocket handler
	// Upgrade to WebSocket, connect to agent via QUIC
	w.Write([]byte("WebSocket endpoint - TODO"))
}
