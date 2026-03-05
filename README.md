# QUIC Robot 🤖

> 基于 QUIC 协议的 WebSSH 堡垒机系统（Go 实现）

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## ✨ 特性

- 🚀 **QUIC 传输** - 低延迟、多路复用、连接迁移
- 🔐 **端到端加密** - TLS 1.3 + mTLS 双向认证
- 📹 **审计录像** - asciinema 格式，精确回放
- 🔄 **会话保持** - 自动重连，网络切换不掉线
- 🌐 **反向连接** - Agent 主动连接 Server，穿越 NAT
- 📊 **完整审计** - 操作日志 + 录像 + 时间戳

## 📋 文档

- [产品需求文档 (PRD)](./PRD.md) - 功能需求和技术选型
- [系统架构设计 (ARCHITECTURE)](./ARCHITECTURE.md) - 详细架构和模块设计
- [常见问题 (FAQ)](./FAQ.md) - 技术选型和实现细节

## 🏗️ 项目结构

```
quic_robot/
├── cmd/
│   ├── server/          # QUIC Server 入口
│   └── agent/           # QUIC Agent 入口
├── internal/
│   ├── server/          # Server 核心逻辑
│   │   ├── quic/        # QUIC 连接管理
│   │   ├── api/         # REST API
│   │   ├── ssh/         # SSH 代理
│   │   └── audit/       # 审计录像
│   └── agent/           # Agent 核心逻辑
│       ├── quic/        # QUIC 客户端
│       ├── ssh/         # SSH 客户端
│       └── reconnect/   # 重连逻辑
├── web/                 # 前端（React + xterm.js）
├── pkg/                 # 公共库
├── docs/                # 详细文档
└── deployments/         # 部署配置
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+ (前端)
- PostgreSQL 14+ (可选，开发可用 SQLite)

### 启动 Server

```bash
# 克隆仓库
git clone https://github.com/voilet/quic_robot.git
cd quic_robot

# 启动 Server
go run cmd/server/main.go --config config/server.yaml
```

### 启动 Agent

```bash
# 在目标机器上
go run cmd/agent/main.go --config config/agent.yaml
```

### 前端访问

```bash
cd web
npm install
npm run dev
```

访问 http://localhost:5173

## 🔐 安全特性

- ✅ TLS 1.3 强制加密
- ✅ mTLS 双向认证（可选）
- ✅ Token + IP 白名单
- ✅ 录像文件完整性校验（SHA256）
- ✅ 审计日志不可篡改

## 📊 性能指标

- 单 Agent 并发会话: 100+
- Server 总并发: 10,000+ (水平扩展)
- 延迟: <50ms (局域网), <200ms (跨地域)
- 内存占用: Agent ~50MB, Server ~200MB (1000 会话)

## 🗺️ 路线图

### Phase 1: 核心功能（MVP）
- [ ] QUIC 连接（Agent ↔ Server）
- [ ] SSH 数据转发
- [ ] 基础 WebSSH（单会话）
- [ ] 简单审计日志

### Phase 2: 增强功能
- [ ] 会话录像（asciinema 格式）
- [ ] 多标签页支持
- [ ] Agent 重连与保持
- [ ] 用户认证（JWT）

### Phase 3: 高级功能
- [ ] 审计回放器
- [ ] 权限管理系统
- [ ] 监控与告警
- [ ] SFTP 文件传输

### Phase 4: 生产就绪
- [ ] 性能优化
- [ ] 压力测试
- [ ] 文档完善
- [ ] Docker 部署

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

---

**⚠️ 当前状态：** 需求分析和架构设计阶段，等待确认后开始开发 🌫️
