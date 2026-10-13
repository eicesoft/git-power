// Package testutil 提供跨包共用的测试辅助函数。
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Git 在 path 下执行一条 git 命令，失败即终止测试。
func Git(t *testing.T, path string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

// Write 在 root 下写入文件，缺失的父目录会被创建。
func Write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Read 读取 root 下文件的内容。
func Read(t *testing.T, root, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// MkRepo 在 path 下初始化一个空 Git 仓库。
func MkRepo(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	Git(t, path, "init", "-q")
}
