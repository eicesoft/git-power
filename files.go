package main

import (
	"fmt"
	"os/exec"
	goruntime "runtime"
)

// openWithSystem 使用系统默认程序打开指定路径。
func openWithSystem(abs string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", abs)
	case "darwin":
		cmd = exec.Command("open", abs)
	default:
		cmd = exec.Command("xdg-open", abs)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	return cmd.Process.Release()
}
