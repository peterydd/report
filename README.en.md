# Report

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.4-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker" alt="Docker Ready">
  <img src="https://img.shields.io/badge/Kubernetes-Supported-326CE5?style=for-the-badge&logo=kubernetes" alt="Kubernetes">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
</p>

<p align="center">A high-performance database report generator and email delivery system with million-row streaming support.</p>

<p align="center">
  <a href="README.md">简体中文</a> ·
  <a href="docs/INDEX.md">Docs</a> ·
  <a href="ROADMAP.md">Roadmap</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Command Line](#command-line)
- [Environment Variables](#environment-variables)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Performance & Tuning](#performance--tuning)
- [Deployment](#deployment)
- [Testing](#testing)
- [FAQ](#faq)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Multi-Database Support** — unified interface for MySQL, PostgreSQL, Oracle, and ClickHouse
- **Streaming Queries** — handle millions of rows with low memory via `database/sql` `rows.Next()` callbacks
- **Concurrent Sheets** — multi-sheet parallel queries within a single report, capped by a semaphore
- **Email Delivery** — SMTP with STARTTLS, base64 attachments, CC/BCC, configurable timeout
- **Flexible Configuration** — YAML with strict validation and viper hot-reload
- **Mock Testing** — `db.MockDB` and `mail.MockMail` for dependency-free unit tests
- **Container-Ready** — multi-stage Dockerfile, Kubernetes CronJob-friendly
- **Observable** — key checkpoints logged via `log.Printf` for log aggregation

## Quick Start

### Prerequisites

- Go 1.26.4 (see `go.mod`)
- A reachable database (MySQL / PostgreSQL / Oracle / ClickHouse)
- A reachable SMTP service

### 1. Clone & Build

```bash
git clone https://github.com/peterydd/report.git
cd report
go mod tidy
make build          # Linux/macOS
# Windows: go build -o report.exe ./cmd/report
```

### 2. Prepare Config

```bash
cp configs/config.yaml.example config.yaml
# Edit config.yaml with real database and SMTP credentials
```

### 3. Run

```bash
./report            # Linux/macOS
# Windows: .\report.exe
```

## Command Line

| Flag | Description |
|------|-------------|
| `-version` | Print version and build time (injected via `main.version` / `main.buildTime`) |
| `-help`    | Print usage information |

Without any flag, the tool runs the normal flow: load config → connect → query → email.

> Inject versions at build time:
> `go build -ldflags "-X main.version=v1.1.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/report`

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `REPORT_INTEGRATION` | Set to non-empty to run integration tests against real services | unset → skip |
| `CONFIG_PATH`        | Override the config search directory list (viper) | `./`, `./configs/`, `/` |

Integration test example:

```bash
REPORT_INTEGRATION=1 go test ./...
```

## Configuration

`config.yaml` is searched in this order:

```
/             (container mount point)
./
./configs
../configs
../../configs
```

Minimal example (full reference in `configs/config.yaml.example`):

```yaml
database:
  driver: "mysql"                              # mysql | postgresql | oracle | clickhouse
  source: "user:pass@tcp(localhost:3306)/db"
  maxOpenConns: 25
  maxIdleConns: 5
  connMaxLifetime: "3m"

smtp:
  host: "smtp.example.com"
  port: "587"
  username: "sender@example.com"
  password: "your-password"
  insecureSkipVerify: false
  timeout: "30s"

reports:
  - name: "daily_sales"
    workBook:
      prefix: "sales_"
      dateFormat: "20060102"
      suffix: ".xlsx"
    sheets:
      - name: "sales"
        sql: "SELECT date, product, amount FROM sales"
        column: "Date,Product,Amount"
        isSum: true
        sumBeginColumn: 3
        enableStream: true
        batchSize: 50000
    message:
      from: "sender@example.com"
      to: ["manager@example.com"]
      cc: ["team@example.com"]
      subject: "Daily Sales Report"
      body: "Please find the report attached"
      contentType: "text/plain;charset=utf-8"
      attachment:
        contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        withFile: true
```

Full reference: [docs/configuration.md](docs/configuration.md).

## Architecture

```
                +-----------------+
                │   cmd/report    │   Entry: parse flag → app.Run()
                +--------+--------+
                         │
                +--------v--------+
                │  internal/app   │   Orchestration: pool / concurrent sheets / email
                +--------+--------+
                         │
   +---------+-----------+-----------+---------+
   │         │           │           │         │
+--v---+ +---v---+   +---v---+   +---v---+ +---v---+
│ db   │ | excel |   │ mail  │   │ cfg   │ (future)
| pkg  │ |  pkg  |   |  pkg  |   |  pkg  |
+------+ +-------+   +-------+   +-------+
   4 drivers   excelize   smtp         viper
   Mock        freeze+SUM  attach+TLS  hot-reload
```

See [docs/architecture.md](docs/architecture.md) for layers and concurrency model.

## Performance & Tuning

### Dataset Size

| Rows | Mode | Memory |
|------|------|--------|
| < 100K   | Normal | ~100MB |
| 100K-1M  | Stream (`enableStream: true`) | ~100MB |
| > 1M     | Stream with larger `batchSize` | ~200MB |

### Connection Pool

- Multi-sheet reports: `maxOpenConns >= sheet count` (capped by internal semaphore `maxConcurrentSheets=8`)
- High-frequency: increase `maxIdleConns` to avoid handshake
- Long-distance DB: shorten `connMaxLifetime`

### Email

- Per-send timeout enforced via `timeout`
- Concurrent sends require splitting `Run()` into a worker pool (planned for v1.1)
- Attachments >20MB → prefer object storage + link

## Deployment

### Docker

```bash
make docker-build
docker run -d \
  -v $(pwd)/config.yaml:/config.yaml \
  --name report peterydd/report:latest
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-report
spec:
  schedule: "0 9 * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: report
              image: peterydd/report:latest
              volumeMounts:
                - name: cfg
                  mountPath: /config.yaml
                  subPath: config.yaml
              resources:
                requests: { memory: "256Mi", cpu: "250m" }
                limits:   { memory: "2Gi",   cpu: "1000m" }
          restartPolicy: OnFailure
      volumes:
        - name: cfg
          configMap: { name: report-config }
```

See [docs/operations.md](docs/operations.md) for full ops guidance.

## Testing

```bash
# Unit tests (no DB/SMTP required)
make test

# Integration tests (real services)
REPORT_INTEGRATION=1 go test ./...

# Coverage
make test-coverage

# Benchmarks
make benchmark
```

Mock usage:

```go
db := db.NewMockDB()
db.SetQueryResult("SELECT 1", [][]interface{}{ {"x"} })
results, _ := db.Query("SELECT 1")

mail := mail.NewMockMail()
mail.SetError("smtp down")
err := mail.Send(msg)  // returns the mocked error
```

## FAQ

### Q: Chinese characters render as garbled text?
A: Use `contentType: "text/plain;charset=utf-8"` (or `text/html;charset=utf-8`). Attachment file names are MIME B-encoded.

### Q: OOM on large datasets?
A: Enable `enableStream: true` and tune `batchSize` (10K–50K is typical).

### Q: Config edits not taking effect?
A: The tool watches `config.yaml` via viper; new values are picked up before the next sheet/report runs (in-flight reports are not interrupted).

### Q: How do I add a new database?
A: See [docs/development.md#adding-a-new-database-driver](docs/development.md).

### Q: Slow DB queries but no failure?
A: Tune the SQL itself. Concurrency is bounded by `maxConcurrentSheets=8` per process.

## Contributing

Issues and PRs are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md). For security issues, follow [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)

## Contact

- GitHub: <https://github.com/peterydd/report>
- Issues: <https://github.com/peterydd/report/issues>
- Discussions: <https://github.com/peterydd/report/discussions>

---

<p align="center">If this project helps you, please give it a ⭐️ Star!</p>
