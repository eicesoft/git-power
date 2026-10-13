package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SwitchBranch 切换本地分支，或为远程分支创建跟踪分支。
func SwitchBranch(path, branch string, remote bool) (Repository, error) {
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

// CreateBranch 新建分支，checkout 为 true 时同时切换过去。
func CreateBranch(path, branch string, checkout bool) (Repository, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return Repository{}, fmt.Errorf("分支名不能为空")
	}
	if checkout {
		if _, err := runGit(path, "switch", "-c", branch); err != nil {
			return Repository{}, fmt.Errorf("新建分支失败: %w", err)
		}
	} else if _, err := runGit(path, "branch", branch); err != nil {
		return Repository{}, fmt.Errorf("新建分支失败: %w", err)
	}
	return LoadRepository(path)
}

// StageChange 暂存单个文件，包含重命名的原路径。
func StageChange(path, file, original string) (Repository, error) {
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
func UnstageChange(path, file, original string) (Repository, error) {
	args := []string{"reset", "-q", "--", file}
	if original != "" && original != file {
		args = append(args, original)
	}
	if _, err := runGit(path, args...); err != nil {
		return Repository{}, fmt.Errorf("取消暂存失败: %w", err)
	}
	return LoadRepository(path)
}

// CommitChanges 提交当前暂存区内容。
func CommitChanges(path, message string) (Repository, error) {
	if strings.TrimSpace(message) == "" {
		return Repository{}, fmt.Errorf("提交说明不能为空")
	}
	if _, err := runGit(path, "commit", "-m", message); err != nil {
		return Repository{}, fmt.Errorf("提交失败: %w", err)
	}
	return LoadRepository(path)
}

// GitAction 执行顶部工具栏提供的常用 Git 操作。
func GitAction(path, action string) (Repository, error) {
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
func Pull(path, remote, branch string, rebase, stash bool) (Repository, error) {
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
func Push(path, remote, branch, target string, tracking, tags, force bool) (Repository, error) {
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

// DiscardChange 丢弃单个文件的未提交更改；未跟踪的文件或目录会被直接删除。
func DiscardChange(path, file, original, status string) (Repository, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return Repository{}, fmt.Errorf("文件路径不能为空")
	}

	if strings.HasPrefix(strings.TrimSpace(status), "?") {
		abs, err := AbsPath(path, file)
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
func IgnoreChange(path, file string) (Repository, error) {
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
