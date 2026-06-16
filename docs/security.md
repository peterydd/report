# 安全实践

> 适用版本：v1.0.x  
> 报告漏洞请见 [SECURITY.md](../SECURITY.md)；本篇聚焦**安全配置与最佳实践**。

## 1. 威胁建模

| 攻击面 | 风险 | 缓解措施 |
|--------|------|----------|
| 配置文件含明文密码 | 容器/主机被入侵后泄露 | 用 Secret 注入 / 加密存储 |
| SQL 注入（配置中含 SQL） | 取决于来源可信度 | 配置仅由可信管理员维护；不接受外部输入 |
| 邮件被窃听 | TLS 关闭即明文 | 强制 `insecureSkipVerify: false`，使用 587/465 |
| 容器逃逸 | 镜像漏洞 / 特权运行 | 非 root、readOnlyRootFilesystem、capabilities drop |
| 日志泄露敏感信息 | DSN / 密码被打出 | 自行审查 `log.Printf` 输出 |
| 配置文件被未授权修改 | RBAC 缺失 | K8s RBAC / 文件权限 600 |

## 2. 配置安全

### 2.1 密码注入

```yaml
smtp:
  password: ${SMTP_PASSWORD}     # viper 读取环境变量
```

K8s 中用 Secret 注入：

```yaml
env:
  - name: SMTP_PASSWORD
    valueFrom:
      secretKeyRef:
        name: report-secret
        key: smtp-password
```

### 2.2 数据库权限

建议为报表创建**只读专用账号**：

```sql
-- MySQL
CREATE USER 'report_ro'@'%' IDENTIFIED BY '...';
GRANT SELECT ON reporting.* TO 'report_ro'@'%';
FLUSH PRIVILEGES;
```

```sql
-- PostgreSQL
CREATE USER report_ro WITH PASSWORD '...';
GRANT CONNECT ON DATABASE reporting TO report_ro;
GRANT USAGE ON SCHEMA public TO report_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO report_ro;
```

### 2.3 SMTP 安全

```yaml
smtp:
  host: smtp.example.com
  port: "587"           # 587 (STARTTLS) 或 465 (SMTPS)
  insecureSkipVerify: false   # 生产必须 false
  timeout: 30s
```

- **不要** 在生产使用 `insecureSkipVerify: true`
- 使用 OAuth2 验证（如 Gmail）需要外部适配（v1.2 计划）

### 2.4 文件权限

```bash
chmod 600 /opt/report/config/config.yaml
chown report:report /opt/report/config/config.yaml
```

## 3. 网络安全

### 3.1 出口白名单

工具只主动出站到：

1. 数据库（TCP 3306/5432/1521/9000）
2. SMTP（TCP 25/465/587）

在云上用 SG / Firewall 限制目的 IP。

### 3.2 入口

工具**不监听任何端口**。如要远程触发，请用 CronJob / 任务调度平台。

## 4. 容器安全

### 4.1 镜像

- 多阶段构建，最终镜像仅含二进制 + `/etc/ssl/certs`
- 基础镜像：`gcr.io/distroless/static-debian12:nonroot`
- 不安装 shell、curl 等

### 4.2 运行配置

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  allowPrivilegeEscalation: false
  capabilities:
    drop: [ALL]
  readOnlyRootFilesystem: true
  seccompProfile:
    type: RuntimeDefault
```

### 4.3 镜像扫描

```bash
docker scout quickview peterydd/report:latest
trivy image peterydd/report:latest
```

CI 集成示例见 [operations.md#镜像扫描](operations.md)。

## 5. 代码安全

### 5.1 SQL 注入

工具不直接拼接用户输入；所有 SQL 来自 `config.yaml` 的 `sheets[].sql`，由可信管理员维护。**不要**让外部用户提供 SQL 字段。

如未来要支持 SQL 模板渲染，必须使用参数化：

```go
db.Query("SELECT * FROM users WHERE id = ?", userID)
```

### 5.2 依赖安全

- CI 中跑 `go mod verify`
- 定期 `go list -m -u all` 升级
- 用 [govulncheck](https://golang.org/x/vuln/) 扫描：

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### 5.3 敏感日志

代码审查时检查 `log.Printf` 是否打印了：

- DSN（含密码）
- SMTP 密码
- 邮件内容
- 个人信息（邮箱除外，因为是收件人）

## 6. 供应链

- 所有依赖走 `go.sum` 校验
- CI 中固定 Go 版本（`go.mod` 中 `go 1.26.4`）
- 镜像 tag 必须用具体版本，禁用 `latest`（生产）

## 7. 数据保护

| 数据 | 保护 |
|------|------|
| 报表数据 | 数据库侧：加密 at rest、TLS 传输 |
| 邮件内容 | SMTP 强制 TLS；正文非加密（业务接受） |
| xlsx 文件 | 文件系统层权限；如含敏感数据，发送前自行加密 |
| 密码 / Token | 永远走 Secret / 环境变量，不入 Git |

## 8. 审计

v1.2 计划支持结构化审计日志：

```json
{
  "ts": "2024-05-01T09:00:00Z",
  "report": "daily_sales",
  "sheets": 3,
  "rows": 12345,
  "duration_ms": 2340,
  "email_to": ["manager@example.com"]
}
```

## 9. 合规清单

部署前自检：

- [ ] `insecureSkipVerify: false`（生产）
- [ ] 数据库使用只读账号
- [ ] SMTP 密码通过 Secret 注入
- [ ] 容器以非 root 用户运行
- [ ] `readOnlyRootFilesystem: true`
- [ ] 镜像 tag 非 `latest`
- [ ] `govulncheck` 通过
- [ ] 网络策略限制出口
- [ ] 配置文件权限 ≤ 600
- [ ] 镜像扫描通过

## 10. 关联

- [SECURITY.md](../SECURITY.md) — 漏洞报告流程
- [operations.md](operations.md) — 部署安全细节
- [development.md](development.md) — 安全编码规范
