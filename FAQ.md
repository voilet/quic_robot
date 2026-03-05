# QUIC Robot - 常见问题（FAQ）

## 关于需求

### Q1: 为什么选择 QUIC 而不是 WebSocket？
**A:** QUIC 有几个关键优势：
- **多路复用：** 单连接支持多个独立流，避免队头阻塞
- **低延迟：** 基于 UDP，减少 RTT（1-RTT 握手）
- **连接迁移：** 网络切换（WiFi ↔ 4G）不掉线
- **内置安全：** TLS 1.3 强制加密

对于远程 SSH 场景，网络稳定性至关重要，QUIC 的连接迁移特性可以显著提升体验。

### Q2: Agent 为什么是反向连接？
**A:** 反向连接（Agent 连 Server）的优势：
- **穿越 NAT：** Agent 在内网也能连接
- **统一管理：** Server 只需开放一个端口
- **动态注册：** Agent 上线自动发现

如果用 Server 连 Agent，需要每个 Agent 有公网 IP 或端口映射，运维成本高。

### Q3: 录像为什么用 asciinema 格式？
**A:** asciinema 的优势：
- **轻量：** 纯文本 JSON，体积小（1小时会话 ~几 MB）
- **精确：** 记录时间戳，可精确回放
- **可搜索：** 文本格式，可用 grep 搜索命令
- **易集成：** 有现成的 JavaScript 播放器

相比视频录制（MP4），asciinema 更适合审计场景。

---

## 技术选型

### Q4: Rust 还是 Go？
**A:**

| 方面 | Rust | Go |
|------|------|-----|
| 性能 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 内存安全 | 编译时保证 | GC（有开销） |
| 并发模型 | async/await | Goroutine |
| 开发速度 | 较慢（学习曲线陡） | 快（语法简单） |
| 生态 | QUIC: quinn, SSH: russh | QUIC: quic-go, SSH: ssh2 |

**建议：**
- 如果追求极致性能 → **Rust**
- 如果快速开发 → **Go**

我推荐 **Rust**，因为：
1. QUIC 库 `quinn` 非常成熟
2. 异步运行时 `tokio` 性能强
3. 内存占用低（适合大规模部署）

### Q5: 为什么不用 OpenSSH？
**A:** OpenSSH 是 C 实现的，集成到 QUIC 系统需要：
- 用 C FFI 调用（复杂）
- 或进程通信（开销大）

用 Rust SSH 库（`russh`）可以：
- 原生集成到异步运行时
- 零拷贝数据转发
- 更容易控制连接生命周期

---

## 功能细节

### Q6: 会话保持和重连如何实现？
**A:** 三个层面的重连：

#### 1. Agent → Server（QUIC 连接）
```rust
// 指数退避重连
loop {
    match connect_to_server().await {
        Ok(conn) => {
            // 连接成功，启动心跳
            heartbeat_loop(conn).await;
        }
        Err(e) => {
            log::error!("连接失败: {}", e);
            tokio::time::sleep(reconnect_delay).await;
            reconnect_delay *= 2;  // 1s, 2s, 4s, ..., 最大 60s
        }
    }
}
```

#### 2. Server → Web（WebSocket）
```javascript
// 前端自动重连
const connect = () => {
  ws = new WebSocket(url);
  ws.onclose = () => {
    setTimeout(connect, 3000);  // 3秒后重连
  };
};
```

#### 3. Agent → 目标主机（SSH）
- 如果 SSH 断了，Agent 自动重连
- 记录断线时刻，重新连接后恢复会话（可选）

### Q7: 录像文件存储在哪里？
**A:** 两种方案：

#### 方案1: 本地存储（开发/小规模）
```
/var/log/quic-robot/recordings/
├── 2026-03-05/
│   ├── session-abc123.cast
│   └── session-def456.cast
```

#### 方案2: 对象存储（生产环境）
- **AWS S3**
- **阿里云 OSS**
- **MinIO**（自托管）

存储元数据在数据库：
```sql
INSERT INTO recordings (session_id, s3_key, file_size)
VALUES ('abc123', 'recordings/2026-03-05/abc123.cast', 1024000);
```

### Q8: 如何防止录像文件被篡改？
**A:** 三个措施：

1. **写后即读校验：**
   ```rust
   async fn write_and_verify(&mut self, data: &[u8]) -> Result<()> {
       self.file.write_all(data).await?;
       self.file.flush().await?;

       // 计算哈希
       let hash = sha256(data);

       // 存储到数据库
       self.db.save_hash(self.session_id, hash).await?;

       Ok(())
   }
   ```

2. **文件签名：**
   ```rust
   // 会话结束时生成签名
   let signature = sign_data(private_key, file_content);
   db.save_signature(session_id, signature).await?;
   ```

3. **只读权限：**
   - 录像文件目录设置为只读
   - 回放时从只读副本读取

### Q9: 多个用户同时连接同一个 Agent 怎么办？
**A:** 支持！每个 WebSocket 连接对应一个独立的 SSH 会话：

```
User A → Server → Agent → SSH Session 1 → Target Host:22
User B → Server → Agent → SSH Session 2 → Target Host:22
```

Agent 端为每个会话创建独立的 SSH 连接，互不干扰。

---

## 安全问题

### Q10: Token 认证安全吗？
**A:** 单 Token 不够，推荐多层防护：

1. **Token + mTLS：**
   ```
   Token: 用于 API 认证
   mTLS: 证书双向验证，防止中间人攻击
   ```

