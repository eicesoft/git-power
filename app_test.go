package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

func TestRepositoryWorkflow(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q")
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
	git(t, root, "init", "-q")
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
	git(t, root, "init", "-q")
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
	git(t, root, "init", "-q")
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

func TestSwitchExistingRemoteTrackingBranch(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q")
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
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
