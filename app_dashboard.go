package main

import (
	"fmt"

	"git-power/internal/git"
	"git-power/internal/scan"
	"git-power/internal/store"
)

// RecordProject 登记一次仓库打开记录，供 Dashboard 的 Recent 与 Repositories 展示。
func (a *App) RecordProject(path, name string) error {
	if a.projects == nil {
		return fmt.Errorf("本地仓库存储不可用")
	}
	return a.projects.RecordProject(path, name)
}

// ListRecentProjects 返回最近打开的最多 limit 个仓库。
func (a *App) ListRecentProjects(limit int) ([]store.ProjectInfo, error) {
	if a.projects == nil {
		return []store.ProjectInfo{}, nil
	}
	return a.projects.RecentProjects(limit)
}

// ListDashboardRepositories 返回自定义目录与全部已登记仓库，供目录树展示。
func (a *App) ListDashboardRepositories() (store.DashboardData, error) {
	if a.projects == nil {
		return store.DashboardData{Folders: []store.Folder{}, Projects: []store.ProjectInfo{}}, nil
	}
	folders, err := a.projects.AllFolders()
	if err != nil {
		return store.DashboardData{}, err
	}
	projects, err := a.projects.AllProjects()
	if err != nil {
		return store.DashboardData{}, err
	}
	return store.DashboardData{Folders: folders, Projects: projects}, nil
}

// AddDashboardFolder 在 parentId 指定的目录下新建文件夹，parentId 为 0 时建在根级。
func (a *App) AddDashboardFolder(name string, parentId int64) (store.Folder, error) {
	if a.projects == nil {
		return store.Folder{}, fmt.Errorf("本地仓库存储不可用")
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

// GetProjectSummary 读取仓库概要信息，供 Dashboard 单击预览。
func (a *App) GetProjectSummary(path string) (git.ProjectSummary, error) {
	return git.LoadProjectSummary(path)
}

// GetProjectReadme 读取仓库根目录的 README 文件。
func (a *App) GetProjectReadme(path string) (git.ProjectReadme, error) {
	return git.LoadProjectReadme(path)
}

// SyncRepositories 扫描设置中的目录，把发现的 Git 仓库按目录结构登记到 Repositories。
func (a *App) SyncRepositories() (scan.SyncResult, error) {
	return scan.Sync(a.projects, a.GetSettings().ScanDirectories)
}
