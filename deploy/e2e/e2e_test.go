/*
Package e2e - end-to-end test harness.
包 e2e - 端到端测试工具。

This package exercises the full report pipeline against real services spun
up by deploy/docker/docker-compose.e2e.yml (MySQL 8.0 + MailHog). It is
opt-in: skipped unless REPORT_INTEGRATION=1 AND REPORT_E2E=1 are set.

Run from the repository root:
  docker compose -f deploy/docker/docker-compose.e2e.yml up -d
  REPORT_INTEGRATION=1 REPORT_E2E=1 \
      go test -count=1 -tags e2e ./test/e2e/...
  docker compose -f deploy/docker/docker-compose.e2e.yml down

Build tag `e2e` keeps this file out of the default `go test ./...` run.
*/
//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/peterydd/report/internal/app"
	"github.com/peterydd/report/pkg/config"
)

// configDir resolves the absolute path of test/e2e/fixtures/config so
// viper.AddConfigPath can pick up e2e.yaml regardless of the test binary's
// working directory (which is package-specific when go test runs).
// configDir 解析 test/e2e/fixtures/config 的绝对路径，使 viper 在任意
// 工作目录下都能找到 e2e.yaml。
func configDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Join(filepath.Dir(file), "fixtures", "config")
}

// mailhogMessages is the trimmed MailHog /api/v2/messages response shape we
// care about. MailHog returns much more; we only decode what we assert on.
// mailhogMessages 是我们关心的 MailHog /api/v2/messages 响应子集。
type mailhogMessages struct {
	Total int `json:"total"`
	Items []struct {
		ID      string `json:"ID"`
		From    struct {
			Mailbox string `json:"Mailbox"`
			Domain string `json:"Domain"`
		} `json:"From"`
		To []struct {
			Mailbox string `json:"Mailbox"`
			Domain string `json:"Domain"`
		} `json:"To"`
		Content struct {
			Headers map[string][]string `json:"Headers"`
			Body    string              `json:"Body"`
		} `json:"Content"`
	} `json:"Items"`
}

