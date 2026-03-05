package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AuditLogger struct {
	file   *os.File
	mu     sync.Mutex
	path   string
}

type AuditEvent struct {
	Timestamp int64                  `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
}

func NewAuditLogger(path string) (*AuditLogger, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{
		file: file,
		path: path,
	}, nil
}

func (a *AuditLogger) LogEvent(agentID, eventType string, data map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	event := AuditEvent{
		Timestamp: time.Now().Unix(),
		AgentID:   agentID,
		EventType: eventType,
		Data:      data,
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(a.file, string(bytes))
	if err != nil {
		return err
	}

	return a.file.Sync()
}

func (a *AuditLogger) Close() error {
	return a.file.Close()
}
