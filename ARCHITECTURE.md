# QUIC Robot - 系统架构设计

## 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户浏览器                                │
│                    (React + xterm.js)                            │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTPS / WSS
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Nginx / Caddy                               │
│                    (反向代理 + SSL终结)                           │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     QUIC Server (Rust)                           │
│  ┌─────────────┬─────────────┬─────────────┬──────────────┐    │
│  │ API Server  │ Session Mgr │ SSH Proxy   │ Audit/Record │    │
│  │  (REST)     │  (内存)     │  (QUIC)     │   (存储)      │    │
│  └─────────────┴─────────────┴─────────────┴──────────────┘    │
│                          │                                       │
│                          │ QUIC (UDP 443)                        │
│                          │                                       │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     QUIC Agent (被控端)                           │
│  ┌─────────────┬─────────────┬─────────────┬──────────────┐    │
│  │ QUIC Client │ SSH Client  │ Reconnect   │ Config Mgr   │    │
│  │  (反向连接)  │  (本地连接)  │  (心跳/重连) │              │    │
│  └─────────────┴─────────────┴─────────────┴──────────────┘    │
│                          │                                       │
│                          │ SSH (TCP 22)                          │
│                          │                                       │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            ▼
                   ┌────────────────┐
                   │  目标主机       │
                   │  (Linux Server)│
                   └────────────────┘
```

---

## 模块详细设计

### 1. QUIC Server 模块

#### 1.1 连接管理器（ConnectionManager）

```rust
pub struct ConnectionManager {
    agents: HashMap<AgentId, AgentConnection>,
    sessions: HashMap<SessionId, SessionContext>,
    next_session_id: Arc<AtomicU64>,
}

pub struct AgentConnection {
    agent_id: AgentId,
    quic_conn: Connection,
    control_stream: Option<SendStream>,
    last_heartbeat: Instant,
    metadata: AgentMetadata,
}

pub struct SessionContext {
    session_id: SessionId,
    agent_id: AgentId,
    ssh_stream: Option<SendStream>,
    websocket: Option<WebSocketSink>,
    started_at: Instant,
    recorder: Option<Box<dyn Recorder>>,
}
```

**职责：**
- 接受 Agent 连接（反向连接）
- 维护 Agent 在线状态
- 分配会话 ID
- 路由 WebSocket ↔ SSH 数据
- 心跳检测（30s 超时）

#### 1.2 API 服务器（ApiServer）

```rust
pub struct ApiServer {
    db: Arc<dyn Database>,
    conn_mgr: Arc<Mutex<ConnectionManager>>,
}

impl ApiServer {
    // 列出在线 Agents
    pub async fn list_agents(&self) -> Result<Vec<AgentInfo>>;

    // 创建 SSH 会话
    pub async fn create_session(&self, req: CreateSessionRequest) -> Result<SessionId>;

    // 关闭会话
    pub async fn close_session(&self, session_id: SessionId) -> Result<()>;

    // 获取审计日志
    pub async fn get_audit_log(&self, session_id: SessionId) -> Result<AuditLog>;

    // 获取录像文件
    pub async fn get_recording(&self, session_id: SessionId) -> Result<Recording>;
}
```

#### 1.3 SSH 代理（SshProxy）

```rust
pub struct SshProxy {
    conn_mgr: Arc<Mutex<ConnectionManager>>,
}

impl SshProxy {
    // 处理 WebSocket 连接
    pub async fn handle_websocket(&self, ws: WebSocket, session_id: SessionId);

    // 转发数据：WebSocket → QUIC Stream
    async fn forward_to_agent(&self, ws: &mut WebSocket, stream: &mut SendStream);

    // 转发数据：QUIC Stream → WebSocket
    async fn forward_to_web(&self, stream: &mut RecvStream, ws: &mut WebSocket);
}
```

#### 1.4 审计录像（AuditRecorder）

```rust
pub trait Recorder {
    fn record_input(&mut self, data: &[u8], timestamp: f64);
    fn record_output(&mut self, data: &[u8], timestamp: f64);
    fn record_resize(&mut self, rows: u16, cols: u16, timestamp: f64);
    fn finish(self: Box<Self>) -> Result<RecordingMetadata>;
}

