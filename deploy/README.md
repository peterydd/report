# 部署指南 / Deployment Guide

本目录提供两种生产部署方案：单机 `docker-compose` 与 Kubernetes `CronJob`。

| 方案 | 适用场景 | 调度方式 |
|---|---|---|
| `docker-compose` | 单机 / 边缘节点 | 宿主机 cron 或 systemd timer |
| `k8s CronJob` | 集群 / 多节点 | Kubernetes 原生 CronJob |

`deploy/docker/docker-compose.e2e.yml` 保留为 E2E 测试用，**不要**用于生产。

---

## 1. docker-compose 部署

### 1.1 准备配置

```bash
cd deploy/docker
cp .env.example .env
# 编辑 .env，填入真实的 SMTP / DB 凭据
```

`.env` 文件包含敏感信息，**不要**提交到仓库。已在 `.gitignore` 忽略。

### 1.2 构建并运行

```bash
# 一次性运行（生成报表 → 发邮件 → 退出 0）
docker compose run --rm report

# 或后台启动（用于人工排错；正常生产用 host cron 触发）
docker compose up
```

### 1.3 定时调度

compose 本身不调度。推荐方式（任选其一）：

**systemd 定时器**（推荐，无 root cron 风险）：
```ini
# /etc/systemd/system/report-daily.service
[Unit]
Description=Daily report generation

[Service]
WorkingDirectory=/opt/report/deploy/docker
ExecStart=/usr/bin/docker compose run --rm report
```

```ini
# /etc/systemd/system/report-daily.timer
[Timer]
OnCalendar=*-*-* 08:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

**传统 crontab**：
```cron
0 8 * * * cd /opt/report/deploy/docker && /usr/bin/docker compose run --rm report >> /var/log/report.log 2>&1
```

---

## 2. Kubernetes 部署

### 2.1 一次性应用

```bash
# 修改 30-secret.yaml 的 REPORT_SMTP_PASSWORD
# 修改 20-configmap.yaml 的 SQL / 收件人
kubectl apply -k deploy/k8s
```

### 2.2 自定义镜像

```bash
# 在项目根目录构建并推送到镜像仓库
docker build -f deploy/docker/Dockerfile.production -t registry.example.com/report:1.1.0 .
docker push registry.example.com/report:1.1.0
```

修改 `50-cronjob.yaml` 的 `image:` 字段指向新镜像；或用 kustomize patch：

```yaml
# kustomize patch 示例
images:
  - name: report
    newName: registry.example.com/report
    newTag: 1.1.0
```

### 2.3 触发时间

编辑 `50-cronjob.yaml`：
- `spec.schedule`：标准 cron 表达式（5 字段）
- `spec.timeZone`：IANA 时区名（k8s ≥ 1.27）

### 2.4 手动触发 / 调试

```bash
# 立即跑一次（不影响 cron 调度）
kubectl create job -n report --from=cronjob/report report-manual-1

# 查看日志
kubectl logs -n report -l app.kubernetes.io/component=runner --tail=200

# 进入 Pod 调试（不推荐，会修改只读根文件系统）
kubectl debug -n report -it --image=busybox:1.36 --target=report
```

### 2.5 资源调优

`50-cronjob.yaml` 默认：
- requests: `100m CPU / 128Mi RAM`
- limits: `1 CPU / 512Mi RAM`

大报表（百万行）建议把内存 limit 提到 `2Gi`；Excel 流式写入内存峰
值 ≈ 50MB / sheet。`activeDeadlineSeconds: 1800` 防止卡死。

### 2.6 升级与回滚

```bash
# 升级镜像版本
kubectl set image -n report cronjob/report report=registry.example.com/report:1.1.1

# 回滚（仅历史镜像可用）
kubectl rollout undo -n report cronjob/report   # CronJob 没有历史；改 tag 重 apply

# 暂停调度
kubectl patch -n report cronjob/report -p '{"spec":{"suspend":true}}'
```

---

## 3. 配置文件覆盖优先级

v1.1.0+ 起，所有 `config.yaml` 字段都能被 `REPORT_*` 环境变量覆盖
（点号转下划线）：

```
config.yaml < REPORT_* env < REPORT_DATABASE_* / REPORT_SMTP_*
```

完整列表见 `pkg/config/config.go:bindEnvOverrides`。推荐把
**SMTP 密码 / DB 密码** 走环境变量，**报表 SQL / 收件人** 走
ConfigMap。

---

## 4. 安全检查清单

部署到生产前请确认：

- [ ] `SMTP_INSECURESKIPVERIFY=false`
- [ ] `REPORT_SMTP_PASSWORD` 通过 Secret 注入，未明文落盘
- [ ] `ConfigMap` 中的 `database.source` 不含明文密码
- [ ] 镜像 tag 锁定到具体版本（不要用 `:latest`）
- [ ] `runAsNonRoot: true` + `runAsUser: 65532`
- [ ] `readOnlyRootFilesystem: true`
- [ ] `capabilities.drop: [ALL]`
- [ ] 启用 Pod Security Admission `restricted`（已默认配置）
- [ ] PVC 加密存储（云厂商 StorageClass 设置）
- [ ] `automountServiceAccountToken: false`（已默认配置）

---

## 5. 故障排查

| 现象 | 检查点 |
|---|---|
| Pod 一直 `Pending` | PVC 绑定？StorageClass 名称？ |
| Pod 一直 `ContainerCreating` | 镜像能 pull 下来？Secret 已创建？ |
| 邮件没收到 | SMTP 端口对？密码对？日志里 SMTP 错误？ |
| SQL 报错 | ConfigMap 内容？`kubectl exec` 进 Pod 跑相同 DSN 测试 |
| 时区不对 | CronJob `timeZone` 与 ConfigMap `dateFormat` 一致？ |

更多排错命令见 `docs/operations.md`。
