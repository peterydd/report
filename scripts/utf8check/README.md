# scripts/utf8check

校验仓库内文本文件是否为合法 UTF-8 编码。专门用于拦截 PowerShell 5.1 在
cp936 控制台下误把 UTF-8 文件按 GBK 解码再以 `?` 写回导致的损坏。

## 用法

```bash
# 检查整个仓库
go run ./scripts/utf8check

# 检查指定路径
go run ./scripts/utf8check docs/ README.md
```

也可通过 Makefile 目标运行：

```bash
make check-encoding
```

## 退出码

| 码 | 含义 |
|---|---|
| 0 | 所有检查通过 |
| 1 | 至少一个文件含非 UTF-8 字节，错误信息打印到 stderr |
| 2 | 遍历目录失败（如权限） |

## 检查范围

默认递归所有文本文件：

- `.md` `.markdown` `.yml` `.yaml` `.json` `.toml` `.go` `.sql`
- `.sh` `.bash` `.zsh` `.py` `.txt` `.example` `.sum` `.mod` `.env`
- 无后缀且 < 64 KB 的文件（如 `VERSION`）

跳过：

- `.git` `build/` `dist/` `node_modules/` `vendor/` `output/` `tmp/` `temp/`
- 二进制后缀：`.png` `.jpg` `.pdf` `.xlsx` `.exe` `.so` ...

## CI 集成

`.github/workflows/ci.yml` 的 `lint` 任务中调用本工具：

```yaml
- name: Validate UTF-8 encoding
  run: go run ./scripts/utf8check
```

## 事故复盘

2026-06-15 v1.1.0 发布时 `docs/operations.md` 因 PowerShell 5.1 + cp936
+ `Set-Content -Encoding utf8` 误操作被写入乱码 blob `f55c7586`。
v1.1.1 修复后加入本工具作 CI 卡口。

## 扩展

需要检查更多文件类型时，编辑 `textExts` map。需要排除更多目录时，
编辑 `excludeDirs` map。
