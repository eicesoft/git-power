package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadProjectSummary 读取仓库概要信息，供 Dashboard 单击预览，不加载完整提交历史与文件更改。
func LoadProjectSummary(path string) (ProjectSummary, error) {
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
	summary.AheadCommits, _ = currentSyncCounts(branches)

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

// LoadProjectReadme 读取仓库根目录的 README 文件，文件名不区分大小写。
func LoadProjectReadme(path string) (ProjectReadme, error) {
	if strings.TrimSpace(path) == "" {
		return ProjectReadme{}, fmt.Errorf("仓库路径不能为空")
	}

	root, err := runGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return ProjectReadme{}, fmt.Errorf("所选目录不是 Git 仓库: %w", err)
	}
	root = strings.TrimSpace(root)

	entries, err := os.ReadDir(root)
	if err != nil {
		return ProjectReadme{}, fmt.Errorf("读取仓库目录失败: %w", err)
	}
	name, rank := "", len(readmeRanks)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if !strings.EqualFold(strings.TrimSuffix(entry.Name(), ext), "readme") {
			continue
		}
		entryRank, ok := readmeRanks[strings.ToLower(ext)]
		if !ok || entryRank >= rank {
			continue
		}
		name, rank = entry.Name(), entryRank
	}
	if name == "" {
		return ProjectReadme{}, nil
	}

	content, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return ProjectReadme{}, fmt.Errorf("读取 %s 失败: %w", name, err)
	}
	return ProjectReadme{Name: name, Content: string(content)}, nil
}
