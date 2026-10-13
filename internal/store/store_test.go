package store

import (
	"path/filepath"
	"testing"
	"time"
)

func ptrInt64(value int64) *int64 {
	return &value
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
