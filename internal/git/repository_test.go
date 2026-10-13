package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-power/internal/testutil"
)

// 未跟踪目录必须展开为具体文件，否则 status 只会给出 "dir/"，
// 前端展示不出文件名，git diff --no-index 也无法对其取 diff。
func TestLoadChangesExpandsUntrackedDirectory(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	testutil.Write(t, root, "src/view/newdir/inside.txt", "new\n")
	changes, err := loadChanges(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Path != "src/view/newdir/inside.txt" {
		t.Fatalf("expected expanded untracked file, got %+v", changes)
	}
}

func TestRepositoryWorkflow(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")

	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	testutil.Write(t, root, "first.txt", "two\n")
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

	repo, err = StageChange(repo.Path, "first.txt", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Changes) != 1 || !repo.Changes[0].Staged {
		t.Fatalf("expected one staged change, got %+v", repo.Changes)
	}

	repo, err = CommitChanges(repo.Path, "更新文件")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Commits) != 2 || repo.Commits[0].Subject != "更新文件" {
		t.Fatalf("unexpected commit history: %+v", repo.Commits)
	}

	repo, err = CreateBranch(repo.Path, "feature", true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentBranch != "feature" {
		t.Fatalf("expected feature branch, got %q", repo.CurrentBranch)
	}
}

// 侧边栏树需要展示 Tags 与 Stashes；annotated 标签必须解析为指向的提交哈希。
func TestLoadRepositoryTagsAndStashes(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	head := strings.TrimSpace(testutil.Git(t, root, "rev-parse", "--short", "HEAD"))
	testutil.Git(t, root, "tag", "v1.0")
	testutil.Git(t, root, "tag", "-a", "v2.0", "-m", "第二个版本")

	testutil.Write(t, root, "first.txt", "two\n")
	testutil.Git(t, root, "stash", "push", "-qm", "临时改动")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %+v", repo.Tags)
	}
	for _, tag := range repo.Tags {
		if tag.Hash != head {
			t.Errorf("tag %s expected hash %s, got %s", tag.Name, head, tag.Hash)
		}
	}
	if len(repo.Stashes) != 1 {
		t.Fatalf("expected 1 stash, got %+v", repo.Stashes)
	}
	// git 会把 stash 消息包装成 "On <分支>: <说明>"，与 git stash list 输出一致。
	if repo.Stashes[0].Ref != "stash@{0}" || !strings.Contains(repo.Stashes[0].Message, "临时改动") {
		t.Fatalf("unexpected stash: %+v", repo.Stashes[0])
	}
}

