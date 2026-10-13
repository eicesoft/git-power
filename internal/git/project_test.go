package git

import (
	"path/filepath"
	"testing"

	"git-power/internal/testutil"
)

// Dashboard 单击项目要展示 README，仓库没有 README 时返回空文件名。
func TestLoadProjectReadme(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")

	readme, err := LoadProjectReadme(root)
	if err != nil {
		t.Fatal(err)
	}
	if readme.Name != "" || readme.Content != "" {
		t.Fatalf("expected empty readme, got %+v", readme)
	}

	testutil.Write(t, root, "README", "纯文本说明\n")
	readme, err = LoadProjectReadme(root)
	if err != nil {
		t.Fatal(err)
	}
	if readme.Name != "README" || readme.Content != "纯文本说明\n" {
		t.Fatalf("unexpected readme: %+v", readme)
	}

	// README.md 与 README 同时存在时优先 README.md，文件名大小写不影响识别。
	testutil.Write(t, root, "readme.md", "# 标题\n")
	readme, err = LoadProjectReadme(root)
	if err != nil {
		t.Fatal(err)
	}
	if readme.Name != "readme.md" || readme.Content != "# 标题\n" {
		t.Fatalf("expected readme.md to win, got %+v", readme)
	}
}

func TestGitGlobalConfigWorkflow(t *testing.T) {
	// 指向临时配置文件，避免改动开发机上的全局 Git 配置。
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))

	entries, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty global config, got %+v", entries)
	}

	if err := SetGlobalConfig("user.name", "设置测试"); err != nil {
		t.Fatal(err)
	}
	if err := SetGlobalConfig("core.quotepath", "false"); err != nil {
		t.Fatal(err)
	}
	entries, err = LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		values[entry.Key] = entry.Value
	}
	if values["user.name"] != "设置测试" || values["core.quotepath"] != "false" {
		t.Fatalf("unexpected global config: %+v", values)
	}

	// 空值删除配置项，删除不存在的配置项不应报错。
	if err := SetGlobalConfig("core.quotepath", ""); err != nil {
		t.Fatal(err)
	}
	if err := SetGlobalConfig("core.pager", ""); err != nil {
		t.Fatal(err)
	}
	entries, err = LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Key != "user.name" {
		t.Fatalf("unexpected global config after unset: %+v", entries)
	}
}
