package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 提供仓库读取和常用 Git 操作。
type App struct {
	ctx        context.Context
	logMu      sync.Mutex
	gitLogs    []GitCommandLog
	logCounter int64
	projects   *ProjectStore
	settingsMu sync.Mutex
	settings   Settings
}

// gitAction 标识前端请求的具体远程或临时保存操作。
type gitAction string

const (
	gitFetch gitAction = "fetch"
	gitPull  gitAction = "pull"
	gitPush  gitAction = "push"
	gitStash gitAction = "stash"
)

const maxGitCommandLogs = 200

type GitCommandLog struct {
	ID        int64  `json:"id"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	Error     string `json:"error"`
	StartedAt string `json:"startedAt"`
	Duration  int64  `json:"duration"`
	Success   bool   `json:"success"`
}

type Repository struct {
	Path          string   `json:"path"`
	Name          string   `json:"name"`
	CurrentBranch string   `json:"currentBranch"`
	AheadCommits  int      `json:"aheadCommits"`
	Detached      bool     `json:"detached"`
	Remotes       []string `json:"remotes"`
	Branches      []Branch `json:"branches"`
	Commits       []Commit `json:"commits"`
	Changes       []Change `json:"changes"`
}

type Branch struct {
	Name     string `json:"name"`
	Ref      string `json:"ref"`
	Hash     string `json:"hash"`
	Upstream string `json:"upstream"`
	Remote   bool   `json:"remote"`
	Current  bool   `json:"current"`
}

type Commit struct {
	Hash        string   `json:"hash"`
	ShortHash   string   `json:"shortHash"`
	Author      string   `json:"author"`
	Date        string   `json:"date"`
	Subject     string   `json:"subject"`
	Decorations string   `json:"decorations"`
	Parents     []string `json:"parents"`
}

type CommitFile struct {
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
	Status       string `json:"status"`
}

type CommitDetail struct {
	Hash      string       `json:"hash"`
	ShortHash string       `json:"shortHash"`
	Author    string       `json:"author"`
	Date      string       `json:"date"`
	Subject   string       `json:"subject"`
	Body      string       `json:"body"`
	Files     []CommitFile `json:"files"`
}

type Change struct {
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
	Status       string `json:"status"`
	Staged       bool   `json:"staged"`
	Unstaged     bool   `json:"unstaged"`
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if store, err := NewProjectStore(projectStorePath()); err != nil {
		log.Printf("初始化本地仓库存储失败: %v", err)
	} else {
		a.projects = store
	}
	settings, err := loadSettings(settingsPath())
	if err != nil {
		log.Printf("读取设置失败: %v", err)
	}
	a.settings = settings
	setGitCommand(settings.GitPath)
	setGitCommandLogSink(a.appendGitLog)
}

// GetSettings 返回当前应用设置。
func (a *App) GetSettings() Settings {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.settings
}

// SaveSettings 保存应用设置，并让 Git 客户端路径立即生效。
func (a *App) SaveSettings(settings Settings) error {
	settings = settings.normalized()
	if err := saveSettings(settingsPath(), settings); err != nil {
		return err
	}
	a.settingsMu.Lock()
	a.settings = settings
	a.settingsMu.Unlock()
	setGitCommand(settings.GitPath)
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
	path, err := exec.LookPath(gitCommand())
	if err != nil {
		return GitInfo{}, fmt.Errorf("未找到 Git 客户端 %q", gitCommand())
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

// GitConfigEntry 是全局 Git 配置中的一条键值。
type GitConfigEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetGitGlobalConfig 读取全局 Git 配置，供设置页展示与编辑。
func (a *App) GetGitGlobalConfig() ([]GitConfigEntry, error) {
	output, err := runGitCommand("config", "--global", "--list", "--null")
	if err != nil {
		// 尚未创建全局配置文件时 Git 会报错，此时视为没有配置。
		if strings.Contains(err.Error(), "unable to read config file") {
			return []GitConfigEntry{}, nil
		}
		return nil, err
	}
	return parseGitConfig(output), nil
}

// SetGitGlobalConfig 写入一条全局 Git 配置；value 为空时删除该配置项。
func (a *App) SetGitGlobalConfig(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("配置项名称不能为空")
	}
	if value = strings.TrimSpace(value); value == "" {
		// 删除不存在的配置项时 Git 会报错，先确认该配置项存在。
		if _, err := runGitCommand("config", "--global", "--get", key); err != nil {
			return nil
		}
		_, err := runGitCommand("config", "--global", "--unset", key)
		return err
	}
	_, err := runGitCommand("config", "--global", key, value)
	return err
}

// parseGitConfig 解析 git config --list --null 的 "key\nvalue\0" 输出。
func parseGitConfig(output string) []GitConfigEntry {
	entries := []GitConfigEntry{}
	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}
		key, value, _ := strings.Cut(record, "\n")
		entries = append(entries, GitConfigEntry{Key: key, Value: value})
	}
	return entries
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

// SelectRepository 打开系统目录选择器并读取所选 Git 仓库。
func (a *App) SelectRepository() (Repository, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                      "打开 Git 仓库",
		CanCreateDirectories:       false,
		ResolvesAliases:            true,
		TreatPackagesAsDirectories: true,
	})
	if err != nil {
		return Repository{}, err
	}
	if path == "" {
		return Repository{}, nil
	}
	return LoadRepository(path)
}

// RefreshRepository 重新读取仓库当前状态。
func (a *App) RefreshRepository(path string) (Repository, error) {
	return LoadRepository(path)
}

// RecordProject 登记一次仓库打开记录，供 Dashboard 的 Recent 与 Repositories 展示。
func (a *App) RecordProject(path, name string) error {
	if a.projects == nil {
		return fmt.Errorf("本地仓库存储不可用")
	}
	return a.projects.RecordProject(path, name)
}

// ListRecentProjects 返回最近打开的最多 limit 个仓库。
func (a *App) ListRecentProjects(limit int) ([]ProjectInfo, error) {
	if a.projects == nil {
		return []ProjectInfo{}, nil
	}
	return a.projects.RecentProjects(limit)
}

// ListDashboardRepositories 返回自定义目录与全部已登记仓库，供目录树展示。
func (a *App) ListDashboardRepositories() (DashboardData, error) {
	if a.projects == nil {
		return DashboardData{Folders: []Folder{}, Projects: []ProjectInfo{}}, nil
	}
	folders, err := a.projects.AllFolders()
	if err != nil {
		return DashboardData{}, err
	}
	projects, err := a.projects.AllProjects()
	if err != nil {
		return DashboardData{}, err
	}
	return DashboardData{Folders: folders, Projects: projects}, nil
}

// AddDashboardFolder 在 parentId 指定的目录下新建文件夹，parentId 为 0 时建在根级。
func (a *App) AddDashboardFolder(name string, parentId int64) (Folder, error) {
	if a.projects == nil {
		return Folder{}, fmt.Errorf("本地仓库存储不可用")
	}
	var parentID *int64
	if parentId > 0 {
		parentID = &parentId
	}
	return a.projects.AddFolder(name, parentID)
}

// RenameDashboardFolder 重命名自定义文件夹。
func (a *App) RenameDashboardFolder(id int64, name string) error {
	if a.projects == nil {
		return fmt.Errorf("本地仓库存储不可用")
	}
	return a.projects.RenameFolder(id, name)
}

// DeleteDashboardFolder 删除自定义文件夹，其中的仓库与子文件夹移到上一层。
func (a *App) DeleteDashboardFolder(id int64) error {
	if a.projects == nil {
		return fmt.Errorf("本地仓库存储不可用")
	}
	return a.projects.DeleteFolder(id)
}

// MoveDashboardProject 把仓库移动到指定文件夹，folderId 为 0 时移回根级。
func (a *App) MoveDashboardProject(path string, folderId int64) error {
	if a.projects == nil {
		return fmt.Errorf("本地仓库存储不可用")
	}
	var folderID *int64
	if folderId > 0 {
		folderID = &folderId
	}
	return a.projects.MoveProject(path, folderID)
}

// ProjectSummary 是 Dashboard 单击项目时展示的仓库概要信息。
type ProjectSummary struct {
	Path           string   `json:"path"`
	Name           string   `json:"name"`
	CurrentBranch  string   `json:"currentBranch"`
	Detached       bool     `json:"detached"`
	Remotes        []string `json:"remotes"`
	AheadCommits   int      `json:"aheadCommits"`
	LocalBranches  int      `json:"localBranches"`
	RemoteBranches int      `json:"remoteBranches"`
	Commits        int      `json:"commits"`
	Changes        int      `json:"changes"`
	LastCommit     *Commit  `json:"lastCommit"`
}

// GetProjectSummary 读取仓库概要信息，供 Dashboard 单击预览，不加载完整提交历史与文件更改。
func (a *App) GetProjectSummary(path string) (ProjectSummary, error) {
	if strings.TrimSpace(path) == "" {
		return ProjectSummary{}, fmt.Errorf("仓库路径不能为空")
	}

	root, err := runGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return ProjectSummary{}, fmt.Errorf("所选目录不是 Git 仓库: %w", err)
	}
	root = strings.TrimSpace(root)

	summary := ProjectSummary{Path: root, Name: filepath.Base(root), Remotes: []string{}}

	if current, err := runGit(root, "symbolic-ref", "--short", "HEAD"); err == nil {
		summary.CurrentBranch = strings.TrimSpace(current)
	} else if head, headErr := runGit(root, "rev-parse", "--short", "HEAD"); headErr == nil {
		summary.Detached = true
		summary.CurrentBranch = strings.TrimSpace(head)
	}

	summary.Remotes, err = loadRemotes(root)
	if err != nil {
		return ProjectSummary{}, err
	}
	branches, err := loadBranches(root, summary.CurrentBranch)
	if err != nil {
		return ProjectSummary{}, err
	}
	for _, branch := range branches {
		if branch.Remote {
			summary.RemoteBranches++
		} else {
			summary.LocalBranches++
		}
	}
	summary.AheadCommits = loadAheadCommits(root, branches)

	if output, err := runGit(root, "rev-list", "--count", "HEAD"); err == nil {
		summary.Commits, _ = strconv.Atoi(strings.TrimSpace(output))
	}
	if changes, err := loadChanges(root); err == nil {
		summary.Changes = len(changes)
	}
	if output, err := runGit(root, "log", "--max-count=1", "--format=%H%x1f%h%x1f%an%x1f%aI%x1f%s%x1f%D%x1f%P"); err == nil {
		fields := strings.SplitN(strings.TrimSpace(output), "\x1f", 7)
		if len(fields) == 7 {
			summary.LastCommit = &Commit{
				Hash:        fields[0],
				ShortHash:   fields[1],
				Author:      fields[2],
				Date:        fields[3],
				Subject:     fields[4],
				Decorations: fields[5],
				Parents:     parseParents(fields[6]),
			}
		}
	}
	return summary, nil
}

// GetGitCommandLogs 返回最近的 Git 命令日志。
func (a *App) GetGitCommandLogs() []GitCommandLog {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	logs := make([]GitCommandLog, len(a.gitLogs))
	copy(logs, a.gitLogs)
	return logs
}

// ClearGitCommandLogs 清空 Git 命令日志。
func (a *App) ClearGitCommandLogs() []GitCommandLog {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	a.gitLogs = nil
	return []GitCommandLog{}
}

func (a *App) appendGitLog(log GitCommandLog) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	a.logCounter++
	log.ID = a.logCounter
	a.gitLogs = append(a.gitLogs, log)
	if len(a.gitLogs) > maxGitCommandLogs {
		a.gitLogs = a.gitLogs[len(a.gitLogs)-maxGitCommandLogs:]
	}
}

var (
	gitCommandLogMu   sync.RWMutex
	gitCommandLogSink func(GitCommandLog)
)

func setGitCommandLogSink(sink func(GitCommandLog)) {
	gitCommandLogMu.Lock()
	defer gitCommandLogMu.Unlock()
	gitCommandLogSink = sink
}

// LoadRepository 读取分支、提交历史和工作区更改。
func LoadRepository(path string) (Repository, error) {
	if strings.TrimSpace(path) == "" {
		return Repository{}, fmt.Errorf("仓库路径不能为空")
	}

	root, err := runGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repository{}, fmt.Errorf("所选目录不是 Git 仓库: %w", err)
	}
	root = strings.TrimSpace(root)

	bare, err := runGit(root, "rev-parse", "--is-bare-repository")
	if err == nil && strings.TrimSpace(bare) == "true" {
		return Repository{}, fmt.Errorf("暂不支持裸仓库")
	}

	repo := Repository{
		Path:     root,
		Name:     filepath.Base(root),
		Remotes:  []string{},
		Branches: []Branch{},
		Commits:  []Commit{},
		Changes:  []Change{},
	}

	if current, err := runGit(root, "symbolic-ref", "--short", "HEAD"); err == nil {
		repo.CurrentBranch = strings.TrimSpace(current)
	} else if head, headErr := runGit(root, "rev-parse", "--short", "HEAD"); headErr == nil {
		repo.Detached = true
		repo.CurrentBranch = strings.TrimSpace(head)
	}

	repo.Remotes, err = loadRemotes(root)
	if err != nil {
		return Repository{}, err
	}
	repo.Branches, err = loadBranches(root, repo.CurrentBranch)
	if err != nil {
		return Repository{}, err
	}
	repo.AheadCommits = loadAheadCommits(root, repo.Branches)
	repo.Commits, err = loadCommits(root)
	if err != nil {
		return Repository{}, err
	}
	repo.Changes, err = loadChanges(root)
	if err != nil {
		return Repository{}, err
	}
	return repo, nil
}

// SwitchBranch 切换本地分支，或为远程分支创建跟踪分支。
func (a *App) SwitchBranch(path, branch string, remote bool) (Repository, error) {
	if strings.TrimSpace(branch) == "" {
		return Repository{}, fmt.Errorf("分支名不能为空")
	}

	var err error
	if remote {
		_, err = runGit(path, "switch", "--track", branch)
		if err != nil && strings.Contains(err.Error(), "already exists") {
			localName := branch
			if parts := strings.SplitN(branch, "/", 2); len(parts) == 2 {
				localName = parts[1]
			}
			_, err = runGit(path, "switch", localName)
		}
	} else {
		_, err = runGit(path, "switch", branch)
	}
	if err != nil {
		return Repository{}, fmt.Errorf("切换分支失败: %w", err)
	}
	return LoadRepository(path)
}

// CreateBranch 新建并切换到指定分支。
func (a *App) CreateBranch(path, branch string) (Repository, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return Repository{}, fmt.Errorf("分支名不能为空")
	}
	if _, err := runGit(path, "switch", "-c", branch); err != nil {
		return Repository{}, fmt.Errorf("新建分支失败: %w", err)
	}
	return LoadRepository(path)
}

// StageChange 暂存单个文件，包含重命名的原路径。
func (a *App) StageChange(path, file, original string) (Repository, error) {
	args := []string{"add", "--", file}
	if original != "" && original != file {
		args = append(args, original)
	}
	if _, err := runGit(path, args...); err != nil {
		return Repository{}, fmt.Errorf("暂存更改失败: %w", err)
	}
	return LoadRepository(path)
}

// UnstageChange 取消暂存单个文件，包含重命名的原路径。
func (a *App) UnstageChange(path, file, original string) (Repository, error) {
	args := []string{"reset", "-q", "--", file}
	if original != "" && original != file {
		args = append(args, original)
	}
	if _, err := runGit(path, args...); err != nil {
		return Repository{}, fmt.Errorf("取消暂存失败: %w", err)
	}
	return LoadRepository(path)
}

// Commit 提交当前暂存区内容。
func (a *App) Commit(path, message string) (Repository, error) {
	if strings.TrimSpace(message) == "" {
		return Repository{}, fmt.Errorf("提交说明不能为空")
	}
	if _, err := runGit(path, "commit", "-m", message); err != nil {
		return Repository{}, fmt.Errorf("提交失败: %w", err)
	}
	return LoadRepository(path)
}

// GitAction 执行顶部工具栏提供的常用 Git 操作。
func (a *App) GitAction(path, action string) (Repository, error) {
	gitCommands := map[gitAction][]string{
		gitFetch: {"fetch", "--prune"},
		gitPull:  {"pull", "--ff-only"},
		gitPush:  {"push"},
		gitStash: {"stash", "push", "--include-untracked"},
	}
	args, exists := gitCommands[gitAction(strings.TrimSpace(action))]
	if !exists {
		return Repository{}, fmt.Errorf("不支持的 Git 操作")
	}
	if _, err := runGit(path, args...); err != nil {
		return Repository{}, fmt.Errorf("%s失败: %w", action, err)
	}
	return LoadRepository(path)
}

// Pull 拉取指定远程分支并合并或变基到当前分支，可选先暂存再恢复本地更改。
func (a *App) Pull(path, remote, branch string, rebase, stash bool) (Repository, error) {
	remote = strings.TrimSpace(remote)
	branch = strings.TrimSpace(branch)
	if remote == "" || branch == "" {
		return Repository{}, fmt.Errorf("远程和分支不能为空")
	}

	stashed := false
	if stash {
		status, err := runGit(path, "status", "--porcelain")
		if err != nil {
			return Repository{}, fmt.Errorf("读取工作区状态失败: %w", err)
		}
		if strings.TrimSpace(status) != "" {
			if _, err := runGit(path, "stash", "push", "--include-untracked"); err != nil {
				return Repository{}, fmt.Errorf("暂存本地更改失败: %w", err)
			}
			stashed = true
		}
	}

	pullArgs := []string{"pull"}
	if rebase {
		pullArgs = append(pullArgs, "--rebase")
	}
	pullArgs = append(pullArgs, remote, branch)
	if _, err := runGit(path, pullArgs...); err != nil {
		if stashed {
			if _, popErr := runGit(path, "stash", "pop"); popErr != nil {
				return Repository{}, fmt.Errorf("拉取失败且恢复暂存失败，请手动执行 git stash pop: %w", err)
			}
		}
		return Repository{}, fmt.Errorf("拉取失败: %w", err)
	}

	if stashed {
		if _, err := runGit(path, "stash", "pop"); err != nil {
			return Repository{}, fmt.Errorf("拉取成功，但恢复暂存更改失败，请手动执行 git stash pop: %w", err)
		}
	}
	return LoadRepository(path)
}

// Push 推送本地分支到远程分支，支持设置上游跟踪、推送全部标签和强制推送。
func (a *App) Push(path, remote, branch, target string, tracking, tags, force bool) (Repository, error) {
	remote = strings.TrimSpace(remote)
	branch = strings.TrimSpace(branch)
	target = strings.TrimSpace(target)
	if remote == "" || branch == "" {
		return Repository{}, fmt.Errorf("远程和分支不能为空")
	}
	if target == "" {
		target = branch
	}

	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	if tags {
		args = append(args, "--tags")
	}
	if tracking {
		args = append(args, "-u")
	}
	args = append(args, remote)
	if target == branch {
		args = append(args, branch)
	} else {
		args = append(args, branch+":"+target)
	}
	if _, err := runGit(path, args...); err != nil {
		return Repository{}, fmt.Errorf("推送失败: %w", err)
	}
	return LoadRepository(path)
}

// GetChangeDiff 返回工作区或暂存区中单个文件的文本差异。
func (a *App) GetChangeDiff(path, file, original, status string, staged bool) (string, error) {
	if file == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}

	if strings.HasPrefix(status, "?") {
		return runGitNoIndex(path, file)
	}

	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", ":(literal)"+file)
	if original != "" && original != file {
		args = append(args, ":(literal)"+original)
	}
	return runGit(path, args...)
}

// GetChangePath 返回更改文件在磁盘上的绝对路径。
func (a *App) GetChangePath(path, file string) (string, error) {
	return changeAbsPath(path, file)
}

// OpenChange 使用系统默认程序打开工作区中的单个文件。
func (a *App) OpenChange(path, file string) error {
	abs, err := changeAbsPath(path, file)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("文件不存在: %s", abs)
	}
	return openWithSystem(abs)
}

// DiscardChange 丢弃单个文件的未提交更改；未跟踪的文件或目录会被直接删除。
func (a *App) DiscardChange(path, file, original, status string) (Repository, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return Repository{}, fmt.Errorf("文件路径不能为空")
	}

	if strings.HasPrefix(strings.TrimSpace(status), "?") {
		abs, err := changeAbsPath(path, file)
		if err != nil {
			return Repository{}, err
		}
		if err := os.RemoveAll(abs); err != nil {
			return Repository{}, fmt.Errorf("删除未跟踪文件失败: %w", err)
		}
	} else {
		args := []string{"restore", "--source=HEAD", "--staged", "--worktree", "--", ":(literal)" + file}
		if original != "" && original != file {
			args = append(args, ":(literal)"+original)
		}
		if _, err := runGit(path, args...); err != nil {
			return Repository{}, fmt.Errorf("丢弃更改失败: %w", err)
		}
	}
	return LoadRepository(path)
}

// IgnoreChange 将单个文件或目录追加到仓库根目录的 .gitignore。
func (a *App) IgnoreChange(path, file string) (Repository, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return Repository{}, fmt.Errorf("文件路径不能为空")
	}

	entry := "/" + strings.TrimSuffix(file, "/")
	gitignorePath := filepath.Join(path, ".gitignore")
	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return Repository{}, fmt.Errorf("读取 .gitignore 失败: %w", err)
	}

	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == entry {
			return LoadRepository(path)
		}
	}

	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"
	if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
		return Repository{}, fmt.Errorf("写入 .gitignore 失败: %w", err)
	}
	return LoadRepository(path)
}

// changeAbsPath 将仓库内相对路径解析为仓库根目录内的绝对路径，并拒绝越界访问。
func changeAbsPath(path, file string) (string, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("解析仓库路径失败: %w", err)
	}
	abs := filepath.Clean(filepath.Join(root, filepath.FromSlash(file)))
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("文件路径越界")
	}
	return abs, nil
}

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

// GetCommitDetail 返回单个提交的说明和变更文件列表。
func (a *App) GetCommitDetail(path, hash string) (CommitDetail, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" || strings.IndexFunc(hash, func(r rune) bool {
		return !strings.ContainsRune("0123456789abcdefABCDEF", r)
	}) >= 0 {
		return CommitDetail{}, fmt.Errorf("无效的提交哈希")
	}

	metadata, err := runGit(
		path,
		"show",
		"-s",
		"--format=%H%x00%h%x00%an%x00%aI%x00%B%x00",
		hash,
	)
	if err != nil {
		return CommitDetail{}, fmt.Errorf("读取提交信息失败: %w", err)
	}
	fields := strings.SplitN(metadata, "\x00", 6)
	if len(fields) < 5 {
		return CommitDetail{}, fmt.Errorf("解析提交信息失败")
	}

	fullMessage := strings.TrimSuffix(fields[4], "\n")
	subject, body, _ := strings.Cut(fullMessage, "\n")

	fileOutput, err := runGit(path, "diff-tree", "--root", "--no-commit-id", "--name-status", "-r", "-z", hash)
	if err != nil {
		return CommitDetail{}, fmt.Errorf("读取提交文件失败: %w", err)
	}

	return CommitDetail{
		Hash:      fields[0],
		ShortHash: fields[1],
		Author:    fields[2],
		Date:      fields[3],
		Subject:   strings.TrimSpace(subject),
		Body:      strings.TrimSpace(body),
		Files:     parseCommitFiles(fileOutput),
	}, nil
}

func loadRemotes(path string) ([]string, error) {
	output, err := runGit(path, "remote")
	if err != nil {
		return nil, fmt.Errorf("读取远程仓库失败: %w", err)
	}
	remotes := []string{}
	for _, line := range strings.Split(output, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			remotes = append(remotes, name)
		}
	}
	return remotes, nil
}

func loadBranches(path, current string) ([]Branch, error) {
	output, err := runGit(
		path,
		"for-each-ref",
		"--format=%(refname:short)%09%(refname)%09%(objectname:short)%09%(upstream:short)",
		"refs/heads",
		"refs/remotes",
	)
	if err != nil {
		return nil, fmt.Errorf("读取分支失败: %w", err)
	}

	var branches []Branch
	for _, line := range strings.Split(strings.Trim(output, "\r\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			continue
		}
		remote := strings.HasPrefix(fields[1], "refs/remotes/")
		if remote && strings.HasSuffix(fields[0], "/HEAD") {
			continue
		}
		branches = append(branches, Branch{
			Name:     fields[0],
			Ref:      fields[1],
			Hash:     fields[2],
			Upstream: fields[3],
			Remote:   remote,
			Current:  !remote && fields[0] == current,
		})
	}
	if branches == nil {
		branches = []Branch{}
	}
	return branches, nil
}

// loadAheadCommits 返回当前分支相对 upstream 尚未推送的提交数。
func loadAheadCommits(path string, branches []Branch) int {
	for _, branch := range branches {
		if !branch.Current || branch.Upstream == "" {
			continue
		}
		output, err := runGit(path, "rev-list", "--count", branch.Upstream+"..HEAD")
		if err != nil {
			return 0
		}
		count, err := strconv.Atoi(strings.TrimSpace(output))
		if err != nil {
			return 0
		}
		return count
	}
	return 0
}

func loadCommits(path string) ([]Commit, error) {
	if _, err := runGit(path, "rev-parse", "--verify", "HEAD"); err != nil {
		return []Commit{}, nil
	}

	output, err := runGit(
		path,
		"log",
		"--max-count=500",
		"--format=%H%x1f%h%x1f%an%x1f%aI%x1f%s%x1f%D%x1f%P%x1e",
	)
	if err != nil {
		return nil, fmt.Errorf("读取提交历史失败: %w", err)
	}

	var commits []Commit
	for _, record := range strings.Split(output, "\x1e") {
		record = strings.Trim(record, "\r\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, "\x1f", 7)
		if len(fields) != 7 {
			continue
		}
		commits = append(commits, Commit{
			Hash:        fields[0],
			ShortHash:   fields[1],
			Author:      fields[2],
			Date:        fields[3],
			Subject:     fields[4],
			Decorations: fields[5],
			Parents:     parseParents(fields[6]),
		})
	}
	if commits == nil {
		commits = []Commit{}
	}
	return commits, nil
}

// parseParents 解析 %P 输出的空格分隔的父提交哈希列表。
func parseParents(output string) []string {
	if strings.TrimSpace(output) == "" {
		return []string{}
	}
	return strings.Fields(output)
}

func loadChanges(path string) ([]Change, error) {
	output, err := runGit(path, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("读取工作区更改失败: %w", err)
	}
	return parseStatus(output), nil
}

func parseStatus(output string) []Change {
	records := strings.Split(output, "\x00")
	changes := []Change{}
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) < 3 {
			continue
		}
		status := record[:2]
		file := record[3:]
		original := ""
		if strings.ContainsAny(status, "RC") && i+1 < len(records) {
			original = records[i+1]
			i++
		}
		changes = append(changes, Change{
			Path:         file,
			OriginalPath: original,
			Status:       status,
			Staged:       status[0] != ' ' && status != "??",
			Unstaged:     status[1] != ' ' || status == "??",
		})
	}
	return changes
}

func parseCommitFiles(output string) []CommitFile {
	records := strings.Split(output, "\x00")
	files := []CommitFile{}
	for i := 0; i < len(records); i++ {
		if records[i] == "" {
			continue
		}
		status := records[i]
		if i+1 >= len(records) {
			break
		}
		path := records[i+1]
		i++
		original := ""
		if strings.ContainsAny(status, "RC") && i+1 < len(records) {
			original = records[i+1]
			i++
		}
		files = append(files, CommitFile{
			Path:         path,
			OriginalPath: original,
			Status:       status,
		})
	}
	return files
}

func runGitNoIndex(path, file string) (string, error) {
	cmd := exec.Command(gitCommand(), "-C", path, "diff", "--no-index", "--", "/dev/null", file)
	startedAt := time.Now()
	output, err := cmd.CombinedOutput()
	commandArgs := []string{"-C", path, "diff", "--no-index", "--", "/dev/null", file}
	success := err == nil || (cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1)
	recordErr := err
	if success {
		recordErr = nil
	}
	recordGitCommand(commandArgs, output, recordErr, success, startedAt, time.Since(startedAt))
	if err != nil && (cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 1) {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

func runGit(path string, args ...string) (string, error) {
	return runGitCommand(append([]string{"-C", path}, args...)...)
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command(gitCommand(), args...)
	startedAt := time.Now()
	output, err := cmd.CombinedOutput()
	recordGitCommand(args, output, err, err == nil, startedAt, time.Since(startedAt))
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

func recordGitCommand(args []string, output []byte, commandErr error, success bool, startedAt time.Time, duration time.Duration) {
	gitCommandLogMu.RLock()
	sink := gitCommandLogSink
	gitCommandLogMu.RUnlock()
	if sink == nil {
		return
	}

	errorMessage := ""
	if commandErr != nil {
		errorMessage = strings.TrimSpace(formatGitOutput(output))
		if errorMessage == "" {
			errorMessage = commandErr.Error()
		}
	}

	sink(GitCommandLog{
		Command:   formatGitCommand(args),
		Output:    formatGitOutput(output),
		Error:     errorMessage,
		StartedAt: startedAt.Format(time.RFC3339Nano),
		Duration:  duration.Milliseconds(),
		Success:   success,
	})
}

func formatGitCommand(args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "git")
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\n\"") {
			parts = append(parts, strconv.Quote(arg))
			continue
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func formatGitOutput(output []byte) string {
	return strings.NewReplacer(
		"\x00", "\n",
		"\x1e", "\n",
		"\x1f", "\t",
	).Replace(string(output))
}