2. **Token 轮换：**
   ```rust
   // 每 30 天生成新 Token
   if token_expired() {
       new_token = generate_token();
       send_to_agent(new_token);
   }
   ```

3. **IP 白名单（可选）：**
   ```rust
   if !whitelist.contains(agent_ip) {
       return Err("Unauthorized IP");
   }
   ```

### Q11: 数据加密了吗？
**A:** 三层加密：

1. **传输层：** QUIC 内置 TLS 1.3
2. **存储层：** 录像文件可选 AES-256 加密
3. **认证层：** Token + mTLS

即使攻击者截获网络包，也无法解密内容。

### Q12: 符合等保要求吗？
**A:** 等保三级核心要求：

| 要求 | 实现 |
|------|------|
| 身份鉴别 | Token + mTLS + 可选 LDAP |
| 访问控制 | 用户-Agent 权限绑定 |
| 安全审计 | 完整日志 + 录像 |
| 数据完整性 | 录像文件签名 + 哈希校验 |
| 数据保密性 | TLS 1.3 传输加密 |

如果需要更高等级（等保四级），可以增加：
- 硬件密钥（HSM）
- 多因素认证（MFA）
- 数据库加密存储

---

## 部署运维

### Q13: 如何监控 Agent 状态？
**A:** 多种方式：

1. **心跳监控：**
   ```rust
   // Server 端
   if now - agent.last_heartbeat > Duration::from_secs(30) {
           notify_admin(agent_id, "Agent 离线");
       }
   }
   ```

2. **Prometheus 指标：**
   ```rust
   // 暴露指标
   gauges!("agents_online").set(online_count);
   gauges!("active_sessions").set(session_count);
   histograms!("session_duration").observe(duration);
   ```

3. **告警通知：**
   - Email
   - Webhook（钉钉/Slack）
   - 短信

### Q14: 如何水平扩展 Server？
**A:** QUIC 连接需要粘性会话（Session Affinity）：

#### 方案1: 一致性哈希
```rust
// 根据 Agent ID 选择 Server 实例
let server_idx = hash(agent_id) % server_count;
route_to_server(agent_id, server_idx);
```

#### 方案2: LoadBalancer 配置
```nginx
# Nginx 配置
upstream quic_backend {
    ip_hash;  # 根据 Client IP 分配
    server backend1:443;
    server backend2:443;
    server backend3:443;
}
```

### Q15: Agent 升级怎么办？
**A:** 滚动升级策略：

1. **灰度发布：**
   ```bash
   # 先升级 10% 的 Agent
   for agent in $(agents | shuf | head -n 10); do
       ssh $agent "upgrade-agent.sh"
   done
   ```

2. **版本兼容：**
   ```rust
   // Agent 注册时上报版本
   struct AgentHandshake {
       agent_id: String,
       version: String,  // "1.2.3"
       capabilities: Vec<String>,
   }

   // Server 检查版本
   if !compatible(agent.version) {
       return Err("请升级 Agent 到 1.2.0+");
   }
   ```

3. **自动回滚：**
   ```bash
   # 升级失败自动回退
   if !health_check; then
       rollback-agent
   fi
   ```

---

## 性能问题

### Q16: 支持 10,000 并发需要什么配置？
**A:** 参考配置：

**服务器：**
- CPU: 16 核
- 内存: 32 GB
- 网络: 10 Gbps
- 存储: SSD (录像写入)

**优化：**
```rust
// Tokio 运行时配置
let runtime = tokio::runtime::Builder::new_multi_thread()
    .worker_threads(16)           // CPU 核心数
    .thread_stack_size(256 * 1024)
    .enable_io()
    .enable_time()
    .build()?;
```

**数据库：**
- 连接池: 100
- 慢查询日志: 开启
- 索引: session_id, agent_id

### Q17: 录像写入会阻塞 SSH 转发吗？
**A:** 不会，异步写入：

```rust
// 使用 channel 解耦
let (recorder_tx, recorder_rx) = mpsc::channel(1000);

// SSH 转发线程
tokio::spawn(async move {
    loop {
        let data = ssh_stream.read().await?;
        websocket.send(data).await?;  // 非阻塞

        // 发送到录像任务（不阻塞）
        recorder_tx.send(data.clone()).await.ok();
    }
});

// 录像写入线程（独立任务）
tokio::spawn(async move {
    while let Some(data) = recorder_rx.recv().await {
        recorder.write(data).await?;  // 异步写入
    }
});
```

---

## 其他问题

### Q18: 能集成现有堡垒机吗？
**A:** 可以，提供 API 兼容层：

```rust
// 兼容 JumpServer API
pub struct JumpServerAdapter {
    client: Reqwest,
}

impl JumpServerAdapter {
    pub async fn create_session(&self, user: &str, host: &str) -> Result<SessionId> {
        // 调用 JumpServer API
        self.client.post("/api/v1/sessions")
            .json(&json!({"user": user, "host": host}))
            .send()
            .await?
    }
}
```

### Q19: 支持 Kubernetes Pod 里的 Agent 吗？
**A:** 支持，通过 DaemonSet 部署：

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: quic-agent
spec:
  selector:
    matchLabels:
      app: quic-agent
  template:
    metadata:
      labels:
        app: quic-agent
    spec:
      containers:
      - name: agent
        image: quic-robot/agent:latest
        env:
        - name: SERVER_URL
          value: "wss://quic-server.example.com"
        - name: AGENT_TOKEN
          valueFrom:
            secretKeyRef:
              name: agent-secret
              key: token
```

---

还有其他疑问吗？🌫️
