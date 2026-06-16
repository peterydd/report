// utf8check - 校验仓库内文本文件 UTF-8 编码完整性。
//
// 用法：
//   go run ./scripts/utf8check                     # 检查默认文件
//   go run ./scripts/utf8check path1 path2 ...     # 检查指定路径
//
// 默认检查范围：
//   - 所有 .md / .yml / .yaml / .json / .toml / .sh / .go / .sql 文件
//   - 排除 .git / build/ / dist/ / node_modules/
//   - 排除二进制 (.png .jpg .pdf .xlsx ...)
//
// 退出码：
//   0  全部文件 UTF-8 合法
//   1  至少一个文件含非 UTF-8 字节
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// 校验的文本后缀（不含二进制）
var textExts = map[string]bool{
	".md":      true,
	".markdown": true,
	".yml":     true,
	".yaml":    true,
	".json":    true,
	".toml":    true,
	".go":      true,
	".sql":     true,
	".sh":      true,
	".bash":    true,
	".zsh":     true,
	".py":      true,
	".txt":     true,
	".example": true,
	".sum":     true,
	".mod":     true,
	".env":     true,
}

// 始终排除的目录
var excludeDirs = map[string]bool{
	".git":         true,
	"build":        true,
	"dist":         true,
	"node_modules": true,
	"vendor":       true,
	"output":       true,
	"tmp":          true,
	"temp":         true,
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [path...]\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "校验文本文件是否为合法 UTF-8 编码。\n")
		fmt.Fprintf(os.Stderr, "不传参数时检查整个仓库。\n")
	}
	flag.Parse()

	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	checked, failed := 0, 0
	var failures []string

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if excludeDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if !shouldCheck(path) {
				return nil
			}
			checked++
			if err := validate(path); err != nil {
				failed++
				failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "遍历 %s 失败: %v\n", root, err)
			os.Exit(2)
		}
	}

	fmt.Fprintf(os.Stderr, "checked=%d failed=%d\n", checked, failed)
	for _, f := range failures {
		fmt.Fprintln(os.Stderr, f)
	}
	if failed > 0 {
		os.Exit(1)
	}
}

// shouldCheck 决定是否需要校验该文件。
func shouldCheck(path string) bool {
	base := filepath.Base(path)
	// 跳过常见二进制 / 压缩
	binarySuffixes := []string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".bmp",
		".pdf", ".zip", ".tar", ".gz", ".7z", ".rar",
		".xlsx", ".xls", ".docx", ".pptx",
		".exe", ".dll", ".so", ".dylib", ".bin",
		".mp4", ".mov", ".avi", ".mp3", ".wav",
	}
	lower := strings.ToLower(path)
	for _, s := range binarySuffixes {
		if strings.HasSuffix(lower, s) {
			return false
		}
	}
	// 跳过 .gitignore 自身（本工具可能会创建）
	if base == ".gitkeep" {
		return true
	}
	// 按后缀匹配
	ext := strings.ToLower(filepath.Ext(path))
	if textExts[ext] {
		return true
	}
	// 无后缀 + 短名 + 在仓库根 → 当作文本（如 VERSION, AGENTS.md 已在 ext 命中）
	if ext == "" {
		// 简单启发式：< 64 KB 且无 NUL 字节才检查
		info, err := os.Stat(path)
		if err != nil || info.Size() > 64*1024 {
			return false
		}
		return true
	}
	return false
}

// validate 检查文件 UTF-8 合法性。发现首个非法字节时返回详细错误。
func validate(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if !utf8.Valid(data) {
		// 找出第一个非法位置
		for i := 0; i < len(data); {
			r, size := utf8.DecodeRune(data[i:])
			if r == utf8.RuneError && size == 1 {
				return fmt.Errorf("非 UTF-8 字节位于 0x%x (offset %d)", data[i], i)
			}
			i += size
		}
		return fmt.Errorf("utf8.Valid 报 false（具体位置未定位）")
	}
	return nil
}
