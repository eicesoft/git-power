// Package config 负责应用设置的读写与应用数据目录的路径解析。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultRecentLimit  = 10
	defaultDiffFontSize = 12
)

// Settings 是设置页可持久化的全部配置，保存在应用数据目录的 settings.json。
type Settings struct {
	ScanDirectories []string `json:"scanDirectories"`
	DiffFontFamily  string   `json:"diffFontFamily"`
	DiffFontSize    int      `json:"diffFontSize"`
	RecentLimit     int      `json:"recentLimit"`
	ConfirmDiscard  bool     `json:"confirmDiscard"`
	GitPath         string   `json:"gitPath"`
	AgentBaseURL    string   `json:"agentBaseUrl"`
	AgentAPIKey     string   `json:"agentApiKey"`
	AgentModel      string   `json:"agentModel"`
}

// Default 返回未持久化任何配置时使用的默认设置。
func Default() Settings {
	return Settings{
		ScanDirectories: []string{},
		DiffFontSize:    defaultDiffFontSize,
		RecentLimit:     defaultRecentLimit,
		ConfirmDiscard:  true,
	}
}

// Normalized 补齐空值与越界值，保证前后端拿到的设置始终可用。
func (s Settings) Normalized() Settings {
	if s.ScanDirectories == nil {
		s.ScanDirectories = []string{}
	}
	if s.DiffFontSize <= 0 {
		s.DiffFontSize = defaultDiffFontSize
	}
	if s.RecentLimit <= 0 {
		s.RecentLimit = defaultRecentLimit
	}
	return s
}

// Load 读取设置文件，文件不存在时返回默认设置，文件中缺失的字段保留默认值。
func Load(path string) (Settings, error) {
	settings := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, fmt.Errorf("读取设置失败: %w", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return Default(), fmt.Errorf("解析设置失败: %w", err)
	}
	return settings.Normalized(), nil
}

// Save 写入设置文件；文件中包含明文 API Key，因此只给当前用户读写权限。
func Save(path string, settings Settings) error {
	data, err := json.MarshalIndent(settings.Normalized(), "", "  ")
	if err != nil {
		return fmt.Errorf("序列化设置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}
	return nil
}

// DataDir 返回应用数据目录。
func DataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base, err = os.UserHomeDir()
		if err != nil {
			base = "."
		}
	}
	return filepath.Join(base, "git-power")
}

// SettingsPath 返回应用数据目录下的设置文件路径。
func SettingsPath() string {
	return filepath.Join(DataDir(), "settings.json")
}

// StorePath 返回应用数据目录下的 SQLite 数据库路径。
func StorePath() string {
	return filepath.Join(DataDir(), "projects.db")
}
