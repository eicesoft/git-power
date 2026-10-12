package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SyncResult 描述一次 Repositories 同步的统计结果。
type SyncResult struct {
	Found   int `json:"found"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
}

// foundRepo 是遍历中发现的一个仓库，dir 是它相对扫描根的父目录路径。
type foundRepo struct {
	path string
	name string
	dir  string
}

// SyncRepositories 扫描设置中的目录，把发现的 Git 仓库按目录结构登记到 Repositories。
// 遍历时一旦遇到 Git 仓库就登记该目录并停止下探，其内部不再扫描；
// 登记位置按去掉扫描目录前缀后的原目录结构逐层建目录，仓库自身取代对应层级的目录节点。
// 扫描范围内本次没发现的记录标记为缺失，交由前端用不同图标区分。
func (a *App) SyncRepositories() (SyncResult, error) {
	if a.projects == nil {
		return SyncResult{}, fmt.Errorf("本地仓库存储不可用")
	}
	roots := []string{}
	for _, dir := range a.GetSettings().ScanDirectories {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return SyncResult{}, fmt.Errorf("解析扫描目录失败 %q: %w", dir, err)
		}
		roots = append(roots, abs)
	}
	if len(roots) == 0 {
		return SyncResult{}, fmt.Errorf("请先在设置中配置扫描目录")
	}

	// 先只遍历收集，全部成功后再写库，出错时不会留下半成品的缺失标记。
	repos := []foundRepo{}
	for _, root := range roots {
		found, err := scanRoot(root)
		if err != nil {
			return SyncResult{}, err
		}
		repos = append(repos, found...)
	}

	for _, root := range roots {
		if err := a.projects.MarkMissingWithin(root); err != nil {
			return SyncResult{}, err
		}
	}
	var result SyncResult
	folders := map[string]*int64{}
	for _, repo := range repos {
		folderID, err := ensureFolderChain(a.projects, folders, repo.dir)
		if err != nil {
			return SyncResult{}, err
		}
		created, err := a.projects.SyncProject(repo.path, repo.name, folderID)
		if err != nil {
			return SyncResult{}, err
		}
		if created {
			result.Added++
		} else {
			result.Updated++
		}
		result.Found++
	}
	return result, nil
}

// scanRoot 返回扫描根下发现的全部仓库；只读遍历，不写库。
func scanRoot(root string) ([]foundRepo, error) {
	found := []foundRepo{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// 读不出来的子树直接跳过，不中断整次同步。
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.IsDir() || !isGitRepository(path) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found = append(found, foundRepo{path: path, name: entry.Name(), dir: filepath.Dir(rel)})
		return fs.SkipDir
	})
	if err != nil {
		return nil, fmt.Errorf("扫描目录 %q 失败: %w", root, err)
	}
	return found, nil
}

// isGitRepository 判断目录本身是否为 Git 仓库：普通仓库的 .git 是目录，worktree 中是文件。
func isGitRepository(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// ensureFolderChain 按相对路径逐层建目录，返回最深层目录 ID。
// 相对路径为 "."（仓库就在扫描目录本身）时返回 nil，表示挂在根级。
func ensureFolderChain(store *ProjectStore, cache map[string]*int64, rel string) (*int64, error) {
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" {
		return nil, nil
	}
	if id, ok := cache[rel]; ok {
		return id, nil
	}
	parent, err := ensureFolderChain(store, cache, filepath.Dir(rel))
	if err != nil {
		return nil, err
	}
	folder, err := store.EnsureFolder(filepath.Base(rel), parent)
	if err != nil {
		return nil, err
	}
	id := folder.Id
	cache[rel] = &id
	return &id, nil
}
