package git

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

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
		Tags:     []Tag{},
		Stashes:  []Stash{},
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
	repo.Tags, err = loadTags(root)
	if err != nil {
		return Repository{}, err
	}
	repo.Stashes, err = loadStashes(root)
	if err != nil {
		return Repository{}, err
	}
	repo.AheadCommits, repo.BehindCommits = currentSyncCounts(repo.Branches)
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
		// refs/remotes/origin/HEAD 的 short name 是 "origin"，只能按完整 refname 判断。
		if remote && strings.HasSuffix(fields[1], "/HEAD") {
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
		if !remote && fields[3] != "" {
			last := &branches[len(branches)-1]
			last.AheadCommits, last.BehindCommits = loadSyncCounts(path, fields[3], fields[1])
		}
	}
	if branches == nil {
		branches = []Branch{}
	}
	return branches, nil
}

// loadTags 读取标签，annotated 标签取其指向的提交哈希。
func loadTags(path string) ([]Tag, error) {
	output, err := runGit(
		path,
		"for-each-ref",
		"--format=%(refname:short)%09%(refname)%09%(objectname:short)%09%(*objectname:short)",
		"refs/tags",
	)
	if err != nil {
		return nil, fmt.Errorf("读取标签失败: %w", err)
	}

	var tags []Tag
	for _, line := range strings.Split(strings.Trim(output, "\r\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			continue
		}
		hash := fields[3]
		if hash == "" {
			hash = fields[2]
		}
		tags = append(tags, Tag{Name: fields[0], Ref: fields[1], Hash: hash})
	}
	if tags == nil {
		tags = []Tag{}
	}
	return tags, nil
}

// loadStashes 读取 stash 列表，按创建顺序倒序排列。
func loadStashes(path string) ([]Stash, error) {
	output, err := runGit(path, "stash", "list", "--format=%gd%x09%h%x09%gs")
	if err != nil {
		return nil, fmt.Errorf("读取 stash 失败: %w", err)
	}

	var stashes []Stash
	for _, line := range strings.Split(strings.Trim(output, "\r\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		stashes = append(stashes, Stash{Ref: fields[0], Hash: fields[1], Message: fields[2]})
	}
	if stashes == nil {
		stashes = []Stash{}
	}
	return stashes, nil
}

// loadSyncCounts 返回分支相对 upstream 尚未推送（ahead）与尚未拉取（behind）的提交数。
func loadSyncCounts(path, upstream, branch string) (ahead, behind int) {
	output, err := runGit(path, "rev-list", "--left-right", "--count", upstream+"..."+branch)
	if err != nil {
		return 0, 0
	}
	fields := strings.Fields(output)
	if len(fields) != 2 {
		return 0, 0
	}
	behind, err = strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0
	}
	ahead, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0
	}
	return ahead, behind
}

// currentSyncCounts 返回当前分支的未推送与未拉取提交数。
func currentSyncCounts(branches []Branch) (ahead, behind int) {
	for _, branch := range branches {
		if branch.Current {
			return branch.AheadCommits, branch.BehindCommits
		}
	}
	return 0, 0
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

func loadChanges(path string) ([]Change, error) {
	output, err := runGit(path, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("读取工作区更改失败: %w", err)
	}
	return parseStatus(output), nil
}
