package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Diff 返回工作区或暂存区中单个文件的文本差异。
func Diff(path, file, original, status string, staged bool) (string, error) {
	if file == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}

	if strings.HasPrefix(status, "?") {
		return runGitNoIndex(path, file)
	}

	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", ":(literal)"+file)
	if original != "" && original != file {
		args = append(args, ":(literal)"+original)
	}
	return runGit(path, args...)
}

// LoadCommitDetail 返回单个提交的说明和变更文件列表。
func LoadCommitDetail(path, hash string) (CommitDetail, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" || strings.IndexFunc(hash, func(r rune) bool {
		return !strings.ContainsRune("0123456789abcdefABCDEF", r)
	}) >= 0 {
		return CommitDetail{}, fmt.Errorf("无效的提交哈希")
	}

	metadata, err := runGit(
		path,
		"show",
		"-s",
		"--format=%H%x00%h%x00%an%x00%aI%x00%B%x00",
		hash,
	)
	if err != nil {
		return CommitDetail{}, fmt.Errorf("读取提交信息失败: %w", err)
	}
	fields := strings.SplitN(metadata, "\x00", 6)
	if len(fields) < 5 {
		return CommitDetail{}, fmt.Errorf("解析提交信息失败")
	}

	fullMessage := strings.TrimSuffix(fields[4], "\n")
	subject, body, _ := strings.Cut(fullMessage, "\n")

	fileOutput, err := runGit(path, "diff-tree", "--root", "--no-commit-id", "--name-status", "-r", "-z", hash)
	if err != nil {
		return CommitDetail{}, fmt.Errorf("读取提交文件失败: %w", err)
	}

	return CommitDetail{
		Hash:      fields[0],
		ShortHash: fields[1],
		Author:    fields[2],
		Date:      fields[3],
		Subject:   strings.TrimSpace(subject),
		Body:      strings.TrimSpace(body),
		Files:     parseCommitFiles(fileOutput),
	}, nil
}

// AbsPath 将仓库内相对路径解析为仓库根目录内的绝对路径，并拒绝越界访问。
func AbsPath(path, file string) (string, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", fmt.Errorf("文件路径不能为空")
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("解析仓库路径失败: %w", err)
	}
	abs := filepath.Clean(filepath.Join(root, filepath.FromSlash(file)))
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("文件路径越界")
	}
	return abs, nil
}
