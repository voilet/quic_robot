import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import './App.css';

function App() {
  const terminalRef = useRef<HTMLDivElement>(null);
  const [connected, setConnected] = useState(false);
  const [agentId, setAgentId] = useState('agent-1');
  const [host, setHost] = useState('localhost');
  const [port, setPort] = useState('22');
  const [user, setUser] = useState('root');
  const [password, setPassword] = useState('');

  useEffect(() => {
    if (!connected || !terminalRef.current) return;

    // Initialize terminal
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#f0f0f0',
        cursor: '#ffffff',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(terminalRef.current);
    fitAddon.fit();

    term.writeln('🤖 QUIC Robot WebSSH');
    term.writeln('Connecting to agent...');

    // Connect to backend
    const ws = new WebSocket(`ws://localhost:8080/ws`);

    ws.onopen = () => {
      term.writeln('\r\n✅ Connected to server');

      // Send SSH request
      ws.send(
        JSON.stringify({
          type: 'connect',
          agentId,
          host,
          port: parseInt(port),
          user,
          password,
        })
      );
    };

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (msg.type === 'output') {
        term.write(msg.data);
      } else if (msg.type === 'error') {
        term.writeln(`\r\n❌ Error: ${msg.message}`);
      }
    };

    ws.onerror = (error) => {
      term.writeln('\r\n❌ WebSocket error');
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      term.writeln('\r\n❌ Connection closed');
      setConnected(false);
    };

    // Handle user input
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            type: 'input',
            data,
          })
        );
      }
    });

    // Handle resize
    window.addEventListener('resize', () => fitAddon.fit());

    return () => {
      ws.close();
      term.dispose();
      window.removeEventListener('resize', () => fitAddon.fit());
    };
  }, [connected, agentId, host, port, user, password]);

  const handleConnect = () => {
    setConnected(true);
  };

  const handleDisconnect = () => {
    setConnected(false);
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>🤖 QUIC Robot WebSSH</h1>
        {!connected && (
          <div className="connection-form">
            <input
              type="text"
              placeholder="Agent ID"
              value={agentId}
              onChange={(e) => setAgentId(e.target.value)}
            />
            <input
              type="text"
              placeholder="Host"
              value={host}
              onChange={(e) => setHost(e.target.value)}
            />
            <input
              type="text"
              placeholder="Port"
              value={port}
              onChange={(e) => setPort(e.target.value)}
            />
            <input
              type="text"
              placeholder="User"
              value={user}
              onChange={(e) => setUser(e.target.value)}
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <button onClick={handleConnect}>Connect</button>
          </div>
        )}
        {connected && (
          <button onClick={handleDisconnect} className="disconnect-btn">
            Disconnect
          </button>
        )}
      </header>
      <div className="terminal-container">
        <div ref={terminalRef} className="terminal" />
      </div>
    </div>
  );
}

export default App;
