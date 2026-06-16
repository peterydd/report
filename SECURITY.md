# 安全政策 (Security Policy)

## 支持的版本

| 版本 | 支持状态 |
|------|----------|
| 1.0.x | ✅ 接受安全更新 |
| < 1.0 | ❌ 不再支持 |

## 报告安全漏洞

我们非常重视安全问题。请 **不要** 通过公开 GitHub Issues 报告。

📧 **安全邮箱**：peterydd@example.com  
📨 **GitHub**：通过 [Security Advisories](https://github.com/peterydd/report/security/advisories/new) 私密报告

请在主题中包含 `SECURITY` 字样，便于优先处理。

### 报告内容

1. **漏洞描述**
   - 漏洞类型（如：信息泄露、SQL 注入、认证绕过）
   - 影响的版本
   - 严重程度评估（CVSS 估算）
   - 是否已公开披露

2. **复现步骤**
   - 详细复现说明
   - 概念验证代码 / 命令
   - 配置文件（**必须脱敏**）

3. **影响范围**
   - 受影响的功能 / 端点
   - 数据泄露 / 提权 / DoS 等
   - 是否需要认证

4. **建议修复**（可选）
   - 你的修复思路
   - 参考链接

### 响应流程

| 阶段 | 时间 | 内容 |
|------|------|------|
| 确认收悉 | 24h 内 | 回执邮件 / Issue |
| 初步评估 | 72h 内 | 验证 + 严重程度定级 |
| 修复开发 | 1-7 天 | 写补丁 + 内部测试 |
| 协调披露 | 视情况 | 与报告者同步 |
| 发布修复 | 按 SLA | 见下表 |
| 公开披露 | 修复后 | Security Advisory + CHANGELOG |

### 修复 SLA

| 严重程度 | 修复发布 |
|----------|----------|
| Critical (RCE / 凭据泄露) | 7 天内 |
| High     | 14 天内 |
| Medium   | 30 天内 |
| Low      | 下一次常规发布 |

## 历史漏洞

| 编号 | 严重 | 版本 | 描述 | 状态 |
|------|------|------|------|------|
| （暂无公开条目）|  |  |  |  |

## 安全实践

> 完整的安全部署指南见 [docs/security.md](docs/security.md)

### 给用户

1. **保持更新**：及时升级到最新补丁版本
2. **配置安全**：
   - 生产环境 `insecureSkipVerify: false`
   - 数据库账号只读
   - SMTP 密码通过 Secret 注入
3. **审计日志**：定期检查 `log.Printf` 输出，确认无敏感信息
4. **网络隔离**：工具**只主动出站**到数据库 + SMTP；不监听任何端口

### 给开发者

1. **SQL 注入防护**：所有 SQL 来自可信配置，**不要**接受外部输入
2. **依赖安全**：CI 跑 `govulncheck`
3. **日志脱敏**：禁止打印 DSN / 密码 / Token
4. **输入校验**：`Config.Validate()` 是第一道关

## 容器安全

- 多阶段构建，distroless 基础镜像
- 以非 root 用户运行
- `readOnlyRootFilesystem: true`
- `capabilities.drop: [ALL]`
- `seccompProfile: RuntimeDefault`

完整清单：[docs/security.md § 容器安全](docs/security.md#4-容器安全)

## 致谢

感谢以下安全研究人员的负责任披露（暂无）。

## 合规清单

部署前自检见 [docs/security.md § 合规清单](docs/security.md#9-合规清单)。

---

最后更新：2026-06-06
