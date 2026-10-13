package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	commandMu   sync.RWMutex
	commandPath string
)

// Command 返回设置页配置的 Git 可执行文件路径，空值表示使用 PATH 中的 git。
func Command() string {
	commandMu.RLock()
	defer commandMu.RUnlock()
	if commandPath == "" {
		return "git"
	}
	return commandPath
}

// SetCommandPath 设置 Git 可执行文件路径，空值表示回退到 PATH。
func SetCommandPath(path string) {
	commandMu.Lock()
	defer commandMu.Unlock()
	commandPath = strings.TrimSpace(path)
}

var (
	logMu   sync.RWMutex
	logSink func(GitCommandLog)
)

// SetLogSink 注册 Git 命令日志的接收方，传 nil 表示不再记录。
func SetLogSink(sink func(GitCommandLog)) {
	logMu.Lock()
	defer logMu.Unlock()
	logSink = sink
}

func runGit(path string, args ...string) (string, error) {
	return runGitCommand(append([]string{"-C", path}, args...)...)
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command(Command(), args...)
	startedAt := time.Now()
	output, err := cmd.CombinedOutput()
	recordGitCommand(args, output, err, err == nil, startedAt, time.Since(startedAt))
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

// runGitNoIndex 比较工作区文件与 /dev/null，用于生成未跟踪文件的完整内容差异。
// git diff --no-index 以退出码 1 表示“存在差异”，这属于正常结果。
func runGitNoIndex(path, file string) (string, error) {
	cmd := exec.Command(Command(), "-C", path, "diff", "--no-index", "--", "/dev/null", file)
	startedAt := time.Now()
	output, err := cmd.CombinedOutput()
	commandArgs := []string{"-C", path, "diff", "--no-index", "--", "/dev/null", file}
	success := err == nil || (cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1)
	recordErr := err
	if success {
		recordErr = nil
	}
	recordGitCommand(commandArgs, output, recordErr, success, startedAt, time.Since(startedAt))
	if err != nil && (cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 1) {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

func recordGitCommand(args []string, output []byte, commandErr error, success bool, startedAt time.Time, duration time.Duration) {
	logMu.RLock()
	sink := logSink
	logMu.RUnlock()
	if sink == nil {
		return
	}

	errorMessage := ""
	if commandErr != nil {
		errorMessage = strings.TrimSpace(formatGitOutput(output))
		if errorMessage == "" {
			errorMessage = commandErr.Error()
		}
	}

	sink(GitCommandLog{
		Command:   formatGitCommand(args),
		Output:    formatGitOutput(output),
		Error:     errorMessage,
		StartedAt: startedAt.Format(time.RFC3339Nano),
		Duration:  duration.Milliseconds(),
		Success:   success,
	})
}

func formatGitCommand(args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "git")
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\n\"") {
			parts = append(parts, strconv.Quote(arg))
			continue
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func formatGitOutput(output []byte) string {
	return strings.NewReplacer(
		"\x00", "\n",
		"\x1e", "\n",
		"\x1f", "\t",
	).Replace(string(output))
}
