# pkg/mail

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

SMTP email delivery with STARTTLS, base64 attachments, CC / BCC, and timeout control.

> Project home: [README](../../README.md) · Architecture: [docs/architecture.md](../../docs/architecture.md)

## Features

- ✅ SMTP + PLAIN auth
- ✅ STARTTLS (configurable `insecureSkipVerify`; **must be false in production**)
- ✅ Multiple recipients: To / CC / BCC
- ✅ Attachments: base64-encoded, RFC 2047 B-encoded file names
- ✅ Configurable timeout
- ✅ Mock: `MockMail` with no network dependency

## Install

```bash
go get github.com/peterydd/report/pkg/mail
```

## Quick Start

```go
import (
    "log"
    "time"
    "github.com/peterydd/report/pkg/mail"
)

attach := mail.SetAttach(
    "report.xlsx",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    true,
)

msg := mail.SetMessage(
    "report@example.com",
    []string{"manager@example.com"},
    []string{"team@example.com"},
    []string{"audit@example.com"},
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
    false,
    30*time.Second,
)

if err := sender.Send(msg); err != nil {
    log.Fatal(err)
}
```

## Exported API

| Symbol | Description |
|--------|-------------|
| `Mail` (interface) | `Auth()` + `Send(m *Message) error` |
| `SetMessage(from, to, cc, bcc []string, subject, body, contentType string, attachment *Attach) *Message` | Build a message |
| `SetAttach(name, contentType string, withFile bool) *Attach` | Build an attachment |
| `NewSendMail(host, port, username, password string, insecureSkipVerify bool, timeout time.Duration) *SendMail` | Build a sender (`timeout=0` → 30s) |
| `(*SendMail).Auth()` | Explicit auth (called by `Send`) |
| `(*SendMail).Send(m *Message) error` | Send |

## Message Fields

```go
type Message struct {
    from, subject, body, contentType string
    to, cc, bcc                      []string
    attachment                       *Attach
}
```

> Unexported; use `SetMessage`.

## MIME Structure

```
From: <from>
To: <to1>,<to2>
Cc: <cc1>,...
Bcc: <bcc1>,...
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

> The boundary `GoBoundary` is hard-coded; collisions with body content are extremely unlikely.

## BCC Privacy

- Included in SMTP `Rcpt` list (required for delivery)
- Included in `Bcc:` header (some clients may display it to recipients)
- v1.1 will remove the `Bcc:` header for stronger privacy

> **v1.0.0 → v1.0.1 fix**: earlier `buildRecipients(m.to, m.cc, m.cc)` accidentally replaced BCC with CC, so BCC addresses were never delivered. Fixed in v1.0.1 to pass `m.bcc`.

## TLS Configuration

- `port=465` → implicit TLS (current code uses `StartTLS`; may not be compatible with strict 465 servers)
- `port=587` → STARTTLS (recommended)
- `port=25` → plaintext or opportunistic STARTTLS

Production:

```yaml
smtp:
  port: "587"
  insecureSkipVerify: false
```

## Mock Testing

```go
mock := mail.NewMockMail()

err := mock.Send(msg)              // success
mock.SetError("smtp auth failed")
err = mock.Send(msg)               // returns mocked error

count := mock.GetSendCount()
sent := mock.GetLastMessage()      // *MockMessage{From,To,Subject,Body,Attachment}
```

## Testing

```bash
# Unit + mock
go test ./pkg/mail

# Integration (real SMTP)
REPORT_INTEGRATION=1 go test -run TestSend ./pkg/mail
```

## Related

- [`pkg/config`](../config) — SMTP block
- [`internal/app`](../../internal/app) — called at the end of `Run()`
