package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	changes := parseStatus(" M changed.go\x00R  new.go\x00old.go\x00?? new.txt\x00")
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}
	if changes[0].Staged || !changes[0].Unstaged {
		t.Errorf("unexpected worktree-only status: %+v", changes[0])
	}
	if changes[1].OriginalPath != "old.go" || !changes[1].Staged {
		t.Errorf("unexpected rename status: %+v", changes[1])
	}
	if changes[2].Staged || !changes[2].Unstaged {
		t.Errorf("unexpected untracked status: %+v", changes[2])
	}
}

func TestParseParents(t *testing.T) {
	if got := parseParents("abc123 def456"); len(got) != 2 || got[0] != "abc123" || got[1] != "def456" {
		t.Errorf("unexpected parents: %v", got)
	}
	if got := parseParents(""); len(got) != 0 {
		t.Errorf("expected empty parents, got %v", got)
	}
	if got := parseParents("   "); len(got) != 0 {
		t.Errorf("expected empty parents for blank input, got %v", got)
	}
}

func TestRepositoryWorkflow(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")

	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")

	write(t, root, "first.txt", "two\n")
	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Branches) != 1 || repo.Branches[0].Name != "main" {
		t.Fatalf("unexpected branches: %+v", repo.Branches)
	}
	if len(repo.Changes) != 1 || !repo.Changes[0].Unstaged {
		t.Fatalf("expected one unstaged change, got %+v", repo.Changes)
	}
	if repo.AheadCommits != 0 {
		t.Fatalf("expected 0 ahead commits without upstream, got %d", repo.AheadCommits)
	}

	app := &App{}
	repo, err = app.StageChange(repo.Path, "first.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Changes) != 1 || !repo.Changes[0].Staged {
		t.Fatalf("expected one staged change, got %+v", repo.Changes)
	}

	repo, err = app.Commit(repo.Path, "更新文件")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Commits) != 2 || repo.Commits[0].Subject != "更新文件" {
		t.Fatalf("unexpected commit history: %+v", repo.Commits)
	}

	repo, err = app.CreateBranch(repo.Path, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentBranch != "feature" {
		t.Fatalf("expected feature branch, got %q", repo.CurrentBranch)
	}
}

func TestLoadRepositoryAheadCommits(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")
	git(t, remote, "init", "-q", "--bare")
	git(t, root, "remote", "add", "origin", remote)
	git(t, root, "push", "-qu", "origin", "HEAD")

	write(t, root, "first.txt", "two\n")
	git(t, root, "commit", "-qam", "第二次提交")
	write(t, root, "first.txt", "three\n")
	git(t, root, "commit", "-qam", "第三次提交")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if repo.AheadCommits != 2 {
		t.Fatalf("expected 2 ahead commits, got %d", repo.AheadCommits)
	}

	pushed, err := (&App{}).GitAction(root, "push")
	if err != nil {
		t.Fatal(err)
	}
	if pushed.AheadCommits != 0 {
		t.Fatalf("expected 0 ahead commits after push, got %d", pushed.AheadCommits)
	}
}

