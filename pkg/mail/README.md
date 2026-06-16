# pkg/mail

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

SMTP 邮件发送包，支持 STARTTLS、base64 附件、CC / BCC、超时控制。

> 项目主页：[README](../../README.md) · 架构：[docs/architecture.md](../../docs/architecture.md)

## 特性

- ✅ SMTP + PLAIN 认证
- ✅ STARTTLS（可配置 `insecureSkipVerify`，**生产必须 false**）
- ✅ 多收件人：To / CC / BCC
- ✅ 附件：base64 编码、文件名 RFC 2047 B 编码
- ✅ 超时控制（`timeout` 字段）
- ✅ Mock：`MockMail` 零网络依赖

## 安装

```bash
go get github.com/peterydd/report/pkg/mail
```

## 快速上手

```go
import (
    "log"
    "time"
    "github.com/peterydd/report/pkg/mail"
)

attach := mail.SetAttach(
    "report.xlsx",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    true, // 包含文件内容
)

msg := mail.SetMessage(
    "report@example.com",
    []string{"manager@example.com"},
    []string{"team@example.com"},
    []string{"audit@example.com"}, // BCC
    "Daily Sales",
    "Please find attached.",
    "text/plain;charset=utf-8",
    attach,
)

sender := mail.NewSendMail(
    "smtp.example.com",
    "587",
    "report@example.com",
    "your-password",
    false,           // 不跳过 TLS 校验
    30*time.Second,
)

if err := sender.Send(msg); err != nil {
    log.Fatal(err)
}
```

## 导出 API

| 符号 | 说明 |
|------|------|
| `Mail` (interface) | `Auth()` + `Send(m *Message) error` |
| `SetMessage(from string, to, cc, bcc []string, subject, body, contentType string, attachment *Attach) *Message` | 构造邮件 |
| `SetAttach(name, contentType string, withFile bool) *Attach` | 构造附件 |
| `NewSendMail(host, port, username, password string, insecureSkipVerify bool, timeout time.Duration) *SendMail` | 构造发送器（`timeout=0` 默认 30s） |
| `(*SendMail).Auth()` | 显式认证（`Send` 内部会调用） |
| `(*SendMail).Send(m *Message) error` | 发送 |

## 消息字段

```go
type Message struct {
    from        string
    to          []string
    cc          []string
    bcc         []string
    subject     string
    body        string
    contentType string
    attachment  *Attach
}
```

> 字段包私有，必须用 `SetMessage` 构造。

## MIME 头部

发送时构造如下多部分消息：

```
From: <from>
To: <to1>,<to2>
Cc: <cc1>,...
Bcc: <bcc1>,...        # 出现在头部但不展示给收件人客户端
Subject: <subject>
MIME-Version: 1.0
Date: <RFC1123>
Content-Type: multipart/mixed; boundary=GoBoundary

--GoBoundary
Content-Type: <contentType>

<body>

--GoBoundary
Content-Transfer-Encoding: base64
Content-Disposition: attachment
Content-Type: <attach.contentType>; name="=?UTF-8?B?<base64name>?="

<base64 payload>
--GoBoundary--
```

> 注意：MIME 边界硬编码为 `GoBoundary`，如邮件内容恰好出现该字符串会被截断（极小概率）。

## BCC 隐私

- 出现在 SMTP `Rcpt` 列表中（必须，否则收件人收不到）
- 出现在邮件头 `Bcc:` 中（**当前实现包含**；RFC 允许但部分客户端会展示给收件人）
- v1.1 计划：从 `Bcc:` 头部中移除以加强隐私

> **v1.0.0 → v1.0.1 修复**：早期 `buildRecipients(m.to, m.cc, m.cc)` 把 BCC 替换为 CC，BCC 实际未投递；已在 v1.0.1 修复为 `m.bcc`。

## TLS 配置

- `port=465` → 隐式 TLS（当前未自动启用，依赖 `StartTLS`，可能与某些 465 服务不兼容）
- `port=587` → STARTTLS（推荐）
- `port=25` → 明文或 STARTTLS（取决于服务器）

生产建议：

```yaml
smtp:
  port: "587"
  insecureSkipVerify: false
```

## Mock 测试

```go
mock := mail.NewMockMail()

// 模拟成功
err := mock.Send(msg)

// 模拟错误
mock.SetError("smtp: authentication failed")
err = mock.Send(msg)  // returns the mocked error

// 检查调用次数
count := mock.GetSendCount()

// 取出已发邮件
sent := mock.GetLastMessage()  // *MockMessage
```

`MockMessage` 包含 `From/To/Subject/Body/Attachment`，便于断言。

## 测试

```bash
# 单元 + Mock
go test ./pkg/mail

# 集成（真实 SMTP）
REPORT_INTEGRATION=1 go test -run TestSend ./pkg/mail
```

## 关联

- [`pkg/config`](../config) — SMTP 配置块
- [`internal/app`](../../internal/app) — 编排层在 `Run()` 末尾调用
