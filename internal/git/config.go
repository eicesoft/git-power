package git

import (
	"fmt"
	"strings"
)

// LoadGlobalConfig 读取全局 Git 配置，供设置页展示与编辑。
func LoadGlobalConfig() ([]GitConfigEntry, error) {
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

// SetGlobalConfig 写入一条全局 Git 配置；value 为空时删除该配置项。
func SetGlobalConfig(key, value string) error {
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
