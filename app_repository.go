package main

import (
	"fmt"
	"os"

	"git-power/internal/git"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectRepository 打开系统目录选择器并读取所选 Git 仓库。
func (a *App) SelectRepository() (git.Repository, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                      "打开 Git 仓库",
		CanCreateDirectories:       false,
		ResolvesAliases:            true,
		TreatPackagesAsDirectories: true,
	})
	if err != nil {
		return git.Repository{}, err
	}
	if path == "" {
		return git.Repository{}, nil
	}
	return git.LoadRepository(path)
}

// RefreshRepository 重新读取仓库当前状态。
func (a *App) RefreshRepository(path string) (git.Repository, error) {
	return git.LoadRepository(path)
}

// SwitchBranch 切换本地分支，或为远程分支创建跟踪分支。
func (a *App) SwitchBranch(path, branch string, remote bool) (git.Repository, error) {
	return git.SwitchBranch(path, branch, remote)
}

// CreateBranch 新建分支，checkout 为 true 时同时切换过去。
func (a *App) CreateBranch(path, branch string, checkout bool) (git.Repository, error) {
	return git.CreateBranch(path, branch, checkout)
}

// StageChange 暂存单个文件，包含重命名的原路径。
func (a *App) StageChange(path, file, original string) (git.Repository, error) {
	return git.StageChange(path, file, original)
}

// UnstageChange 取消暂存单个文件，包含重命名的原路径。
func (a *App) UnstageChange(path, file, original string) (git.Repository, error) {
	return git.UnstageChange(path, file, original)
}

// Commit 提交当前暂存区内容。
func (a *App) Commit(path, message string) (git.Repository, error) {
	return git.CommitChanges(path, message)
}

// GitAction 执行顶部工具栏提供的常用 Git 操作。
func (a *App) GitAction(path, action string) (git.Repository, error) {
	return git.GitAction(path, action)
}

// Pull 拉取指定远程分支并合并或变基到当前分支，可选先暂存再恢复本地更改。
func (a *App) Pull(path, remote, branch string, rebase, stash bool) (git.Repository, error) {
	return git.Pull(path, remote, branch, rebase, stash)
}

// Push 推送本地分支到远程分支，支持设置上游跟踪、推送全部标签和强制推送。
func (a *App) Push(path, remote, branch, target string, tracking, tags, force bool) (git.Repository, error) {
	return git.Push(path, remote, branch, target, tracking, tags, force)
}

// GetChangeDiff 返回工作区或暂存区中单个文件的文本差异。
func (a *App) GetChangeDiff(path, file, original, status string, staged bool) (string, error) {
	return git.Diff(path, file, original, status, staged)
}

// GetChangePath 返回更改文件在磁盘上的绝对路径。
func (a *App) GetChangePath(path, file string) (string, error) {
	return git.AbsPath(path, file)
}

// OpenChange 使用系统默认程序打开工作区中的单个文件。
func (a *App) OpenChange(path, file string) error {
	abs, err := git.AbsPath(path, file)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("文件不存在: %s", abs)
	}
	return openWithSystem(abs)
}

// DiscardChange 丢弃单个文件的未提交更改；未跟踪的文件或目录会被直接删除。
func (a *App) DiscardChange(path, file, original, status string) (git.Repository, error) {
	return git.DiscardChange(path, file, original, status)
}

// IgnoreChange 将单个文件或目录追加到仓库根目录的 .gitignore。
func (a *App) IgnoreChange(path, file string) (git.Repository, error) {
	return git.IgnoreChange(path, file)
}

// GetCommitDetail 返回单个提交的说明和变更文件列表。
func (a *App) GetCommitDetail(path, hash string) (git.CommitDetail, error) {
	return git.LoadCommitDetail(path, hash)
}

// GetGitGlobalConfig 读取全局 Git 配置，供设置页展示与编辑。
func (a *App) GetGitGlobalConfig() ([]git.GitConfigEntry, error) {
	return git.LoadGlobalConfig()
}

// SetGitGlobalConfig 写入一条全局 Git 配置；value 为空时删除该配置项。
func (a *App) SetGitGlobalConfig(key, value string) error {
	return git.SetGlobalConfig(key, value)
}