pub struct AsciiinemaRecorder {
    session_id: SessionId,
    file: BufWriter<File>,
    start_time: Instant,
    width: u16,
    height: u16,
}
```

**录像格式（asciinema）：**
```json
{"version": 2, "width": 80, "height": 24, "timestamp": 1234567890.123}
[1.234, "o", "Welcome to Ubuntu\r\n"]
[1.456, "i", "ls -la\r\n"]
[1.789, "o", "total 88\r\ndrwxr-xr-x..."]
```

---

### 2. QUIC Agent 模块

#### 2.1 QUIC 客户端（QuicClient）

```rust
pub struct QuicClient {
    server_url: String,
    agent_id: AgentId,
    token: String,
    conn: Option<Connection>,
    reconnect_config: ReconnectConfig,
}

pub struct ReconnectConfig {
    initial_interval: Duration,  // 1s
    max_interval: Duration,      // 60s
    multiplier: f64,             // 2.0 (指数退避)
}

impl QuicClient {
    // 连接到服务器（带重试）
    pub async fn connect_with_retry(&mut self) -> Result<()> {
        let mut interval = self.reconnect_config.initial_interval;

        loop {
            match self.try_connect().await {
                Ok(conn) => {
                    self.conn = Some(conn);
                    return Ok(());
                }
                Err(e) => {
                    log::warn!("连接失败: {}, {} 秒后重试...", e, interval.as_secs());
                    tokio::time::sleep(interval).await;
                    interval = std::cmp::min(
                        Duration::from_secs_f64(interval.as_secs_f64() * self.reconnect_config.multiplier),
                        self.reconnect_config.max_interval,
                    );
                }
            }
        }
    }

    // 心跳保活
    pub async fn heartbeat_loop(&self) {
        let mut interval = tokio::time::interval(Duration::from_secs(15));

        loop {
            interval.tick().await;
            if let Err(e) = self.send_heartbeat().await {
                log::error!("心跳失败: {}", e);
                break;
            }
        }
    }
}
```

#### 2.2 SSH 客户端（SshClient）

```rust
pub struct SshClient {
    host: String,
    port: u16,
    user: String,
    auth: AuthMethod,
}

pub enum AuthMethod {
    Password(String),
    PublicKey { key_path: PathBuf, passphrase: Option<String> },
    Agent,
}

impl SshClient {
    // 建立本地 SSH 连接
    pub async fn connect(&self) -> Result<Session>;

    // 转发数据：QUIC Stream ↔ SSH
    pub async fn forward(&self, mut quic_stream: (SendStream, RecvStream)) -> Result<()> {
        let mut session = self.connect().await?;
        let mut channel = session.channel_open_session().await?;

        // 启动双向转发
        tokio::select! {
            res = self.forward_quic_to_ssh(&mut quic_stream.1, &mut channel) => res,
            res = self.forward_ssh_to_quic(&mut channel, &mut quic_stream.0) => res,
        }
    }
}
```

---

### 3. 前端架构

#### 3.1 组件结构

```
src/
├── App.tsx                      # 主应用
├── components/
│   ├── Terminal.tsx             # 终端组件
│   ├── SessionList.tsx          # 会话列表
│   ├── VideoPlayer.tsx          # 录像回放器
│   └── ConnectionStatus.tsx     # 连接状态指示
├── api/
│   └── client.ts                # API 客户端
├── hooks/
│   ├── useWebSocket.ts          # WebSocket Hook
│   └── useTerminal.ts           # xterm.js Hook
└── types/
    └── index.ts                 # TypeScript 类型定义
```

#### 3.2 Terminal 组件（核心）

```typescript
interface TerminalProps {
  sessionId: string;
  onResize?: (rows: number, cols: number) => void;
}