func TestGetCommitDetail(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")

	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "提交主题")
	write(t, root, "first.txt", "two\n")
	write(t, root, "second.txt", "new\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "更新文件\n\n详细说明")

	detail, err := (&App{}).GetCommitDetail(root, git(t, root, "rev-parse", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	if detail.Subject != "更新文件" || detail.Body != "详细说明" {
		t.Fatalf("unexpected commit message: %+v", detail)
	}
	if len(detail.Files) != 2 {
		t.Fatalf("expected 2 files, got %+v", detail.Files)
	}
	statuses := map[string]string{}
	for _, file := range detail.Files {
		statuses[file.Path] = file.Status
	}
	if statuses["first.txt"] != "M" || statuses["second.txt"] != "A" {
		t.Fatalf("unexpected file statuses: %+v", detail.Files)
	}
}

func TestGetChangeDiff(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")

	write(t, root, "first.txt", "two\n")
	app := &App{}
	unstaged, err := app.GetChangeDiff(root, "first.txt", "", " M", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unstaged, "+two") {
		t.Fatalf("unexpected unstaged diff: %q", unstaged)
	}

	git(t, root, "add", "first.txt")
	staged, err := app.GetChangeDiff(root, "first.txt", "", "M ", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(staged, "+two") {
		t.Fatalf("unexpected staged diff: %q", staged)
	}
}

func TestParseCommitFilesRename(t *testing.T) {
	files := parseCommitFiles("R100\x00new.txt\x00old.txt\x00A\x00added.txt\x00")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %+v", files)
	}
	if files[0].Path != "new.txt" || files[0].OriginalPath != "old.txt" || files[0].Status != "R100" {
		t.Fatalf("unexpected rename: %+v", files[0])
	}
}

func TestGitCommandLogs(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")

	app := &App{}
	setGitCommandLogSink(app.appendGitLog)
	t.Cleanup(func() {
		setGitCommandLogSink(nil)
	})

	if _, err := LoadRepository(root); err != nil {
		t.Fatal(err)
	}

	logs := app.GetGitCommandLogs()
	if len(logs) == 0 {
		t.Fatal("expected Git command logs")
	}
	if !logs[0].Success || !strings.HasPrefix(logs[0].Command, "git -C ") {
		t.Fatalf("unexpected Git command log: %+v", logs[0])
	}
	if cleared := app.ClearGitCommandLogs(); len(cleared) != 0 {
		t.Fatalf("expected cleared Git command logs, got %+v", cleared)
	}
}

func TestSwitchExistingRemoteTrackingBranch(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")
	git(t, remote, "init", "-q", "--bare")
	git(t, root, "remote", "add", "origin", remote)
	git(t, root, "push", "-qu", "origin", "HEAD:main")
	git(t, root, "branch", "feature")
	git(t, root, "push", "-q", "origin", "feature")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{}
	repo, err = app.SwitchBranch(repo.Path, "origin/feature", true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentBranch != "feature" {
		t.Fatalf("expected feature branch, got %q", repo.CurrentBranch)
	}
}

func TestPull(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")
	git(t, remote, "init", "-q", "--bare", "-b", "main")
	git(t, root, "remote", "add", "origin", remote)
	git(t, root, "push", "-qu", "origin", "HEAD:main")

	clone := filepath.Join(t.TempDir(), "clone")
	parent := filepath.Dir(clone)
	git(t, parent, "clone", "-q", remote, clone)
	git(t, clone, "config", "user.name", "Test User")
	git(t, clone, "config", "user.email", "test@example.com")
	write(t, clone, "remote.txt", "from remote\n")
	git(t, clone, "add", "remote.txt")
	git(t, clone, "commit", "-qm", "远程新增文件")
	git(t, clone, "push", "-q", "origin", "HEAD:main")

	app := &App{}

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Remotes) != 1 || repo.Remotes[0] != "origin" {
		t.Fatalf("expected remotes [origin], got %v", repo.Remotes)
	}

	repo, err = app.Pull(repo.Path, "origin", "main", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "remote.txt")); err != nil {
		t.Fatalf("expected remote.txt after pull: %v", err)
	}

	// 远程再次更新，同时本地保留未提交更改，验证暂存并恢复。
	write(t, clone, "another.txt", "another\n")
	git(t, clone, "add", "another.txt")
	git(t, clone, "commit", "-qm", "远程再次新增")
	git(t, clone, "push", "-q", "origin", "HEAD:main")
	write(t, root, "local.txt", "local change\n")

	repo, err = app.Pull(root, "origin", "main", false, true)
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
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")
	git(t, remote, "init", "-q", "--bare", "-b", "main")
	git(t, root, "remote", "add", "origin", remote)

	app := &App{}

	// 首次推送新分支并建立跟踪引用。
	repo, err := app.Push(root, "origin", "main", "", true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	branch := repo.Branches[0]
	if !branch.Current || branch.Upstream != "origin/main" {
		t.Fatalf("expected main tracking origin/main, got %+v", branch)
	}

	// 推送到不同命名的远程分支。
	git(t, root, "checkout", "-qb", "feature")
	write(t, root, "feature.txt", "feature\n")
	git(t, root, "add", "feature.txt")
	git(t, root, "commit", "-qm", "功能提交")
	if _, err := app.Push(root, "origin", "feature", "feature", false, false, false); err != nil {
		t.Fatal(err)
	}
	list := git(t, remote, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if !strings.Contains(strings.ReplaceAll(list, "\r\n", "\n"), "feature") {
		t.Fatalf("expected remote branch feature, got %q", list)
	}

	// 推送全部标签。
	git(t, root, "tag", "v1.0")
	if _, err := app.Push(root, "origin", "feature", "", false, true, false); err != nil {
		t.Fatal(err)
	}
	tags := git(t, remote, "tag", "-l")
	if !strings.Contains(strings.ReplaceAll(tags, "\r\n", "\n"), "v1.0") {
		t.Fatalf("expected remote tag v1.0, got %q", tags)
	}
}

func TestDiscardChange(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	write(t, root, "sub/second.txt", "one\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "初始提交")

	app := &App{}

	// 已修改文件恢复到 HEAD。
	write(t, root, "first.txt", "two\n")
	if _, err := app.DiscardChange(root, "first.txt", "", " M"); err != nil {
		t.Fatal(err)
	}
	if content := strings.ReplaceAll(read(t, root, "first.txt"), "\r\n", "\n"); content != "one\n" {
		t.Fatalf("expected restored content, got %q", content)
	}

	// 已暂存的新增文件同时从暂存区和工作区移除。
	write(t, root, "added.txt", "new\n")
	git(t, root, "add", "added.txt")
	if _, err := app.DiscardChange(root, "added.txt", "", "A "); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "added.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected added.txt removed, got %v", err)
	}

	// 已暂存的删除文件被恢复。
	git(t, root, "rm", "-q", "sub/second.txt")
	if _, err := app.DiscardChange(root, "sub/second.txt", "", "D "); err != nil {
		t.Fatal(err)
	}
	if content := strings.ReplaceAll(read(t, root, "sub/second.txt"), "\r\n", "\n"); content != "one\n" {
		t.Fatalf("expected restored content, got %q", content)
	}

	// 未跟踪文件被删除。
	write(t, root, "untracked.txt", "new\n")
	if _, err := app.DiscardChange(root, "untracked.txt", "", "??"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "untracked.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected untracked.txt removed, got %v", err)
	}

	// 未跟踪目录被删除。
	write(t, root, "newdir/inside.txt", "new\n")
	if _, err := app.DiscardChange(root, "newdir/", "", "??"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "newdir")); !os.IsNotExist(err) {
		t.Fatalf("expected newdir removed, got %v", err)
	}

	// 越界路径被拒绝。
	if _, err := app.DiscardChange(root, "../outside.txt", "", " M"); err == nil {
		t.Fatal("expected error for escaping path")
	}
}

