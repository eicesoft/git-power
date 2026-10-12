package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ProjectInfo 描述 Dashboard 侧栏中展示的一个已登记仓库。
type ProjectInfo struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	LastOpenedAt string `json:"lastOpenedAt"`
	FolderId     *int64 `json:"folderId"`
}

// Folder 描述 Dashboard Repositories 中用户自定义的一个目录。
type Folder struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	ParentId *int64 `json:"parentId"`
}

// DashboardData 是 Dashboard Repositories 目录树的完整数据。
type DashboardData struct {
	Folders  []Folder      `json:"folders"`
	Projects []ProjectInfo `json:"projects"`
}

// ProjectStore 使用 SQLite 持久化打开过的仓库，供 Dashboard 的 Recent 与 Repositories 展示。
type ProjectStore struct {
	db *sql.DB
}

func NewProjectStore(path string) (*ProjectStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("数据库路径不能为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	// modernc.org/sqlite 同一连接串复用单连接，避免并发写入锁冲突。
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS projects (
		path TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		last_opened_at TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据表失败: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER REFERENCES folders(id) ON DELETE SET NULL,
		position INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化目录表失败: %w", err)
	}
	// 旧库缺少 folder_id 列时补充。
	hasFolderID := false
	if rows, err := db.Query(`PRAGMA table_info(projects)`); err == nil {
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err == nil && name == "folder_id" {
				hasFolderID = true
			}
		}
		rows.Close()
	}
	if !hasFolderID {
		if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN folder_id INTEGER REFERENCES folders(id) ON DELETE SET NULL`); err != nil {
			db.Close()
			return nil, fmt.Errorf("迁移仓库目录字段失败: %w", err)
		}
	}
	return &ProjectStore{db: db}, nil
}

// RecordProject 登记或刷新一次仓库打开记录。
func (s *ProjectStore) RecordProject(path, name string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("仓库路径不能为空")
	}
	path = filepath.Clean(path)
	if name == "" {
		name = filepath.Base(path)
	}
	_, err := s.db.Exec(
		`INSERT INTO projects (path, name, last_opened_at) VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET name = excluded.name, last_opened_at = excluded.last_opened_at`,
		path, name, time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("保存仓库记录失败: %w", err)
	}
	return nil
}

// RecentProjects 返回最近打开的 limit 个仓库。
func (s *ProjectStore) RecentProjects(limit int) ([]ProjectInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(
		`SELECT path, name, last_opened_at, folder_id FROM projects ORDER BY last_opened_at DESC, path DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("读取最近仓库失败: %w", err)
	}
	return scanProjects(rows)
}

// AllProjects 返回全部已登记仓库，按名称排序供目录树展示。
func (s *ProjectStore) AllProjects() ([]ProjectInfo, error) {
	rows, err := s.db.Query(`SELECT path, name, last_opened_at, folder_id FROM projects ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("读取仓库列表失败: %w", err)
	}
	return scanProjects(rows)
}

// AllFolders 返回全部自定义目录，按同层位置排序。
func (s *ProjectStore) AllFolders() ([]Folder, error) {
	rows, err := s.db.Query(`SELECT id, name, parent_id FROM folders ORDER BY position, id`)
	if err != nil {
		return nil, fmt.Errorf("读取目录列表失败: %w", err)
	}
	defer rows.Close()
	folders := []Folder{}
	for rows.Next() {
		var folder Folder
		if err := rows.Scan(&folder.Id, &folder.Name, &folder.ParentId); err != nil {
			return nil, fmt.Errorf("解析目录记录失败: %w", err)
		}
		folders = append(folders, folder)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历目录记录失败: %w", err)
	}
	return folders, nil
}

// AddFolder 新建一个自定义目录，parentID 为 nil 时挂在根级。
func (s *ProjectStore) AddFolder(name string, parentID *int64) (Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Folder{}, fmt.Errorf("目录名称不能为空")
	}
	if parentID != nil {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM folders WHERE id = ?`, *parentID).Scan(&exists); err != nil || exists == 0 {
			return Folder{}, fmt.Errorf("父目录不存在")
		}
	}
	var position int
	s.db.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM folders WHERE parent_id IS ?`, parentID).Scan(&position)
	result, err := s.db.Exec(`INSERT INTO folders (name, parent_id, position) VALUES (?, ?, ?)`, name, parentID, position)
	if err != nil {
		return Folder{}, fmt.Errorf("新建目录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Folder{}, fmt.Errorf("读取目录 ID 失败: %w", err)
	}
	return Folder{Id: id, Name: name, ParentId: parentID}, nil
}

// RenameFolder 修改自定义目录名称。
func (s *ProjectStore) RenameFolder(id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("目录名称不能为空")
	}
	result, err := s.db.Exec(`UPDATE folders SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("重命名目录失败: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("目录不存在")
	}
	return nil
}

// DeleteFolder 删除自定义目录，其中包含的仓库与子目录移到被删目录的父级。
func (s *ProjectStore) DeleteFolder(id int64) error {
	var parentID *int64
	if err := s.db.QueryRow(`SELECT parent_id FROM folders WHERE id = ?`, id).Scan(&parentID); err != nil {
		return fmt.Errorf("目录不存在")
	}
	_, err := s.db.Exec(`UPDATE projects SET folder_id = ? WHERE folder_id = ?`, parentID, id)
	if err != nil {
		return fmt.Errorf("迁移目录内仓库失败: %w", err)
	}
	_, err = s.db.Exec(`UPDATE folders SET parent_id = ? WHERE parent_id = ?`, parentID, id)
	if err != nil {
		return fmt.Errorf("迁移子目录失败: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM folders WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除目录失败: %w", err)
	}
	return nil
}

// MoveProject 把仓库移动到指定目录，folderID 为 nil 时移回根级。
func (s *ProjectStore) MoveProject(path string, folderID *int64) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("仓库路径不能为空")
	}
	path = filepath.Clean(path)
	if folderID != nil {
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM folders WHERE id = ?`, *folderID).Scan(&exists); err != nil || exists == 0 {
			return fmt.Errorf("目标目录不存在")
		}
	}
	result, err := s.db.Exec(`UPDATE projects SET folder_id = ? WHERE path = ?`, folderID, path)
	if err != nil {
		return fmt.Errorf("移动仓库失败: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("仓库未登记")
	}
	return nil
}

func scanProjects(rows *sql.Rows) ([]ProjectInfo, error) {
	defer rows.Close()
	projects := []ProjectInfo{}
	for rows.Next() {
		var project ProjectInfo
		if err := rows.Scan(&project.Path, &project.Name, &project.LastOpenedAt, &project.FolderId); err != nil {
			return nil, fmt.Errorf("解析仓库记录失败: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历仓库记录失败: %w", err)
	}
	return projects, nil
}

func (s *ProjectStore) Close() error {
	return s.db.Close()
}

// projectStorePath 返回应用数据目录下的 SQLite 数据库路径。
func projectStorePath() string {
	return filepath.Join(appConfigDir(), "projects.db")
}