// TestE2E_FullPipeline spins up the App against MySQL + MailHog, runs the
// report, and asserts that:
//  1. Run() returns nil (no DB / SMTP errors)
//  2. An .xlsx file is produced in the test working directory
//  3. MailHog's HTTP API reports exactly one message with the expected
//     subject and recipient
//
// TestE2E_FullPipeline 启动 App 连接 MySQL + MailHog，运行报表并断言：
//   1. Run() 返回 nil
//   2. 当前工作目录生成 .xlsx 文件
//   3. MailHog HTTP API 报告恰好一封主题与收件人匹配的邮件
func TestE2E_FullPipeline(t *testing.T) {
	if os.Getenv("REPORT_INTEGRATION") == "" || os.Getenv("REPORT_E2E") == "" {
		t.Skip("set REPORT_INTEGRATION=1 REPORT_E2E=1 to run end-to-end tests")
	}

	// Make viper look at our fixture directory.
	// 指引 viper 加载 fixture 配置目录。
	dir := configDir(t)

	// Wait up to 30s for MySQL to become reachable so docker compose has
	// time to finish initialising the schema.
	// 等待 MySQL 最多 30s 以便 docker compose 完成 schema 初始化。
	if err := waitForPort("127.0.0.1:13306", 30*time.Second); err != nil {
		t.Skipf("MySQL not reachable, did you run `make e2e-up`? %v", err)
	}
	if err := waitForHTTP("http://127.0.0.1:8025/api/v2/messages", 30*time.Second); err != nil {
		t.Skipf("MailHog not reachable, did you run `make e2e-up`? %v", err)
	}

	// Clear any leftover MailHog messages from a previous run.
	// 清空 MailHog 上一轮残留的邮件。
	resetMailhog(t)

	// Load the e2e config fixture explicitly so the test is independent
	// of whatever config.yaml the developer happens to have in the cwd.
	// 显式加载 e2e 配置 fixture，使测试不依赖开发机当前目录的 config.yaml。
	conf, err := config.NewConfigFromPath(dir)
	if err != nil {
		t.Fatalf("load e2e config: %v", err)
	}

	// Run the report through the App's DI-aware constructor with
	// production collaborators (real MySQL + real SMTP).
	// 用 DI 感知构造函数跑报表，注入生产协作者（真实 MySQL + 真实 SMTP）。
	a := app.NewAppWithDeps(conf, nil)
	if err := a.Run(); err != nil {
		t.Fatalf("app.Run() failed: %v", err)
	}

	// 1. Excel file exists in cwd.
	// 1. 当前目录有 .xlsx 生成。
	xlsx, err := findXLSX(".")
	if err != nil {
		t.Fatalf("expected an .xlsx report: %v", err)
	}
	if xlsx.Size() == 0 {
		t.Errorf("xlsx %s is empty", xlsx.Name())
	}

	// 2. MailHog received exactly one message.
	// 2. MailHog 收到恰好一封邮件。
	msgs := fetchMailhog(t)
	if msgs.Total != 1 {
		t.Fatalf("expected 1 MailHog message, got %d", msgs.Total)
	}
	m := msgs.Items[0]
	if got := m.Content.Headers["Subject"]; len(got) == 0 || got[0] != "E2E Orders Report" {
		gotDump, _ := json.Marshal(m.Content.Headers)
		t.Errorf("unexpected subject: %s", string(gotDump))
	}
	if len(m.To) != 1 || m.To[0].Mailbox != "qa" || m.To[0].Domain != "report.local" {
		t.Errorf("unexpected recipient: %+v", m.To)
	}
	if !bytes.Contains([]byte(m.Content.Body), []byte("orders report attached")) {
		t.Errorf("body missing expected fragment; got: %s", m.Content.Body)
	}

	t.Cleanup(func() { _ = os.Remove(xlsx.Name()) })
}

// waitForPort polls a TCP address until it accepts connections or the
// timeout expires. Used to give docker compose time to start MySQL.
// waitForPort 轮询 TCP 端口直到可连接或超时。用于等 docker compose 启动 MySQL。
func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := dialTCP(addr)
		if err == nil {
			conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s: %w", addr, lastErr)
}

// waitForHTTP polls an HTTP URL until it returns 2xx or the timeout expires.
// waitForHTTP 轮询 HTTP URL 直到返回 2xx 或超时。
func waitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode/100 == 2 {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s: %w", url, lastErr)
}

// resetMailhog deletes all messages currently held by MailHog.
// resetMailhog 删除 MailHog 当前持有的所有邮件。
func resetMailhog(t *testing.T) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodDelete, "http://127.0.0.1:8025/api/v1/messages", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("resetMailhog: %v (ignored)", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}

// fetchMailhog returns the current MailHog inbox via /api/v2/messages.
// fetchMailhog 通过 /api/v2/messages 拉取 MailHog 当前收件箱。
func fetchMailhog(t *testing.T) mailhogMessages {
	t.Helper()
	resp, err := http.Get("http://127.0.0.1:8025/api/v2/messages")
	if err != nil {
		t.Fatalf("fetchMailhog: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out mailhogMessages
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode MailHog response: %v (body=%s)", err, body)
	}
	return out
}

// dialTCP opens a TCP connection to the supplied host:port and closes it
// immediately. Used as a cheap reachability probe.
// dialTCP 打开到 host:port 的 TCP 连接后立即关闭，用作轻量级可达性探测。
func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 2*time.Second)
}

// findXLSX returns the first .xlsx file in dir. The test fails if no
// report was produced.
// findXLSX 返回 dir 中第一个 .xlsx 文件；没有则测试失败。
func findXLSX(dir string) (os.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".xlsx" {
			return e.Info()
		}
	}
	return nil, fmt.Errorf("no .xlsx file in %s", dir)
}