func TestIgnoreChange(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "Test User")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "first.txt", "one\n")
	git(t, root, "add", "first.txt")
	git(t, root, "commit", "-qm", "初始提交")
	write(t, root, "log.txt", "new\n")
	write(t, root, "cache/tmp.txt", "new\n")

	app := &App{}
	repo, err := app.IgnoreChange(root, "log.txt")
	if err != nil {
		t.Fatal(err)
	}
	ignored := map[string]bool{}
	for _, change := range repo.Changes {
		ignored[change.Path] = true
	}
	if ignored["log.txt"] || !ignored[".gitignore"] || !ignored["cache/"] {
		t.Fatalf("expected log.txt ignored and .gitignore change reported, got %+v", repo.Changes)
	}

	repo, err = app.IgnoreChange(root, "cache/")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Changes) != 1 || repo.Changes[0].Path != ".gitignore" {
		t.Fatalf("expected only .gitignore change remaining, got %+v", repo.Changes)
	}

	// 重复忽略不会写入重复行。
	if _, err := app.IgnoreChange(root, "log.txt"); err != nil {
		t.Fatal(err)
	}
	content := read(t, root, ".gitignore")
	if strings.Count(content, "/log.txt") != 1 {
		t.Fatalf("expected single ignore entry, got %q", content)
	}
}

