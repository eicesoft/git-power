package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-power/internal/store"
	"git-power/internal/testutil"
)

func TestSync(t *testing.T) {
	root := t.TempDir()
	// 第二个扫描根本身是仓库，应挂在根级而不建目录。
	selfRoot := t.TempDir()
	// 扫描目录本身是仓库时挂在根级，其余仓库按去除扫描目录前缀的路径归档。
	testutil.MkRepo(t, selfRoot)
	testutil.MkRepo(t, filepath.Join(root, "alpha"))
	testutil.MkRepo(t, filepath.Join(root, "alpha", "inner"))
	testutil.MkRepo(t, filepath.Join(root, "group", "beta"))
	testutil.MkRepo(t, filepath.Join(root, "plain", "deep", "gamma"))
	testutil.Write(t, root, "notes.txt", "ignore\n")

	projects, err := store.NewProjectStore(filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer projects.Close()

	result, err := Sync(projects, []string{root, selfRoot})
	if err != nil {
		t.Fatal(err)
	}
	if result.Found != 4 || result.Added != 4 || result.Updated != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	all, err := projects.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 projects, got %+v", all)
	}
	byPath := map[string]store.ProjectInfo{}
	for _, project := range all {
		byPath[project.Path] = project
	}
	// 仓库内部的子仓库不再下探。
	if _, ok := byPath[filepath.Join(root, "alpha", "inner")]; ok {
		t.Fatal("expected inner repository skipped")
	}

	folders, err := projects.AllFolders()
	if err != nil {
		t.Fatal(err)
	}
	names := map[int64]string{}
	parents := map[int64]*int64{}
	for _, folder := range folders {
		names[folder.Id] = folder.Name
		parents[folder.Id] = folder.ParentId
	}
	// 相对扫描目录的父级路径：alpha 在根级，group/beta 与 plain/deep/gamma 归入目录。
	got := map[string]bool{}
	for path, project := range byPath {
		segments := []string{}
		for id := project.FolderId; id != nil; id = parents[*id] {
			segments = append([]string{names[*id]}, segments...)
		}
		got[path+" -> "+strings.Join(segments, "/")] = true
	}
	want := []string{
		filepath.Join(root, "alpha") + " -> ",
		filepath.Join(root, "group", "beta") + " -> group",
		filepath.Join(root, "plain", "deep", "gamma") + " -> plain/deep",
		selfRoot + " -> ",
	}
	for _, key := range want {
		if !got[key] {
			t.Fatalf("missing %q, got %v", key, got)
		}
	}

	// 手工把 alpha 挪进 group，删除 beta 仓库，并登记一个扫描范围外的仓库。
	var groupID int64
	for _, folder := range folders {
		if folder.Name == "group" && folder.ParentId == nil {
			groupID = folder.Id
		}
	}
	if groupID == 0 {
		t.Fatalf("expected group folder, got %+v", folders)
	}
	if err := projects.MoveProject(filepath.Join(root, "alpha"), &groupID); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "group", "beta")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := projects.RecordProject(outside, "outside"); err != nil {
		t.Fatal(err)
	}

	// 再次同步：beta 已不在磁盘上，标记为缺失；手工归位与扫描范围外的仓库不受影响。
	again, err := Sync(projects, []string{root, selfRoot})
	if err != nil {
		t.Fatal(err)
	}
	if again.Found != 3 || again.Added != 0 || again.Updated != 3 {
		t.Fatalf("unexpected second result: %+v", again)
	}
	folders, err = projects.AllFolders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 3 {
		t.Fatalf("expected 3 folders after resync, got %+v", folders)
	}
	all, err = projects.AllProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 projects after resync, got %+v", all)
	}
	byPath = map[string]store.ProjectInfo{}
	for _, project := range all {
		byPath[project.Path] = project
	}
	if id := byPath[filepath.Join(root, "alpha")].FolderId; id == nil || *id != groupID {
		t.Fatalf("expected alpha kept in user folder, got %+v", byPath[filepath.Join(root, "alpha")])
	}
	if !byPath[filepath.Join(root, "group", "beta")].Missing {
		t.Fatal("expected removed repository marked missing")
	}
	if byPath[filepath.Join(root, "plain", "deep", "gamma")].Missing ||
		byPath[selfRoot].Missing ||
		byPath[outside].Missing {
		t.Fatalf("expected existing repositories not marked missing: %+v", byPath)
	}

	if _, err := Sync(projects, nil); err == nil {
		t.Fatal("expected error without scan directories")
	}
}
