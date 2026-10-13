// Package git 封装 Git 仓库的读取与常用操作，是后端的 Git 领域层。
package git

// MaxCommandLogs 是内存中保留的 Git 命令日志条数上限。
const MaxCommandLogs = 200

// gitAction 标识前端请求的具体远程或临时保存操作。
type gitAction string

const (
	gitFetch gitAction = "fetch"
	gitPull  gitAction = "pull"
	gitPush  gitAction = "push"
	gitStash gitAction = "stash"
)

// GitCommandLog 记录一次 Git 命令的执行情况，供设置页的诊断面板展示。
type GitCommandLog struct {
	ID        int64  `json:"id"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	Error     string `json:"error"`
	StartedAt string `json:"startedAt"`
	Duration  int64  `json:"duration"`
	Success   bool   `json:"success"`
}

// GitConfigEntry 是全局 Git 配置中的一条键值。
type GitConfigEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Repository struct {
	Path          string   `json:"path"`
	Name          string   `json:"name"`
	CurrentBranch string   `json:"currentBranch"`
	AheadCommits  int      `json:"aheadCommits"`
	BehindCommits int      `json:"behindCommits"`
	Detached      bool     `json:"detached"`
	Remotes       []string `json:"remotes"`
	Branches      []Branch `json:"branches"`
	Tags          []Tag    `json:"tags"`
	Stashes       []Stash  `json:"stashes"`
	Commits       []Commit `json:"commits"`
	Changes       []Change `json:"changes"`
}

type Tag struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
	Hash string `json:"hash"`
}

type Stash struct {
	Ref     string `json:"ref"`
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

type Branch struct {
	Name          string `json:"name"`
	Ref           string `json:"ref"`
	Hash          string `json:"hash"`
	Upstream      string `json:"upstream"`
	Remote        bool   `json:"remote"`
	Current       bool   `json:"current"`
	AheadCommits  int    `json:"aheadCommits"`
	BehindCommits int    `json:"behindCommits"`
}

type Commit struct {
	Hash        string   `json:"hash"`
	ShortHash   string   `json:"shortHash"`
	Author      string   `json:"author"`
	Date        string   `json:"date"`
	Subject     string   `json:"subject"`
	Decorations string   `json:"decorations"`
	Parents     []string `json:"parents"`
}

type CommitFile struct {
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
	Status       string `json:"status"`
}

type CommitDetail struct {
	Hash      string       `json:"hash"`
	ShortHash string       `json:"shortHash"`
	Author    string       `json:"author"`
	Date      string       `json:"date"`
	Subject   string       `json:"subject"`
	Body      string       `json:"body"`
	Files     []CommitFile `json:"files"`
}

type Change struct {
	Path         string `json:"path"`
	OriginalPath string `json:"originalPath"`
	Status       string `json:"status"`
	Staged       bool   `json:"staged"`
	Unstaged     bool   `json:"unstaged"`
}

// ProjectSummary 是 Dashboard 单击项目时展示的仓库概要信息。
type ProjectSummary struct {
	Path           string   `json:"path"`
	Name           string   `json:"name"`
	CurrentBranch  string   `json:"currentBranch"`
	Detached       bool     `json:"detached"`
	Remotes        []string `json:"remotes"`
	AheadCommits   int      `json:"aheadCommits"`
	LocalBranches  int      `json:"localBranches"`
	RemoteBranches int      `json:"remoteBranches"`
	Commits        int      `json:"commits"`
	Changes        int      `json:"changes"`
	LastCommit     *Commit  `json:"lastCommit"`
}

// ProjectReadme 是仓库根目录 README 的内容，供 Dashboard 单击项目时渲染。
// 仓库没有 README 时 Name 与 Content 均为空。
type ProjectReadme struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// readmeRanks 决定仓库根目录下多个 README 同时存在时的优先级，数字越小越优先。
var readmeRanks = map[string]int{".md": 0, ".markdown": 1, "": 2, ".txt": 3}
