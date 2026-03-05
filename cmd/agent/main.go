package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

type Agent struct {
	serverAddr    string
	agentID       string
	token         string
	quicConn      quic.Connection
	controlStream quic.Stream
	logger        zerolog.Logger
}

type SSHSession struct {
	AgentID    string
	SessionID  string
	Host       string
	Port       int
	User       string
	Password   string
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	QUICStream quic.Stream
}

func main() {
	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Configuration
	serverAddr := os.Getenv("QUIC_SERVER")
	if serverAddr == "" {
		serverAddr = "localhost:443"
	}

	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = fmt.Sprintf("agent-%d", time.Now().Unix())
	}

	token := os.Getenv("AGENT_TOKEN")
	if token == "" {
		token = "dev-token"
	}

	agent := &Agent{
		serverAddr: serverAddr,
		agentID:    agentID,
		token:      token,
		logger:     logger,
	}

	// Connect with retry
	for {
		if err := agent.Connect(); err != nil {
			logger.Error().Err(err).Msg("Connection failed, retrying in 5s...")
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	// Start heartbeat
	go agent.heartbeatLoop()

	// Accept SSH requests
	logger.Info().Msg("✅ Agent ready, waiting for SSH requests...")
	agent.handleSSHRequests()
}

func (a *Agent) Connect() error {
	// Create TLS config (skip verification for dev)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-robot"},
	}

	// Connect to server
	a.logger.Info().Str("server", a.serverAddr).Msg("🔌 Connecting to QUIC server...")

	conn, err := quic.DialAddr(context.Background(), a.serverAddr, tlsConfig, &quic.Config{
		MaxIdleTimeout:  time.Minute * 30,
		KeepAlivePeriod: time.Second * 15,
	})
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	a.quicConn = conn

	// Open control stream
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}

	// Send agent registration
	stream.Write([]byte(a.agentID))

	// Wait for confirmation
	buf := make([]byte, 2)
	_, err = stream.Read(buf)
	if err != nil || string(buf) != "OK" {
		return fmt.Errorf("registration failed: %w", err)
	}

	a.controlStream = stream
	a.logger.Info().Str("agent_id", a.agentID).Msg("✅ Registered with server")

	return nil
}

func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.controlStream.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				a.logger.Error().Err(err).Msg("Heartbeat failed")
				return
			}
			a.controlStream.Write([]byte("PING"))
		}
	}
}

func (a *Agent) handleSSHRequests() {
	for {
		stream, err := a.quicConn.AcceptStream(context.Background())
		if err != nil {
			a.logger.Error().Err(err).Msg("Failed to accept stream")
			continue
		}

		go a.handleSSHRequest(stream)
	}
}

func (a *Agent) handleSSHRequest(stream quic.Stream) {
	// Read request
	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		a.logger.Error().Err(err).Msg("Failed to read request")
		return
	}

	// Parse SSH request
	var req SSHSession
	if err := json.Unmarshal(buf[:n], &req); err != nil {
		a.logger.Error().Err(err).Msg("Failed to parse request")
		return
	}

	a.logger.Info().
		Str("host", req.Host).
		Int("port", req.Port).
		Str("user", req.User).
		Msg("🔐 New SSH request")

	// Connect to target SSH server
	sshConfig := &ssh.ClientConfig{
		User: req.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", req.Host, req.Port), sshConfig)
	if err != nil {
		a.logger.Error().Err(err).Msg("SSH dial failed")
		stream.Write([]byte("ERROR: " + err.Error()))
		return
	}

	defer sshClient.Close()

	// Open SSH session
	session, err := sshClient.NewSession()
	if err != nil {
		a.logger.Error().Err(err).Msg("SSH session failed")
		return
	}
	defer session.Close()

	// Setup PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		a.logger.Error().Err(err).Msg("PTY request failed")
		return
	}

	// Setup pipes
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		a.logger.Error().Err(err).Msg("Stdin pipe failed")
		return
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		a.logger.Error().Err(err).Msg("Stdout pipe failed")
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		a.logger.Error().Err(err).Msg("Shell start failed")
		return
	}

	// Send success
	stream.Write([]byte("OK"))

	// Bidirectional forwarding
	go func() {
		defer stream.Close()
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				return
			}
			stream.Write(buf[:n])
		}
	}()

	buf = make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			a.logger.Info().Msg("SSH stream closed")
			return
		}
		stdinPipe.Write(buf[:n])
	}
}
