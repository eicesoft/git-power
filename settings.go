package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func defaultSettings() Settings {
	return Settings{
		ScanDirectories: []string{},
		DiffFontSize:    defaultDiffFontSize,
		RecentLimit:     defaultRecentLimit,
		ConfirmDiscard:  true,
	}
}

// normalized 补齐空值与越界值，保证前后端拿到的设置始终可用。
func (s Settings) normalized() Settings {
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

// loadSettings 读取设置文件，文件不存在时返回默认设置，文件中缺失的字段保留默认值。
func loadSettings(path string) (Settings, error) {
	settings := defaultSettings()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, fmt.Errorf("读取设置失败: %w", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), fmt.Errorf("解析设置失败: %w", err)
	}
	return settings.normalized(), nil
}

// saveSettings 写入设置文件；文件中包含明文 API Key，因此只给当前用户读写权限。
func saveSettings(path string, settings Settings) error {
	data, err := json.MarshalIndent(settings.normalized(), "", "  ")
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

// appConfigDir 返回应用数据目录。
func appConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base, err = os.UserHomeDir()
		if err != nil {
			base = "."
		}
	}
	return filepath.Join(base, "git-power")
}

// settingsPath 返回应用数据目录下的设置文件路径。
func settingsPath() string {
	return filepath.Join(appConfigDir(), "settings.json")
}

// gitCommandPath 保存设置页配置的 Git 可执行文件路径，空值表示使用 PATH 中的 git。
var (
	gitCommandMu   sync.RWMutex
	gitCommandPath string
)

func gitCommand() string {
	gitCommandMu.RLock()
	defer gitCommandMu.RUnlock()
	if gitCommandPath == "" {
		return "git"
	}
	return gitCommandPath
}

func setGitCommand(path string) {
	gitCommandMu.Lock()
	defer gitCommandMu.Unlock()
	gitCommandPath = strings.TrimSpace(path)
}