// refs/remotes/origin/HEAD 的 short name 是 "origin"，按 short name 过滤不到，
// 侧边栏 Remotes 树会多出一个与真实远程目录并列的假节点。
func TestLoadBranchesSkipsRemoteHead(t *testing.T) {
	remote := t.TempDir()
	testutil.Git(t, remote, "init", "-q", "-b", "main")
	testutil.Git(t, remote, "config", "user.name", "Test User")
	testutil.Git(t, remote, "config", "user.email", "test@example.com")
	testutil.Write(t, remote, "first.txt", "one\n")
	testutil.Git(t, remote, "add", "first.txt")
	testutil.Git(t, remote, "commit", "-qm", "初始提交")

	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")
	testutil.Git(t, root, "remote", "add", "origin", remote)
	testutil.Git(t, root, "fetch", "-q", "origin")
	testutil.Git(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	branches, err := loadBranches(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range branches {
		if branch.Name == "origin" {
			t.Fatalf("remote HEAD should be skipped, got %+v", branches)
		}
	}
	if len(branches) != 2 {
		t.Fatalf("expected main + origin/main, got %+v", branches)
	}
}

func TestCreateBranchWithoutCheckout(t *testing.T) {
	root := t.TempDir()
	testutil.Git(t, root, "init", "-q", "-b", "main")
	testutil.Git(t, root, "config", "user.name", "Test User")
	testutil.Git(t, root, "config", "user.email", "test@example.com")
	testutil.Write(t, root, "first.txt", "one\n")
	testutil.Git(t, root, "add", "first.txt")
	testutil.Git(t, root, "commit", "-qm", "初始提交")

	repo, err := CreateBranch(root, "feature", false)
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentBranch != "main" {
		t.Fatalf("expected to stay on main, got %q", repo.CurrentBranch)
	}
	found := false
	for _, branch := range repo.Branches {
		if branch.Name == "feature" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected feature branch to exist, got %+v", repo.Branches)
	}
}

func TestLoadRepositoryAheadCommits(t *testing.T) {
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
	testutil.Git(t, remote, "init", "-q", "--bare")
	testutil.Git(t, root, "remote", "add", "origin", remote)
	testutil.Git(t, root, "push", "-qu", "origin", "HEAD")

	testutil.Write(t, root, "first.txt", "two\n")
	testutil.Git(t, root, "commit", "-qam", "第二次提交")
	testutil.Write(t, root, "first.txt", "three\n")
	testutil.Git(t, root, "commit", "-qam", "第三次提交")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if repo.AheadCommits != 2 {
		t.Fatalf("expected 2 ahead commits, got %d", repo.AheadCommits)
	}

	pushed, err := GitAction(root, "push")
	if err != nil {
		t.Fatal(err)
	}
	if pushed.AheadCommits != 0 {
		t.Fatalf("expected 0 ahead commits after push, got %d", pushed.AheadCommits)
	}
}

func TestLoadRepositoryBehindCommits(t *testing.T) {
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
	testutil.Git(t, remote, "init", "-q", "--bare")
	testutil.Git(t, root, "remote", "add", "origin", remote)
	testutil.Git(t, root, "push", "-qu", "origin", "HEAD")

	// 另一个工作副本向远程推送一笔提交，本地 fetch 后应产生 1 笔未拉取提交。
	other := t.TempDir()
	testutil.Git(t, other, "init", "-q", "-b", "main")
	testutil.Git(t, other, "config", "user.name", "Test User")
	testutil.Git(t, other, "config", "user.email", "test@example.com")
	testutil.Git(t, other, "remote", "add", "origin", remote)
	testutil.Git(t, other, "fetch", "-q", "origin")
	testutil.Git(t, other, "checkout", "-q", "-B", "main", "origin/main")
	testutil.Write(t, other, "second.txt", "two\n")
	testutil.Git(t, other, "add", "second.txt")
	testutil.Git(t, other, "commit", "-qm", "远程提交")
	testutil.Git(t, other, "push", "-q", "origin", "main")
	testutil.Git(t, root, "fetch", "-q", "origin")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if repo.BehindCommits != 1 || repo.AheadCommits != 0 {
		t.Fatalf("expected 0 ahead and 1 behind commits, got ahead=%d behind=%d", repo.AheadCommits, repo.BehindCommits)
	}
}

func TestLoadBranchesSyncCounts(t *testing.T) {
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
	testutil.Git(t, remote, "init", "-q", "--bare")
	testutil.Git(t, root, "remote", "add", "origin", remote)
	testutil.Git(t, root, "push", "-qu", "origin", "HEAD")

	// feature 分支推送后再追加一笔提交，应计入该分支而非当前分支。
	testutil.Git(t, root, "switch", "-q", "-c", "feature")
	testutil.Git(t, root, "push", "-qu", "origin", "HEAD")
	testutil.Write(t, root, "feature.txt", "x\n")
	testutil.Git(t, root, "add", "feature.txt")
	testutil.Git(t, root, "commit", "-qm", "feature 提交")
	testutil.Git(t, root, "switch", "-q", "main")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range repo.Branches {
		if branch.Remote || branch.Name != "feature" {
			continue
		}
		if branch.Current {
			t.Fatal("expected feature branch to be inactive")
		}
		if branch.AheadCommits != 1 || branch.BehindCommits != 0 {
			t.Fatalf("expected feature branch ahead=1 behind=0, got ahead=%d behind=%d", branch.AheadCommits, branch.BehindCommits)
		}
		return
	}
	t.Fatalf("expected feature branch to exist, got %+v", repo.Branches)
}

func TestSwitchExistingRemoteTrackingBranch(t *testing.T) {
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
	testutil.Git(t, remote, "init", "-q", "--bare")
	testutil.Git(t, root, "remote", "add", "origin", remote)
	testutil.Git(t, root, "push", "-qu", "origin", "HEAD:main")
	testutil.Git(t, root, "branch", "feature")
	testutil.Git(t, root, "push", "-q", "origin", "feature")

	repo, err := LoadRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	repo, err = SwitchBranch(repo.Path, "origin/feature", true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.CurrentBranch != "feature" {
		t.Fatalf("expected feature branch, got %q", repo.CurrentBranch)
	}
}
