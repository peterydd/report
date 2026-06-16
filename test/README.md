# test/

> 测试数据与 fixtures。  
> 单元测试位于各自包内 `*_test.go`；本目录**只放共享 fixture / 真实集成场景所需数据**。

## 目录

```
test/
├── README.md                       # 本文件
└── fixtures/
    └── config/
        └── test.yaml               # 集成测试用配置示例
```

## fixtures/config/test.yaml

最小可用的真实集成配置示例，供：

- 本地手动跑 `cp test/fixtures/config/test.yaml config.yaml` 后修改
- 集成测试 `REPORT_INTEGRATION=1 go test` 时参考

> 仓库里默认是占位 DSN，**不会**自动被任何测试加载；不要在 CI 中使用此文件连真实服务。

## 加载 fixtures 的示例

```go
import (
    "os"
    "testing"

    "github.com/peterydd/report/pkg/config"
)

func loadFixture(t *testing.T) *config.Config {
    t.Helper()
    data, err := os.ReadFile("../../test/fixtures/config/test.yaml")
    if err != nil {
        t.Fatalf("read fixture: %v", err)
    }
    // 自行 yaml.Unmarshal，或自己写解析器
    // 这里仅示意
    _ = data
    return nil
}
```

## Mock vs Fixture

| 类型 | 用途 | 速度 | 维护成本 |
|------|------|------|----------|
| Mock | 单元测试，模拟依赖 | 快 | 低 |
| Fixture | 真实数据 / 配置 | 中 | 中 |
| 集成测试 | 真实服务（MySQL/SMTP） | 慢 | 高 |

> **默认行为**：本项目绝大多数测试使用 Mock；真实服务集成测试需显式 `REPORT_INTEGRATION=1` 启用。

## 敏感数据

- **不要** 在 fixtures 中放真实密码
- 测试用的 DSN 应使用临时账号
- 提交前检查 `git diff test/`

## 未来扩展

- `test/fixtures/data/` — 报表数据快照（JSON / CSV）
- `test/fixtures/expected/` — 期望输出 xlsx
- `test/integration/` — 真实数据库迁移脚本

> 当前未启用，按需添加。
