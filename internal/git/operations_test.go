package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-power/internal/testutil"
)

func TestPull(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")
	testutil.Git(t, remote, "init", "-q", "--bare", "-b", "main")
	testutil.Git(t, root, "remote", "add", "origin", remote)
	testutil.Git(t, root, "push", "-qu", "origin", "HEAD:main")

	clone := filepath.Join(t.TempDir(), "clone")
	parent := filepath.Dir(clone)
	testutil.Git(t, parent, "clone", "-q", remote, clone)
	testutil.Git(t, clone, "config", "user.name", "Test User")
	testutil.Git(t, clone, "config", "user.email", "test@example.com")
	testutil.Write(t, clone, "remote.txt", "from remote\n")
	testutil.Git(t, clone, "add", "remote.txt")
	testutil.Git(t, clone, "commit", "-qm", "远程新增文件")
	testutil.Git(t, clone, "push", "-q", "origin", "HEAD:main")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Remotes) != 1 || repo.Remotes[0] != "origin" {
		t.Fatalf("expected remotes [origin], got %v", repo.Remotes)
	}

	repo, err = Pull(repo.Path, "origin", "main", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "remote.txt")); err != nil {
		t.Fatalf("expected remote.txt after pull: %v", err)
	}

	// 远程再次更新，同时本地保留未提交更改，验证暂存并恢复。
	testutil.Write(t, clone, "another.txt", "another\n")
	testutil.Git(t, clone, "add", "another.txt")
	testutil.Git(t, clone, "commit", "-qm", "远程再次新增")
	testutil.Git(t, clone, "push", "-q", "origin", "HEAD:main")
	testutil.Write(t, root, "local.txt", "local change\n")

	repo, err = Pull(root, "origin", "main", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "another.txt")); err != nil {
		t.Fatalf("expected another.txt after pull: %v", err)
	}
	local, err := os.ReadFile(filepath.Join(root, "local.txt"))
	if err != nil || strings.TrimSpace(string(local)) != "local change" {
		t.Fatalf("expected local change preserved, got %q (err: %v)", string(local), err)
	}
}

func TestPush(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")
	testutil.Git(t, remote, "init", "-q", "--bare", "-b", "main")
	testutil.Git(t, root, "remote", "add", "origin", remote)

	// 首次推送新分支并建立跟踪引用。
	repo, err := Push(root, "origin", "main", "", true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	branch := repo.Branches[0]
	if !branch.Current || branch.Upstream != "origin/main" {
		t.Fatalf("expected main tracking origin/main, got %+v", branch)
	}

	// 推送到不同命名的远程分支。
	testutil.Git(t, root, "checkout", "-qb", "feature")
	testutil.Write(t, root, "feature.txt", "feature\n")
	testutil.Git(t, root, "add", "feature.txt")
	testutil.Git(t, root, "commit", "-qm", "功能提交")
	if _, err := Push(root, "origin", "feature", "feature", false, false, false); err != nil {
		t.Fatal(err)
	}
	list := testutil.Git(t, remote, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if !strings.Contains(strings.ReplaceAll(list, "\r\n", "\n"), "feature") {
		t.Fatalf("expected remote branch feature, got %q", list)
	}

	// 推送全部标签。
	testutil.Git(t, root, "tag", "v1.0")
	if _, err := Push(root, "origin", "feature", "", false, true, false); err != nil {
		t.Fatal(err)
	}
	tags := testutil.Git(t, remote, "tag", "-l")
	if !strings.Contains(strings.ReplaceAll(tags, "\r\n", "\n"), "v1.0") {
		t.Fatalf("expected remote tag v1.0, got %q", tags)
	}
}

func TestDiscardChange(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Write(t, root, "sub/second.txt", "one\n")
	testutil.Git(t, root, "add", ".")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	// 已修改文件恢复到 HEAD。
	testutil.Write(t, root, "first.txt", "two\n")
	if _, err := DiscardChange(root, "first.txt", "", " M"); err != nil {
		t.Fatal(err)
	}
	if content := strings.ReplaceAll(testutil.Read(t, root, "first.txt"), "\r\n", "\n"); content != "one\n" {
		t.Fatalf("expected restored content, got %q", content)
	}

	// 已暂存的新增文件同时从暂存区和工作区移除。
	testutil.Write(t, root, "added.txt", "new\n")
	testutil.Git(t, root, "add", "added.txt")
	if _, err := DiscardChange(root, "added.txt", "", "A "); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "added.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected added.txt removed, got %v", err)
	}

	// 已暂存的删除文件被恢复。
	testutil.Git(t, root, "rm", "-q", "sub/second.txt")
	if _, err := DiscardChange(root, "sub/second.txt", "", "D "); err != nil {
		t.Fatal(err)
	}
	if content := strings.ReplaceAll(testutil.Read(t, root, "sub/second.txt"), "\r\n", "\n"); content != "one\n" {
		t.Fatalf("expected restored content, got %q", content)
	}

	// 未跟踪文件被删除。
	testutil.Write(t, root, "untracked.txt", "new\n")
	if _, err := DiscardChange(root, "untracked.txt", "", "??"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "untracked.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected untracked.txt removed, got %v", err)
	}

	// 未跟踪目录被删除。
	testutil.Write(t, root, "newdir/inside.txt", "new\n")
	if _, err := DiscardChange(root, "newdir/", "", "??"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "newdir")); !os.IsNotExist(err) {
		t.Fatalf("expected newdir removed, got %v", err)
	}

	// 越界路径被拒绝。
	if _, err := DiscardChange(root, "../outside.txt", "", " M"); err == nil {
		t.Fatal("expected error for escaping path")
	}
}

func TestIgnoreChange(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")
	testutil.Write(t, root, "log.txt", "new\n")
	testutil.Write(t, root, "cache/tmp.txt", "new\n")

	repo, err := IgnoreChange(root, "log.txt")
	if err != nil {
		t.Fatal(err)
	}
	ignored := map[string]bool{}
	for _, change := range repo.Changes {
		ignored[change.Path] = true
	}
	if ignored["log.txt"] || !ignored[".gitignore"] || !ignored["cache/tmp.txt"] {
		t.Fatalf("expected log.txt ignored and .gitignore change reported, got %+v", repo.Changes)
	}

	repo, err = IgnoreChange(root, "cache/")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Changes) != 1 || repo.Changes[0].Path != ".gitignore" {
		t.Fatalf("expected only .gitignore change remaining, got %+v", repo.Changes)
	}

	// 重复忽略不会写入重复行。
	if _, err := IgnoreChange(root, "log.txt"); err != nil {
		t.Fatal(err)
	}
	content := testutil.Read(t, root, ".gitignore")
	if strings.Count(content, "/log.txt") != 1 {
		t.Fatalf("expected single ignore entry, got %q", content)
	}
}
