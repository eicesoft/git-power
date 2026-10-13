package git

import "testing"

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

func TestParseCommitFilesRename(t *testing.T) {
	files := parseCommitFiles("R100\x00new.txt\x00old.txt\x00A\x00added.txt\x00")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %+v", files)
	}
	if files[0].Path != "new.txt" || files[0].OriginalPath != "old.txt" || files[0].Status != "R100" {
		t.Fatalf("unexpected rename: %+v", files[0])
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