export const Terminal: React.FC<TerminalProps> = ({ sessionId, onResize }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const [xterm, setXterm] = useState<Terminal | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    // 初始化 xterm.js
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Monaco, "Courier New", monospace',
      theme: { background: '#1e1e1e', foreground: '#f0f0f0' },
    });

    if (terminalRef.current) {
      term.open(terminalRef.current);
    }

    // 连接 WebSocket
    const websocket = new WebSocket(`wss://server.com/api/sessions/${sessionId}`);
    websocket.binaryType = 'arraybuffer';

    websocket.onopen = () => {
      // 处理终端输入
      term.onData((data) => {
        websocket.send(JSON.stringify({ type: 'input', data }));
      });

      // 处理窗口大小变化
      term.onResize((size) => {
        websocket.send(JSON.stringify({
          type: 'resize',
          rows: size.rows,
          cols: size.cols,
        }));
        onResize?.(size.rows, size.cols);
      });
    };

    websocket.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (msg.type === 'output') {
        term.write(msg.data);
      }
    };

    setXterm(term);
    setWs(websocket);

    return () => {
      websocket.close();
      term.dispose();
    };
  }, [sessionId]);

  return <div ref={terminalRef} className="terminal-container" />;
};
```

---

## 数据库设计

### 表结构（PostgreSQL）

```sql
-- Agents 表
CREATE TABLE agents (
    agent_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    last_heartbeat TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB
);

-- Sessions 表
CREATE TABLE sessions (
    session_id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(64) REFERENCES agents(agent_id),
    user_id VARCHAR(64),
    target_host VARCHAR(255) NOT NULL,
    target_port INTEGER DEFAULT 22,
    started_at TIMESTAMP DEFAULT NOW(),
    ended_at TIMESTAMP,
    status VARCHAR(32) DEFAULT 'active',
    recording_path TEXT
);

-- Audit Logs 表
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) REFERENCES sessions(session_id),
    event_type VARCHAR(32) NOT NULL,
    timestamp TIMESTAMP DEFAULT NOW(),
    data JSONB
);

-- 录像文件索引（可选，如果用对象存储）
CREATE TABLE recordings (
    session_id VARCHAR(64) PRIMARY KEY REFERENCES sessions(session_id),
    file_path TEXT NOT NULL,
    file_size BIGINT,
    duration FLOAT,
    format VARCHAR(32) DEFAULT 'asciinema',
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 部署架构

### 开发环境
```
┌─────────────┐
│  localhost  │
│             │  ┌────────────────┐
│  Browser    │─►│  QUIC Server   │
│             │  │  (:443)        │
└─────────────┘  └────────┬───────┘
                           │
                           │ QUIC
                           ▼
                    ┌──────────────┐
                    │  Local Agent │
                    │  (localhost) │
                    └──────────────┘
```

### 生产环境（K8s）
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: quic-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: quic-server
  template:
    metadata:
      labels:
        app: quic-server
    spec:
      containers:
      - name: server
        image: quic-robot/server:latest
        ports:
        - containerPort: 443
          protocol: UDP
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: url
---
apiVersion: v1
kind: Service
metadata:
  name: quic-server
spec:
  type: LoadBalancer
  ports:
  - port: 443
    targetPort: 443
    protocol: UDP
  selector:
    app: quic-server
```

---

## 性能指标

### 预期性能
- **单 Agent 并发会话：** 100+
- **Server 总并发：** 10,000+ (水平扩展)
- **延迟：** <50ms (局域网), <200ms (跨地域)
- **吞吐量：** 1 Gbps+ (单连接)
- **内存占用：** Agent ~50MB, Server ~200MB (1000 会话)

### 优化点
1. **零拷贝：** 使用 `bytes::Bytes` 避免数据复制
2. **连接池：** 复用 SSH 连接（可选）
3. **压缩：** QUIC 流压缩（可选）
4. **缓存：** 静态资源 CDN

---

这个架构设计能满足你的需求吗？有需要调整的地方告诉我 🌫️
