# QUIC Robot - WebSSH 系统需求文档

## 📋 项目概述

**项目名称：** quic_robot  
**项目类型：** 基于 QUIC 协议的 WebSSH 堡垒机系统  
**架构模式：** 前后端分离 + Agent-Server 反向连接  

---

## 🎯 核心功能

### 1. QUIC 传输层
- **协议优势：**
  - 多路复用（单连接多流）
  - 低延迟（UDP 优化）
  - 连接迁移（网络切换不掉线）
  - 内置 TLS 1.3 加密

- **实现要点：**
  - Server 监听 QUIC 端口（默认 443）
  - Agent 主动连接 Server（穿越 NAT/FW）
  - 双向流（Stream）管理：
    - 控制流：心跳、认证、命令
    - 数据流：SSH 会话数据
    - 录像流：实时审计数据传输

### 2. WebSSH 前端
**技术栈：** React + xterm.js + WebRTC/WebSocket (网关)

**核心功能：**
- 终端模拟器（xterm.js）
- 多标签页支持（同时多个 SSH 会话）
- 分屏布局（可选）
- 快捷键绑定（复制/粘贴/清屏）
- 主题切换（暗色/亮色/自定义）
- 字体大小调节

**交互特性：**
- 实时响应用户输入
- ANSI 颜色支持
- 滚动缓冲区管理
- 自动重连提示

### 3. 后端 Server
**技术栈：** Rust (quinn) 或 Go (quic-go)

**核心模块：**

#### 3.1 连接管理
```rust
// 伪代码示例
struct ConnectionManager {
    agents: HashMap<AgentId, AgentConnection>,
    sessions: HashMap<SessionId, SshSession>,
}

struct AgentConnection {
    agent_id: String,
    quic_conn: Connection,
    streams: Vec<Stream>,
    last_heartbeat: Instant,
}
```

**功能点：**
- Agent 注册与认证（Token/证书）
- 心跳检测（30s 超时）
- 会话路由（Web 请求 → Agent）
- 连接状态监控

#### 3.2 SSH 代理
- 接收 WebSSH 连接请求
- 通过 QUIC 流转发到 Agent
- 双向数据管道：`WebSocket ←→ QUIC Stream ←→ SSH`
- 窗口大小协商
- 环境变量传递

#### 3.3 审计系统
**审计日志：**
- 会话开始/结束时间
- 连接来源 IP
- 操作用户
- 目标主机信息
- 命令记录（每条命令+时间戳）

**录像功能：**
- **格式：** asciinema / JSON Line
- **内容：** 终端输出 + 时间戳 + 用户输入
- **存储：** 本地文件 / 对象存储（S3）
- **压缩：** gzip（可选）
- **回放：** Web 播放器（asciinema recorder）

**录像内容示例（JSONL）：**
```json
{"type": "input", "data": "ls -la\n", "time": 1.234}
{"type": "output", "data": "total 88\r\ndrwxr-xr+...", "time": 1.456}
{"type": "resize", "rows": 24, "cols": 80, "time": 5.678}
```

#### 3.4 API 接口
```http
GET  /api/agents              # 列出在线 Agents
GET  /api/sessions            # 列出活跃会话
POST /api/sessions            # 创建新会话
DELETE /api/sessions/:id      # 关闭会话
GET  /api/audits/:session_id  # 获取审计日志
GET  /api/videos/:session_id  # 获取录像文件
```

### 4. Agent（被控端）
**技术栈：** Rust / Go（跨平台二进制）

**核心功能：**

#### 4.1 反向连接
```rust
// Agent 启动时主动连接 Server
fn connect_to_server(server_addr: &str) -> Result<Connection> {
    let config = QuicConfig::new()
        .with_cert(agent_cert)
        .with_server_name(server_name);

    let conn = quinn::connect(server_addr, config)?;
    Ok(conn)
}
```

**特性：**
- 启动自动连接
- 断线重连（指数退避：1s, 2s, 4s... 最大 60s）
- 心跳保活
- 连接迁移（网络切换自动重连）

#### 4.2 SSH 客户端
- 接收 Server 转发的 SSH 连接请求
- 建立本地 SSH 连接到目标主机
- 数据转发：`QUIC Stream ←→ SSH Socket`
- 支持：
  - 密码认证
  - 密钥认证
  - Agent 转发
  - SFTP 子系统（可选）

#### 4.3 本地管理
```bash
# Agent CLI
quic-agent start --server=wss://server.example.com --token=xxx
quic-agent status
quic-agent logs
quic-agent stop
```

**配置文件：**
```yaml
server:
  url: "wss://server.example.com"
  token: "your-auth-token"
  reconnect_interval: 5s

ssh:
  default_user: "root"
  private_key_path: "~/.ssh/id_rsa"

recording:
  enabled: true
  path: "/var/log/quic-agent/sessions"
```

---

## 🔐 安全设计

### 认证与授权
- **Agent 认证：** Token + 双向 TLS（mTLS）
- **用户认证：** JWT / OAuth2 / LDAP（可选）
- **权限控制：**
  - 用户只能访问授权的 Agent
  - Agent 只能连接授权的目标主机

### 数据安全
- QUIC 内置 TLS 1.3 加密
- 敏感信息（Token/密钥）不在日志中出现
- 录像文件存储加密（可选）

### 审计合规
- 所有操作记录不可篡改（日志签名）
- 录像文件完整性校验（SHA256）
- 符合等保三级要求（可选）

