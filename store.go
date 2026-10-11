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
		`SELECT path, name, last_opened_at FROM projects ORDER BY last_opened_at DESC, path DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("读取最近仓库失败: %w", err)
	}
	return scanProjects(rows)
}

// AllProjects 返回全部已登记仓库，按路径排序供目录树展示。
func (s *ProjectStore) AllProjects() ([]ProjectInfo, error) {
	rows, err := s.db.Query(`SELECT path, name, last_opened_at FROM projects ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("读取仓库列表失败: %w", err)
	}
	return scanProjects(rows)
}

func scanProjects(rows *sql.Rows) ([]ProjectInfo, error) {
	defer rows.Close()
	projects := []ProjectInfo{}
	for rows.Next() {
		var project ProjectInfo
		if err := rows.Scan(&project.Path, &project.Name, &project.LastOpenedAt); err != nil {
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
	base, err := os.UserConfigDir()
	if err != nil {
		base, err = os.UserHomeDir()
		if err != nil {
			base = "."
		}
	}
	return filepath.Join(base, "git-power", "projects.db")
}
