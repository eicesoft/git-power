package git

import (
	"path/filepath"
	"strings"
	"testing"

	"git-power/internal/testutil"
)

func TestLoadCommitDetail(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")

	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "提交主题")
	testutil.Write(t, root, "first.txt", "two\n")
	testutil.Write(t, root, "second.txt", "new\n")
	testutil.Git(t, root, "add", ".")
	testutil.Git(t, root, "commit", "-qm", "更新文件\n\n详细说明")

	detail, err := LoadCommitDetail(root, testutil.Git(t, root, "rev-parse", "HEAD"))
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

func TestDiff(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	testutil.Write(t, root, "first.txt", "two\n")
	unstaged, err := Diff(root, "first.txt", "", " M", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unstaged, "+two") {
		t.Fatalf("unexpected unstaged diff: %q", unstaged)
	}

	testutil.Git(t, root, "add", "first.txt")
	staged, err := Diff(root, "first.txt", "", "M ", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(staged, "+two") {
		t.Fatalf("unexpected staged diff: %q", staged)
	}
}

func TestAbsPath(t *testing.T) {
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	abs, err := AbsPath(root, "sub/dir/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(root, "sub", "dir", "file.txt")
	if abs != expected {
		t.Fatalf("expected %q, got %q", expected, abs)
	}
	if _, err := AbsPath(root, "../escape.txt"); err == nil {
		t.Fatal("expected error for escaping path")
	}
	if _, err := AbsPath(root, ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}
