# 运维指南

> 适用版本：v1.1.0  
> 面向：SRE、运维、值班工程师  
> 部署相关的快速索引见 [deploy/README.md](../deploy/README.md)。

## 目录

1. 部署形态
2. 本地二进制部署
3. Docker 部署
4. Kubernetes 部署
5. 配置管理
6. 监控与日志
7. 备份与恢复
8. 故障排查
9. 升级与回滚
10. 安全合规

---

## 1. 部署形态

| 形态 | 适用场景 | 调度方式 | 文档 |
|---|---|---|---|
| 本地二进制 | 调试 / 一次性任务 | 手动 / cron | [§2](#2-本地二进制部署) |
| Docker | 单机 / 边缘节点 | 宿主机 cron / systemd timer | [§3](#3-docker-部署) |
| Kubernetes | 集群 / 多节点 / 高可用 | Kubernetes CronJob | [§4](#4-kubernetes-部署) |

不论哪种形态，业务流程一致：

```
加载 config.yaml (可被 REPORT_* 环境变量覆盖)
    ↓
连接数据库 → 池化（MaxOpenConns / MaxIdleConns / ConnMaxLifetime）
    ↓
并发查询各 sheet（maxConcurrentSheets=8 信号量）
    ↓
生成 .xlsx（excleize 流式写）
    ↓
SMTP 发送（含 attachment、CC、BCC）
    ↓
退出码 0
```

---

## 2. 本地二进制部署

### 2.1 构建

```bash
make build VERSION=v1.1.0
# 产物：./report (Linux/macOS) 或 report.exe (Windows)
```

### 2.2 目录结构（建议）

```
/opt/report/
├── bin/report                 # 可执行
├── config/config.yaml         # 配置（限 600 权限）
├── output/                    # xlsx 输出（可选）
└── logs/                      # 日志（可选）
```

### 2.3 systemd 单元

```ini
# /etc/systemd/system/report@daily.service
[Unit]
Description=Report Generator
After=network.target

[Service]
Type=oneshot
User=report
WorkingDirectory=/opt/report
ExecStart=/opt/report/bin/report
EnvironmentFile=/opt/report/config/report.env
StandardOutput=append:/var/log/report/report.log
StandardError=append:/var/log/report/report.err

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/report@daily.timer
[Unit]
Description=Run report daily

[Timer]
OnCalendar=*-*-* 08:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
systemctl daemon-reload
systemctl enable --now report@daily.timer
```

---

## 3. Docker 部署

### 3.1 准备

```bash
cd deploy/docker
cp .env.example .env
# 编辑 .env：SMTP_HOST / SMTP_PASSWORD / DB_SOURCE 等
```

`.env` 含敏感信息，**不要**提交仓库（已在 `.gitignore` 忽略）。

### 3.2 运行

```bash
# 一次性运行
docker compose run --rm report

# 后台启动（调试用；生产推荐 host cron 触发）
docker compose up

# 查看产物（xlsx 默认落到 report-output 卷）
docker compose run --rm report
ls -la /var/lib/docker/volumes/report-output/_data/
```

### 3.3 定时调度

compose 本身不调度。建议用 **systemd timer** 触发（见 §2.3，命令改为
`/usr/bin/docker compose run --rm report`），或传统 crontab：

```cron
0 8 * * * cd /opt/report/deploy/docker && /usr/bin/docker compose run --rm report >> /var/log/report.log 2>&1
```

### 3.4 镜像版本升级

```bash
docker compose pull
docker compose run --rm report
```

---

## 4. Kubernetes 部署

### 4.1 一次性应用

```bash
# 编辑 30-secret.yaml：REPORT_SMTP_PASSWORD
# 编辑 20-configmap.yaml：SQL / 收件人
kubectl apply -k deploy/k8s
```

### 4.2 自定义镜像

```bash
docker build -f deploy/docker/Dockerfile.production \
  -t registry.example.com/report:1.1.0 .
docker push registry.example.com/report:1.1.0
```

修改 `50-cronjob.yaml` 的 `image:` 字段，或用 kustomize patch：

```yaml
images:
  - name: report
    newName: registry.example.com/report
    newTag: 1.1.0
```

### 4.3 触发时间

编辑 `50-cronjob.yaml`：
- `spec.schedule`：标准 cron 表达式（5 字段）
- `spec.timeZone`：IANA 时区名（k8s ≥ 1.27 支持）

### 4.4 手动触发 / 调试

```bash
# 立即跑一次
kubectl create job -n report --from=cronjob/report report-manual-1

# 查看日志
kubectl logs -n report -l app.kubernetes.io/component=runner --tail=200

# 进入 Pod 调试（不推荐，会修改只读根文件系统）
kubectl debug -n report -it --image=busybox:1.36 --target=report
```

### 4.5 资源调优

`50-cronjob.yaml` 默认：
- requests：`100m CPU / 128Mi RAM`
- limits：`1 CPU / 512Mi RAM`

大报表（百万行）建议把内存 limit 提到 `2Gi`；Excel 流式写入
内存峰值 ≈ 50MB / sheet。`activeDeadlineSeconds: 1800` 防止卡死。

### 4.6 升级与回滚

```bash
# 升级镜像版本
kubectl set image -n report cronjob/report report=registry.example.com/report:1.1.1

# 暂停调度
kubectl patch -n report cronjob/report -p '{"spec":{"suspend":true}}'
```

---

## 5. 配置管理

### 5.1 覆盖优先级

v1.1.0+ 起优先级：

```
默认值  <  config.yaml  <  REPORT_* 环境变量
```

完整 env 变量列表见 `pkg/config/config.go:bindEnvOverrides`。
生产推荐：

| 字段 | 走 yaml | 走 env | 走 Secret |
|---|---|---|---|
| `smtp.host` | ✅ | 可 | — |
| `smtp.port` | ✅ | 可 | — |
| `smtp.username` | ✅ | 可 | — |
| **`smtp.password`** | ❌ | — | ✅ 必走 |
| **`database.source`**（含密码） | ❌ | — | ✅ 必走 |
| SQL / 收件人 | ✅ | ❌ | ❌ |

### 5.2 配置校验

启动时 `Config.Validate()` 严格检查：
- `database.{driver,source}` 必填
- `smtp.{host,port,username,password}` 必填
- 至少一个 report，且包含 sheets + message

校验失败进程退出码非 0，stderr 输出原因。

### 5.3 配置热重载

`pkg/config` 监听 `config.yaml` 文件变更，校验通过后自动重新加载。
热重载只覆盖内存中的 `*Config`，不会重启进程。

---

## 6. 监控与日志

### 6.1 日志

所有日志走 Go 标准 `log` 包，**输出到 stderr**：

```
2026/06/15 10:24:15 sheet Orders using streaming mode, batch size: 100
2026/06/15 10:24:15 sheet Orders streaming query completed, 1234 rows processed
2026/06/15 10:24:16 report rpt_20260615.xlsx generated successfully
2026/06/15 10:24:16 email Daily Report sent successfully
```

关键事件：
- `configuration loaded successfully` — 启动 OK
- `sheet %s query completed` / `streaming query completed` — 单 sheet 完成
- `report %s generated successfully` — Excel OK
- `email %s sent successfully` — 邮件 OK
- `panic: ...` — sheet goroutine panic（v1.1.0 会被 recover，不再致命）
- `report %s skipped: ...` — 整报表跳过

### 6.2 退出码

| 码 | 含义 |
|---|---|
| 0 | 全部成功 |
| 1 | 配置加载 / 校验失败 |
| 1 | DB 连接失败 |
| 1 | 致命 panic 未恢复 |

> 注：单 sheet 失败 / 单邮件失败 / 部分报表失败 → Run() 继续后续，**进程仍退出 0**。
> 这是设计：一次失败不应让整个计划崩溃。监控告警应基于「期望邮件数 vs 实际邮件数」而非退出码。

### 6.3 建议接入的指标

- `report_run_total{result="success|fail"}`（自定义埋点）
- `report_email_sent_total{to=*,report=*}`
- `report_duration_seconds{report=*}`
- Kubernetes：`CronJob.status.active` / `.lastSuccessfulTime` / `.lastFailedTime`

v1.2 计划引入 Prometheus exporter（[ROADMAP § v1.2](../ROADMAP.md)）。

---

## 7. 备份与恢复

### 7.1 配置

```bash
# config.yaml 必须纳入配置管理（etcd / Vault / Git）
cp /opt/report/config/config.yaml backup/config-$(date +%Y%m%d).yaml
chmod 600 backup/config-*.yaml
```

### 7.2 输出（生成的 xlsx）

- 单机：默认落到 `/var/reports`（k8s 通过 PVC 持久化）
- 建议推送到 OSS / S3 / 备份存储，保留 N 天

```bash
# 简易同步（单机）
rsync -av /opt/report/output/ backup@backup-host:/srv/report/$(date +%Y%m)/
```

### 7.3 数据库

report CLI 是**只读**消费者，但请确保：
- DB 账号只授予 `SELECT` 权限（最小权限）
- 启用 DB 自带的备份策略（MySQL binlog / PostgreSQL WAL）

---

## 8. 故障排查

| 现象 | 排查步骤 |
|---|---|
| 进程立即退出，无日志 | 检查 `config.yaml` 是否存在且格式正确；设 `REPORT_*` env 覆盖验证 |
| `failed to load configuration` | 路径？权限？`Config.Validate` 错误信息 |
| `dial tcp ...: connect: connection refused` | DB host/port 可达？防火墙？VPN？ |
| `SMTP authentication failed` | 密码对？应用专用密码（不是登录密码）？ |
| 邮件进了垃圾箱 | SPF / DKIM / DMARC 配置；From 地址与 smtp 用户名一致？ |
| 部分 sheet 失败 | 日志看 `sheet %s query failed`；先用 `mysql` 客户端跑相同 SQL |
| 整报表跳过 | 全部 sheet 失败才触发；查 sheet 级日志 |
| CronJob 一直 Pending | PVC 绑定？StorageClass 名称？ |
| CronJob ContainerCreating | 镜像能 pull？Secret / ConfigMap 存在？ |
| 时区不对 | CronJob `timeZone` + ConfigMap `dateFormat` 一致？ |
| Goroutine panic | v1.1.0 已自动 recover，进程继续；查 `panicked:` 日志 |

### 调试工具

```bash
# 启用 trace 日志（v1.2+）
REPORT_LOG_LEVEL=debug ./report

# 模拟 SMTP（不真发）
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog

# 验证 config 加载
./report --config-check
```

---

## 9. 升级与回滚

### 9.1 升级路径

| 从 | 到 | 文档 |
|---|---|---|
| v1.0.x | v1.1.0 | [docs/migration.md § 从-v106-升级到-v110](migration.md) |
| v1.0.0 | v1.0.1 | [docs/migration.md § 从-v100-升级到-v101](migration.md) |

升级前：
1. 读 [CHANGELOG.md](../CHANGELOG.md)
2. 在测试环境跑 `make test`
3. 灰度一台机器或一个 CronJob
4. 24 小时观察后全量推送

### 9.2 回滚

```bash
# K8s：改镜像 tag 到上一个版本
kubectl set image -n report cronjob/report report=registry.example.com/report:1.0.6

# Docker compose
docker compose down
docker compose run --rm report   # 用旧镜像需先 docker tag

# 本地
git checkout v1.0.6
make build VERSION=v1.0.6
```

镜像 tag 永久保留（语义化版本承诺），旧版本随时可拉。

---

## 10. 安全合规

- **配置含敏感字段**（SMTP 密码 / DB DSN）→ 走 env / Secret，**绝不**入 git
- **镜像 tag 锁定**具体版本（不要用 `:latest`）
- **Pod Security**：`runAsNonRoot: true` + `readOnlyRootFilesystem: true`
- **网络**：SMTP 必须走 TLS 1.2+（v1.1.0 强制）
- **审计日志**：v1.2 计划引入 JSON 行审计

完整安全清单见 [docs/security.md](security.md) + [deploy/README.md § 4](../deploy/README.md)。

---

## 关联

- [deploy/README.md](../deploy/README.md) — 部署快速索引
- [docs/configuration.md](configuration.md) — 完整配置参考
- [docs/security.md](security.md) — 安全实践
- [docs/migration.md](migration.md) — 升级指南
- [CHANGELOG.md](../CHANGELOG.md) — 版本变更