---

## 🏗️ 系统架构

```
┌─────────────┐         QUIC (UDP)          ┌─────────────┐
│             │  ◄─────────────────────────►  │             │
│  Web Browser│                              │  QUIC Agent │
│             │  WebSocket  │                 │ (被控端)    │
└──────┬──────┘             │                 └──────┬──────┘
       │                    │                        │
       │ HTTP/WebSocket     │ QUIC Streams           │ SSH
       ▼                    ▼                        ▼
┌─────────────┐    ┌─────────────────┐    ┌─────────────┐
│  Frontend   │    │  QUIC Server    │    │  Target     │
│  (React)    │    │  (Rust/Go)      │    │  Hosts      │
└─────────────┘    └─────────────────┘    └─────────────┘
                          │
                          ▼
                   ┌──────────────┐
                   │  PostgreSQL  │
                   │  / SQLite    │
                   └──────────────┘
```

### 数据流

1. **Agent 注册：**
   ```
   Agent → Server: CONNECT (AgentID, Token)
   Server → Agent: ACCEPT (SessionID)
   ```

2. **SSH 会话建立：**
   ```
   Web → Server: CREATE_SESSION (AgentID, TargetHost, User)
   Server → Agent: SSH_REQUEST (SessionID, Host, Port, User)
   Agent → Server: SSH_ACCEPTED (SessionID)
   Server → Web: SESSION_READY (SessionID)
   ```

3. **数据传输：**
   ```
   Web → Server: WebSocket (Input)
   Server → Agent: QUIC Stream (Input)
   Agent → Target: SSH Socket (Input)
   Target → Agent: SSH Socket (Output)
   Agent → Server: QUIC Stream (Output)
   Server → Web: WebSocket (Output)
   ```

4. **录像：**
   ```
   Agent → Server: QUIC Stream (Raw Output)
   Server → Storage: Write to File/DB
   ```

---

## 📦 项目结构

```
quic_robot/
├── server/                 # QUIC Server (Rust)
│   ├── Cargo.toml
│   ├── src/
│   │   ├── main.rs
│   │   ├── quic/           # QUIC 连接管理
│   │   ├── api/            # REST API
│   │   ├── ssh/            # SSH 代理
│   │   ├── audit/          # 审计录像
│   │   └── db/             # 数据库
│   └── certs/              # TLS 证书
│
├── agent/                  # QUIC Agent (Rust)
│   ├── Cargo.toml
│   ├── src/
│   │   ├── main.rs
│   │   ├── quic/           # QUIC 客户端
│   │   ├── ssh/            # SSH 客户端
│   │   ├── reconnect/      # 重连逻辑
│   │   └── config/         # 配置管理
│   └── certs/
│
├── web/                    # Frontend (React)
│   ├── package.json
│   ├── src/
│   │   ├── components/
│   │   │   ├── Terminal.tsx
│   │   │   ├── SessionList.tsx
│   │   │   └── VideoPlayer.tsx
│   │   ├── api/
│   │   │   └── client.ts
│   │   └── App.tsx
│   └── public/
│
└── docs/
    ├── API.md
    ├── DEPLOY.md
    └── ARCHITECTURE.md
```

---

## 🚀 实施计划

### Phase 1: 核心功能（MVP）
1. ✅ QUIC 连接（Agent ↔ Server）
2. ✅ SSH 数据转发
3. ✅ 基础 WebSSH（单会话）
4. ✅ 简单审计日志

### Phase 2: 增强功能
1. ⬜ 会话录像（asciinema 格式）
2. ⬜ 多标签页支持
3. ⬜ Agent 重连与保持
4. ⬜ 用户认证（JWT）

### Phase 3: 高级功能
1. ⬜ 审计回放器
2. ⬜ 权限管理系统
3. ⬜ 监控与告警
4. ⬜ SFTP 文件传输

### Phase 4: 生产就绪
1. ⬜ 性能优化
2. ⬜ 压力测试
3. ⬜ 文档完善
4. ⬜ Docker 部署

---

## 🎨 技术选型建议

| 组件 | 推荐技术 | 理由 |
|------|---------|------|
| QUIC 库 | **quinn** (Rust) / **quic-go** (Go) | 成熟稳定，文档完善 |
| SSH 库 | **russh** (Rust) / **ssh2** (Go) | 异步支持好 |
| 前端框架 | **React + TypeScript** | 生态丰富 |
| 终端组件 | **xterm.js** | 功能强大，易集成 |
| 数据库 | **PostgreSQL** (生产) / **SQLite** (开发) | 灵活选择 |
| 视频录制 | **asciinema** format | 轻量，易回放 |

---

## ❓ 待确认问题

1. **技术栈选择：** Rust 还是 Go？（Rust 性能更好，Go 开发更快）
2. **认证方式：** Token 还是 mTLS？（建议两者结合）
3. **录像存储：** 本地文件还是对象存储？（影响扩展性）
4. **并发规模：** 预期支持多少并发会话？（影响架构设计）
5. **部署环境：** Docker / K8s / 二进制？（影响打包方式）

---

## ✅ 下一步行动

等你确认需求无误后，我将：

1. 在 GitHub 创建 `quic_robot` 仓库
2. 初始化项目结构（Rust + React）
3. 实现 Phase 1 核心功能
4. 编写部署文档
5. 提供 Demo 演示

---

**有任何疑问或需要调整的地方，随时告诉我！** 🌫️