func TestChangeAbsPath(t *testing.T) {
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	abs, err := changeAbsPath(root, "sub/dir/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(root, "sub", "dir", "file.txt")
	if abs != expected {
		t.Fatalf("expected %q, got %q", expected, abs)
	}
	if _, err := changeAbsPath(root, "../escape.txt"); err == nil {
		t.Fatal("expected error for escaping path")
	}
	if _, err := changeAbsPath(root, ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestProjectStore(t *testing.T) {
	store, err := NewProjectStore(filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.RecordProject(filepath.Join("D:", "work", "alpha"), "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordProject(filepath.Join("D:", "work", "beta"), ""); err != nil {
		t.Fatal(err)
	}
	// 重新打开 alpha 应刷新其最近打开时间。
	if err := store.RecordProject(filepath.Join("D:", "work", "alpha"), "alpha"); err != nil {
		t.Fatal(err)
	}

	recent, err := store.RecentProjects(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent projects, got %d", len(recent))
	}
	if recent[0].Path != filepath.Join("D:", "work", "alpha") {
		t.Fatalf("expected alpha most recent, got %q", recent[0].Path)
	}
	if recent[1].Name != "beta" {
		t.Fatalf("expected derived name beta, got %q", recent[1].Name)
	}
	if _, err := time.Parse(time.RFC3339Nano, recent[0].LastOpenedAt); err != nil {
		t.Fatalf("invalid lastOpenedAt %q: %v", recent[0].LastOpenedAt, err)
	}

	limited, err := store.RecentProjects(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 || limited[0].Path != recent[0].Path {
		t.Fatalf("expected only %q, got %+v", recent[0].Path, limited)
	}

	all, err := store.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Path >= all[1].Path {
		t.Fatalf("expected projects ordered by path, got %+v", all)
	}

	if err := store.RecordProject("", "x"); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestProjectFolderStore(t *testing.T) {
	store, err := NewProjectStore(filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	alpha := filepath.Join("D:", "work", "alpha")
	if err := store.RecordProject(alpha, "alpha"); err != nil {
		t.Fatal(err)
	}

	work, err := store.AddFolder("工作", nil)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := store.AddFolder("子项目", &work.Id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddFolder("  ", nil); err == nil {
		t.Fatal("expected error for empty folder name")
	}
	if _, err := store.AddFolder("孤儿", ptrInt64(9999)); err == nil {
		t.Fatal("expected error for missing parent folder")
	}

	if err := store.MoveProject(alpha, &sub.Id); err != nil {
		t.Fatal(err)
	}
	projects, err := store.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].FolderId == nil || *projects[0].FolderId != sub.Id {
		t.Fatalf("expected alpha in sub folder, got %+v", projects)
	}
	if err := store.MoveProject(filepath.Join("D:", "work", "missing"), nil); err == nil {
		t.Fatal("expected error for unrecorded project")
	}

	if err := store.RenameFolder(work.Id, "公司"); err != nil {
		t.Fatal(err)
	}
	if err := store.RenameFolder(work.Id, ""); err == nil {
		t.Fatal("expected error for empty folder name")
	}

	// 删除父目录后，子目录与仓库应上移到根级。
	if err := store.DeleteFolder(work.Id); err != nil {
		t.Fatal(err)
	}
	folders, err := store.AllFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].Id != sub.Id || folders[0].ParentId != nil {
		t.Fatalf("expected sub folder reparented to root, got %+v", folders)
	}
	projects, err = store.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].FolderId == nil || *projects[0].FolderId != sub.Id {
		t.Fatalf("expected alpha kept in sub folder, got %+v", projects)
	}
	if err := store.DeleteFolder(work.Id); err == nil {
		t.Fatal("expected error for missing folder")
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}

func git(t *testing.T, path string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	// 设置文件不存在时应返回默认设置。
	defaults, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.RecentLimit != 10 || defaults.DiffFontSize != 12 || !defaults.ConfirmDiscard {
		t.Fatalf("unexpected default settings: %+v", defaults)
	}

	saved := Settings{
		ScanDirectories: []string{"/tmp/workspace", "/tmp/other"},
		DiffFontFamily:  "JetBrains Mono",
		DiffFontSize:    14,
		RecentLimit:     20,
		ConfirmDiscard:  false,
		GitPath:         "/usr/local/bin/git",
		AgentBaseURL:    "https://api.example.com/v1",
		AgentAPIKey:     "sk-test",
		AgentModel:      "example-model",
	}
	if err := saveSettings(path, saved); err != nil {
		t.Fatal(err)
	}

	reloaded, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reloaded, saved) {
		t.Fatalf("unexpected reloaded settings: %+v", reloaded)
	}
}

func TestLoadSettingsKeepsDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"diffFontSize": 15, "confirmDiscard": false}`), 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := loadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.DiffFontSize != 15 || settings.ConfirmDiscard {
		t.Fatalf("unexpected settings: %+v", settings)
	}
	// 文件中未出现的字段应保留默认值。
	if settings.RecentLimit != 10 || len(settings.ScanDirectories) != 0 {
		t.Fatalf("expected defaults for missing fields, got %+v", settings)
	}
}

func TestParseGitConfig(t *testing.T) {
	entries := parseGitConfig("user.name\nT\x00core.editor\n\x00core.quotepath\x00")
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %+v", entries)
	}
	if entries[0].Key != "user.name" || entries[0].Value != "T" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Key != "core.editor" || entries[1].Value != "" {
		t.Errorf("unexpected empty-value entry: %+v", entries[1])
	}
	if entries[2].Key != "core.quotepath" {
		t.Errorf("unexpected valueless entry: %+v", entries[2])
	}
}

func TestGitGlobalConfigWorkflow(t *testing.T) {
	// 指向临时配置文件，避免改动开发机上的全局 Git 配置。
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))

	app := &App{}
	entries, err := app.GetGitGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty global config, got %+v", entries)
	}

	if err := app.SetGitGlobalConfig("user.name", "设置测试"); err != nil {
		t.Fatal(err)
	}
	if err := app.SetGitGlobalConfig("core.quotepath", "false"); err != nil {
		t.Fatal(err)
	}
	entries, err = app.GetGitGlobalConfig()
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
	if err := app.SetGitGlobalConfig("core.quotepath", ""); err != nil {
		t.Fatal(err)
	}
	if err := app.SetGitGlobalConfig("core.pager", ""); err != nil {
		t.Fatal(err)
	}
	entries, err = app.GetGitGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Key != "user.name" {
		t.Fatalf("unexpected global config after unset: %+v", entries)
	}
}

func TestSaveSettingsAppliesGitPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := &App{}
	settings := defaultSettings()
	settings.GitPath = "/tmp/custom-git"
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if gitCommand() != "/tmp/custom-git" {
		t.Fatalf("expected configured git path, got %q", gitCommand())
	}
	if got := app.GetSettings(); got.GitPath != "/tmp/custom-git" {
		t.Fatalf("unexpected stored settings: %+v", got)
	}
	settings.GitPath = ""
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if gitCommand() != "git" {
		t.Fatalf("expected fallback git command, got %q", gitCommand())
	}
}
