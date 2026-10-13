package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"git-power/internal/config"
	"git-power/internal/git"
	"git-power/internal/store"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是暴露给前端的 Wails 绑定对象，方法体只做参数转发与状态存取。
type App struct {
	ctx        context.Context
	logMu      sync.Mutex
	gitLogs    []git.GitCommandLog
	logCounter int64
	projects   *store.ProjectStore
	settingsMu sync.Mutex
	settings   config.Settings
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if projects, err := store.NewProjectStore(config.StorePath()); err != nil {
		log.Printf("初始化本地仓库存储失败: %v", err)
	} else {
		a.projects = projects
	}
	settings, err := config.Load(config.SettingsPath())
	if err != nil {
		log.Printf("读取设置失败: %v", err)
	}
	a.settings = settings
	git.SetCommandPath(settings.GitPath)
	git.SetLogSink(a.appendGitLog)
}

// GetSettings 返回当前应用设置。
func (a *App) GetSettings() config.Settings {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.settings
}

// SaveSettings 保存应用设置，并让 Git 客户端路径立即生效。
func (a *App) SaveSettings(settings config.Settings) error {
	settings = settings.Normalized()
	if err := config.Save(config.SettingsPath(), settings); err != nil {
		return err
	}
	a.settingsMu.Lock()
	a.settings = settings
	a.settingsMu.Unlock()
	git.SetCommandPath(settings.GitPath)
	return nil
}

// SelectScanDirectory 打开系统目录选择器，返回用户选择的扫描目录。
func (a *App) SelectScanDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                      "选择扫描目录",
		CanCreateDirectories:       true,
		ResolvesAliases:            true,
		TreatPackagesAsDirectories: true,
	})
}

// GitInfo 描述设置页展示的 Git 客户端信息。
type GitInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// GetGitInfo 返回当前 Git 客户端的可执行文件路径与版本号。
func (a *App) GetGitInfo() (GitInfo, error) {
	path, err := exec.LookPath(git.Command())
	if err != nil {
		return GitInfo{}, fmt.Errorf("未找到 Git 客户端 %q", git.Command())
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	output, err := exec.Command(path, "--version").Output()
	if err != nil {
		return GitInfo{}, fmt.Errorf("执行 %s 失败: %w", path, err)
	}
	return GitInfo{Path: path, Version: strings.TrimSpace(string(output))}, nil
}

// GetGitCommandLogs 返回最近的 Git 命令日志。
func (a *App) GetGitCommandLogs() []git.GitCommandLog {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	logs := make([]git.GitCommandLog, len(a.gitLogs))
	copy(logs, a.gitLogs)
	return logs
}

// ClearGitCommandLogs 清空 Git 命令日志。
func (a *App) ClearGitCommandLogs() []git.GitCommandLog {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	a.gitLogs = nil
	return []git.GitCommandLog{}
}

func (a *App) appendGitLog(log git.GitCommandLog) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	a.logCounter++
	log.ID = a.logCounter
	a.gitLogs = append(a.gitLogs, log)
	if len(a.gitLogs) > git.MaxCommandLogs {
		a.gitLogs = a.gitLogs[len(a.gitLogs)-git.MaxCommandLogs:]
	}
}

// WindowMinimise 最小化窗口。
func (a *App) WindowMinimise() {
	runtime.WindowMinimise(a.ctx)
}

// WindowToggleMaximise 在最大化与还原之间切换窗口。
func (a *App) WindowToggleMaximise() {
	if runtime.WindowIsMaximised(a.ctx) {
		runtime.WindowUnmaximise(a.ctx)
	} else {
		runtime.WindowMaximise(a.ctx)
	}
}

// WindowIsMaximised 查询窗口是否已最大化。
func (a *App) WindowIsMaximised() bool {
	return runtime.WindowIsMaximised(a.ctx)
}

// WindowClose 关闭窗口。
func (a *App) WindowClose() {
	runtime.Quit(a.ctx)
}
