package main

import (
	"strings"
	"testing"

	"git-power/internal/config"
	"git-power/internal/git"
	"git-power/internal/testutil"
)

func TestGitCommandLogs(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	app := &App{}
	git.SetLogSink(app.appendGitLog)
	t.Cleanup(func() {
		git.SetLogSink(nil)
	})

	if _, err := git.LoadRepository(root); err != nil {
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

func TestSaveSettingsAppliesGitPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := &App{}
	settings := config.Default()
	settings.GitPath = "/tmp/custom-git"
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if git.Command() != "/tmp/custom-git" {
		t.Fatalf("expected configured git path, got %q", git.Command())
	}
	if got := app.GetSettings(); got.GitPath != "/tmp/custom-git" {
		t.Fatalf("unexpected stored settings: %+v", got)
	}
	settings.GitPath = ""
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if git.Command() != "git" {
		t.Fatalf("expected fallback git command, got %q", git.Command())
	}
}
