# 配置参考

> 适用版本：v1.0.x  
> 配置文件：`config.yaml`（也可放在 `./configs/config.yaml` 等，详见 [配置搜索路径](#配置搜索路径)）

## 1. 配置搜索路径

启动时按顺序查找 `config.yaml`：

```
/             (容器挂载点)
./
./configs
../configs
../../configs
```

找到第一个即用，未找到则报错退出。

## 2. 顶层结构

```yaml
database: { ... }     # 必填
smtp:     { ... }     # 必填
reports:  [ ... ]     # 必填，至少 1 个
```

## 3. `database` 块

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `driver` | string | ✅ | - | `mysql` / `postgresql` / `oracle` / `clickhouse` |
| `source` | string | ✅ | - | DSN 连接串（见各驱动示例） |
| `maxOpenConns` | int | ❌ | 25 | 连接池最大打开数 |
| `maxIdleConns` | int | ❌ | 5  | 连接池最大空闲数 |
| `connMaxLifetime` | string (duration) | ❌ | `3m` | 连接最长存活时间 |

### 3.1 各驱动 DSN 示例

#### MySQL

```
user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=true&loc=Local
```

常用参数：
- `charset=utf8mb4` — 推荐，支持 emoji
- `parseTime=true` — 自动解析 DATETIME 到 `time.Time`
- `loc=Local` — 使用本地时区

#### PostgreSQL

```
postgres://user:pass@127.0.0.1:5432/dbname?sslmode=disable
```

参数：
- `sslmode=disable` — 仅测试；生产建议 `require` 或 `verify-full`
- `connect_timeout=10`

#### Oracle

```
oracle://user:pass@127.0.0.1:1521/FREEPDB1
```

#### ClickHouse

```
clickhouse://user:pass@127.0.0.1:9000/dbname?dial_timeout=10s
```

参数：
- `dial_timeout=10s`
- `read_timeout=30s`

### 3.2 调优建议

| 场景 | maxOpenConns | maxIdleConns | connMaxLifetime |
|------|--------------|--------------|------------------|
| 单报表少 sheet | 10 | 2 | 3m |
| 单报表多 sheet 并发 | ≥ sheet 数（受信号量 8 约束） | 5–10 | 3m |
| 高频调度 | 50 | 10 | 5m |
| 跨地域长延迟 | 30 | 5 | 1m（主动重建） |

## 4. `smtp` 块

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `host` | string | ✅ | - | SMTP 服务器域名 |
| `port` | string | ✅ | - | 通常 `25` / `465` / `587` |
| `username` | string | ✅ | - | 认证账号 |
| `password` | string | ✅ | - | 认证密码（推荐用环境变量） |
| `insecureSkipVerify` | bool | ❌ | `false` | 跳过 TLS 证书校验（**生产环境必须 false**） |
| `timeout` | string (duration) | ❌ | `30s` | 连接与写超时 |

示例：

```yaml
smtp:
  host: smtp.gmail.com
  port: "587"
  username: ${SMTP_USER}        # viper 支持环境变量插值
  password: ${SMTP_PASSWORD}
  insecureSkipVerify: false
  timeout: 30s
```

## 5. `reports` 列表

每个报表对象：

```yaml
- name: string              # 必填
  workBook:                 # 必填
    prefix: string
    dateFormat: string      # Go 时间格式
    suffix: string          # 通常 .xlsx
  sheets:                   # 必填，至少 1 个
    - { ... }
  message:                  # 必填
    { ... }
```

### 5.1 `workBook`

| 字段 | 必填 | 说明 |
|------|------|------|
| `prefix` | ❌ | 文件名前缀（可用业务名） |
| `dateFormat` | ❌ | `time.Now().Format(...)` 的格式串，默认空 |
| `suffix` | ❌ | 文件后缀，默认空 |

> 注意：**最终文件名 = `prefix + time.Now().Format(dateFormat) + suffix`**，三者拼装，不带路径。  
> 多报表同时运行可能因 `time.Now()` 在同一秒而撞名 → 建议 `dateFormat: "20060102_150405"` 精确到秒。

### 5.2 `sheets[]`

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `name` | string | ✅ | - | Sheet 名（同时是工作表标签） |
| `sql` | string | ✅ | - | 待执行的 SELECT 语句 |
| `column` | string | ❌ | - | 逗号分隔的表头，如 `"日期,产品,金额"`；为空则用 DB 返回的列名 |
| `isSum` | bool | ❌ | `false` | 是否在末行追加合计 |
| `sumBeginColumn` | int | ❌ | 0 | 从哪一列开始求和（1-based） |
| `enableStream` | bool | ❌ | `false` | 是否流式查询（>10 万行建议 `true`） |
| `batchSize` | int | ❌ | `10000` | 流式批次大小 |

`isSum=true` 时，会在最后一行写入 "Total/总计"，并对 `sumBeginColumn` 起的所有列写 `=SUM(列:列)` 公式。

### 5.3 `message`

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `from` | string | ✅ | - | 发件人邮箱 |
| `to` | []string | ✅ | - | 主要收件人，至少 1 个 |
| `cc` | []string | ❌ | `[]` | 抄送 |
| `bcc` | []string | ❌ | `[]` | 密送（不出现在邮件头中） |
| `subject` | string | ✅ | - | 主题 |
| `body` | string | ❌ | `""` | 正文 |
| `contentType` | string | ❌ | `text/plain;charset=utf-8` | MIME 类型 |
| `attachment.contentType` | string | ❌ | - | 附件 MIME，推荐 `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| `attachment.withFile` | bool | ❌ | `false` | 是否带附件 |

> **历史 bug**：`buildRecipients` 早期版本误把 `bcc` 替换为 `cc`，已在 v1.0.1 修复（见 [CHANGELOG](../CHANGELOG.md)）。

## 6. 完整配置示例

```yaml
database:
  driver: "mysql"
  source: "report:secret@tcp(10.0.0.10:3306)/reporting?charset=utf8mb4&parseTime=true&loc=Local"
  maxOpenConns: 30
  maxIdleConns: 8
  connMaxLifetime: "5m"

smtp:
  host: "smtp.exmail.qq.com"
  port: "465"
  username: "report@example.com"
  password: "your-password"
  insecureSkipVerify: false
  timeout: "30s"

reports:
  - name: "daily_sales"
    workBook:
      prefix: "销售报表_"
      dateFormat: "20060102_150405"
      suffix: ".xlsx"
    sheets:
      - name: "订单明细"
        sql: |
          SELECT order_id, customer, amount, created_at
          FROM orders
          WHERE created_at >= CURDATE()
        column: "订单号,客户,金额,创建时间"
        isSum: true
        sumBeginColumn: 3
        enableStream: false
        batchSize: 10000
      - name: "客户汇总"
        sql: |
          SELECT customer, COUNT(*) AS cnt, SUM(amount) AS total
          FROM orders
          WHERE created_at >= CURDATE()
          GROUP BY customer
        column: "客户,订单数,总金额"
        isSum: true
        sumBeginColumn: 2
        enableStream: true
        batchSize: 20000
    message:
      from: "report@example.com"
      to: ["manager@example.com"]
      cc: ["team-lead@example.com"]
      bcc: ["audit@example.com"]
      subject: "每日销售报表 - {{date}}"
      body: |
        您好，
        附件是今日销售数据，请查收。
      contentType: "text/plain;charset=utf-8"
      attachment:
        contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        withFile: true
```

## 7. 配置热重载

工具使用 [viper](https://github.com/spf13/viper) 的 `WatchConfig`：

- 修改 `config.yaml` 后保存即生效
- 新值在**下一个 sheet / 报表**开始时读取
- 正在执行的报表不会被中断
- 校验失败时**不**覆盖当前配置，原配置继续生效

> v1.1 计划：通过 `SetOnChange` 回调主动通知告警通道。

## 8. 校验

`Config.Validate()` 会按以下顺序检查（首个错误即返回）：

1. `database` 块存在
2. `database.driver` 非空
3. `database.source` 非空
4. `smtp` 块存在
5. `smtp.host/port/username/password` 全部非空
6. `reports` 至少 1 个
7. 每个 `report.name` 非空
8. 每个 `report.sheets` 至少 1 个
9. 每个 `report.message.from/to/subject` 必填

## 9. 常见问题

### Q: `{{date}}` 占位符为什么不生效？
A: 当前未做模板渲染，主题里出现 `{{...}}` 会原样发送。v1.1 计划支持。

### Q: 数据库密码能写环境变量吗？
A: 支持 viper 插值：`password: ${SMTP_PASSWORD}`，需设置对应环境变量。

### Q: 多报表共享同一份 DB 配置？
A: 当前是 `database` 顶层唯一，所有报表共用。v1.1 计划支持按报表覆盖 DSN。
