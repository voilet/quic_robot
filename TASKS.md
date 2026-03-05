# QUIC Robot - 任务清单

## Phase 1: MVP 核心功能

### ✅ 已完成

- [x] 需求分析与文档编写
  - [x] PRD.md - 产品需求文档
  - [x] ARCHITECTURE.md - 系统架构设计
  - [x] FAQ.md - 常见问题解答
  - [x] README.md - 项目说明

- [x] 项目初始化
  - [x] GitHub 仓库创建
  - [x] Go 项目结构搭建（cmd/, internal/, web/）
  - [x] 依赖配置（go.mod, package.json）

- [x] QUIC Server 实现
  - [x] QUIC 监听器（TLS 配置）
  - [x] Agent 注册与认证
  - [x] 心跳检测机制
  - [x] WebSocket 网关（:8080）
  - [x] 审计日志系统

- [x] QUIC Agent 实现
  - [x] 反向连接到 Server
  - [x] 自动重连机制
  - [x] SSH 客户端（golang.org/x/crypto/ssh）
  - [x] 双向数据转发（QUIC ↔ SSH）

- [x] WebSSH 前端
  - [x] React + TypeScript 项目
  - [x] xterm.js 终端集成
  - [x] WebSocket 客户端
  - [x] 暗色主题 UI
  - [x] 连接表单

- [x] 开发工具
  - [x] 自签名证书生成脚本
  - [x] 一键初始化脚本
  - [x] .gitignore 配置
  - [x] 开发文档（DEVELOPMENT.md）

### 🚧 进行中

- [ ] 端到端测试
  - [ ] Server 启动测试
  - [ ] Agent 连接测试
  - [ ] WebSSH 界面测试
  - [ ] SSH 会话建立测试
  - [ ] 数据转发验证

### 📋 待完成

- [ ] WebSocket ↔ QUIC 双向流实现
  - [ ] 处理 WebSocket 输入 → QUIC Stream
  - [ ] 处理 QUIC Stream 输出 → WebSocket
  - [ ] 会话 ID 管理
  - [ ] 多会话支持

- [ ] 会话录像功能
  - [ ] asciinema 格式记录
  - [ ] 时间戳同步
  - [ ] 文件存储
  - [ ] 回放器（可选）

- [ ] 用户认证
  - [ ] JWT Token 生成/验证
  - [ ] 登录接口
  - [ ] 权限控制

- [ ] Agent 会话保持
  - [ ] 断线重连优化
  - [ ] 连接迁移支持
  - [ ] 心跳超时处理

- [ ] 生产部署
  - [ ] Docker 镜像
  - [ ] K8s 部署配置
  - [ ] 性能测试
  - [ ] 压力测试

---

## Phase 2: 增强功能

- [ ] 会话录像（asciinema 格式）
- [ ] 多标签页支持
- [ ] Agent 重连与保持增强
- [ ] 用户认证（JWT）
- [ ] 审计日志查看器
- [ ] Agent 状态监控面板

## Phase 3: 高级功能

- [ ] 审计回放器
- [ ] 权限管理系统
- [ ] 监控与告警
- [ ] SFTP 文件传输
- [ ] 数据库集成（PostgreSQL）

## Phase 4: 生产就绪

- [ ] 性能优化
- [ ] 压力测试
- [ ] 文档完善
- [ ] Docker 部署
- [ ] CI/CD 流水线

---

**最后更新：** 2026-03-05 16:38
