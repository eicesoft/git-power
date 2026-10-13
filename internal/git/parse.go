package git

import "strings"

// parseParents 解析 %P 输出的空格分隔的父提交哈希列表。
func parseParents(output string) []string {
	if strings.TrimSpace(output) == "" {
		return []string{}
	}
	return strings.Fields(output)
}

// parseStatus 解析 git status --porcelain=v1 -z 的输出。
// 重命名与复制会多跟一条以 NUL 分隔的原路径记录。
func parseStatus(output string) []Change {
	records := strings.Split(output, "\x00")
	changes := []Change{}
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) < 3 {
			continue
		}
		status := record[:2]
		file := record[3:]
		original := ""
		if strings.ContainsAny(status, "RC") && i+1 < len(records) {
			original = records[i+1]
			i++
		}
		changes = append(changes, Change{
			Path:         file,
			OriginalPath: original,
			Status:       status,
			Staged:       status[0] != ' ' && status != "??",
			Unstaged:     status[1] != ' ' || status == "??",
		})
	}
	return changes
}

// parseCommitFiles 解析 git diff-tree --name-status -z 的输出。
func parseCommitFiles(output string) []CommitFile {
	records := strings.Split(output, "\x00")
	files := []CommitFile{}
	for i := 0; i < len(records); i++ {
		if records[i] == "" {
			continue
		}
		status := records[i]
		if i+1 >= len(records) {
			break
		}
		path := records[i+1]
		i++
		original := ""
		if strings.ContainsAny(status, "RC") && i+1 < len(records) {
			original = records[i+1]
			i++
		}
		files = append(files, CommitFile{
			Path:         path,
			OriginalPath: original,
			Status:       status,
		})
	}
	return files
}

// parseGitConfig 解析 git config --list --null 的 "key\nvalue\0" 输出。
func parseGitConfig(output string) []GitConfigEntry {
	entries := []GitConfigEntry{}
	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}
		key, value, _ := strings.Cut(record, "\n")
		entries = append(entries, GitConfigEntry{Key: key, Value: value})
	}
	return entries
}
