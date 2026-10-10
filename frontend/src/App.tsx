import { Fragment, useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Menu } from '@base-ui/react/menu'
import {
  ChevronDown,
  Archive,
  Check,
  Cloud,
  FileText,
  Folder,
  FolderGit2,
  GitBranch,
  Home,
  History,
  List,
  Monitor,
  Download,
  Upload,
  Minus,
  Plus,
  RefreshCw,
  X,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Environment } from '../wailsjs/runtime/runtime'
import {
  Commit,
  CreateBranch,
  GetChangeDiff,
  GetCommitDetail,
  RefreshRepository,
  GitAction,
  SelectRepository,
  StageChange,
  SwitchBranch,
  UnstageChange,
} from '../wailsjs/go/main/App'

type Repository = Awaited<ReturnType<typeof SelectRepository>>
type Branch = Repository['branches'][number]
type CommitItem = Repository['commits'][number]
type CommitDetailItem = Awaited<ReturnType<typeof GetCommitDetail>>
type CommitFileItem = CommitDetailItem['files'][number]
type Change = Repository['changes'][number]

type ChangeListItem =
  | { type: 'directory'; directory: string }
  | { type: 'change'; directory: string; change: Change }

const windowDragStyle = { '--wails-draggable': 'drag' } as CSSProperties
const noDragStyle = { '--wails-draggable': 'no-drag' } as CSSProperties

const toolbarActions = [
  { action: 'fetch', label: 'Fetch', icon: Download },
  { action: 'pull', label: 'Pull', icon: Download },
  { action: 'push', label: 'Push', icon: Upload },
  { action: 'stash', label: 'Stash', icon: Archive },
] as const

const statusLabels: Record<string, string> = {
  M: '已修改',
  A: '新增',
  D: '删除',
  R: '重命名',
  C: '复制',
  U: '冲突',
  '?': '未跟踪',
}

const statusMarks: Record<string, { label: string; className: string }> = {
  M: { label: 'M', className: 'bg-yellow-500 text-white' },
  A: { label: '+', className: 'bg-green-500 text-white' },
  R: { label: 'M', className: 'bg-yellow-500 text-white' },
  C: { label: 'M', className: 'bg-yellow-500 text-white' },
  U: { label: '!', className: 'bg-red-500 text-white' },
  '?': { label: '+', className: 'bg-green-500 text-white' },
  D: { label: '-', className: 'bg-red-500 text-white' },
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function statusLabel(status: string) {
  return statusLabels[status.trim()[0]] ?? status
}

function statusMark(status: string) {
  return statusMarks[status.trim()[0]]
}

type DiffLine = {
  oldNumber: number | null
  newNumber: number | null
  kind: 'add' | 'remove' | 'context' | 'hunk' | 'meta'
  text: string
}

function parseDiff(diff: string): DiffLine[] {
  let oldNumber = 0
  let newNumber = 0
  const sourceLines = diff.split(/\r?\n/)
  if (sourceLines.at(-1) === '') sourceLines.pop()

  const visibleLines = sourceLines.filter(
    (line) =>
      !line.startsWith('diff ') &&
      !line.startsWith('index ') &&
      !line.startsWith('--- ') &&
      !line.startsWith('+++ '),
  )

  return visibleLines.map((line) => {
    const hunk = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line)
    if (hunk) {
      oldNumber = Number(hunk[1])
      newNumber = Number(hunk[2])
      return { oldNumber: null, newNumber: null, kind: 'hunk', text: line }
    }

    if (
      line.startsWith('new file') ||
      line.startsWith('deleted file') ||
      line.startsWith('rename ') ||
      line.startsWith('similarity ') ||
      line.startsWith('Binary ') ||
      line.startsWith('\\')
    ) {
      return { oldNumber: null, newNumber: null, kind: 'meta', text: line }
    }

    if (line.startsWith('+')) {
      return { oldNumber: null, newNumber: newNumber++, kind: 'add', text: line.slice(1) }
    }
    if (line.startsWith('-')) {
      return { oldNumber: oldNumber++, newNumber: null, kind: 'remove', text: line.slice(1) }
    }
    if (line.startsWith(' ')) {
      return { oldNumber: oldNumber++, newNumber: newNumber++, kind: 'context', text: line.slice(1) }
    }

    return { oldNumber: null, newNumber: null, kind: 'meta', text: line }
  })
}

function DiffViewer({ diff }: { diff: string }) {
  const lines = useMemo(() => parseDiff(diff), [diff])

  return (
    <div className="w-max min-w-full font-mono text-xs leading-5">
      {lines.map((line, index) => (
        <div
          key={index}
          className={`flex min-w-full ${
            line.kind === 'add'
              ? 'bg-green-500/15'
              : line.kind === 'remove'
                ? 'bg-red-500/15'
                : line.kind === 'hunk'
                  ? 'text-muted-foreground'
                  : line.kind === 'meta'
                    ? 'bg-muted/30 text-muted-foreground'
                    : ''
          }`}
        >
          <span className="w-12 shrink-0 select-none border-r px-2 text-right text-muted-foreground/60">
            {line.oldNumber ?? ''}
          </span>
          <span className="w-12 shrink-0 select-none border-r px-2 text-right text-muted-foreground/60">
            {line.newNumber ?? ''}
          </span>
          <span
            className={`w-6 shrink-0 select-none text-center ${
              line.kind === 'add' ? 'text-green-500' : line.kind === 'remove' ? 'text-red-500' : ''
            }`}
          >
            {line.kind === 'add' ? '+' : line.kind === 'remove' ? '-' : ''}
          </span>
          <code className="whitespace-pre pr-4">{line.text}</code>
        </div>
      ))}
    </div>
  )
}

function App() {
  const [repositories, setRepositories] = useState<Repository[]>([])
  const [activePath, setActivePath] = useState<string | null>(null)
  const [workspaceView, setWorkspaceView] = useState<'local' | 'commits'>('local')
  const [selectedChangeKey, setSelectedChangeKey] = useState('')
  const [selectedCommitHash, setSelectedCommitHash] = useState('')
  const [changeDiff, setChangeDiff] = useState('')
  const [commitDetail, setCommitDetail] = useState<CommitDetailItem | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailTab, setDetailTab] = useState('info')
  const [commitSubjects, setCommitSubjects] = useState<Record<string, string>>({})
  const [commitDescriptions, setCommitDescriptions] = useState<Record<string, string>>({})
  const [branchName, setBranchName] = useState('')
  const [creatingBranch, setCreatingBranch] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [platform, setPlatform] = useState('')

  const activeRepository = useMemo(
    () => repositories.find((repository) => repository.path === activePath) ?? null,
    [activePath, repositories],
  )

  useEffect(() => {
    setCreatingBranch(false)
    setBranchName('')
    setSelectedChangeKey('')
    setSelectedCommitHash('')
    setChangeDiff('')
    setCommitDetail(null)
    setDetailTab('info')
  }, [activePath])

  useEffect(() => {
    Environment()
      .then((environment) => setPlatform(environment.platform))
      .catch(() => setPlatform(''))
  }, [])

  const updateRepository = (repository: Repository) => {
    if (!repository.path) return
    setRepositories((current) => {
      const exists = current.some((item) => item.path === repository.path)
      return exists
        ? current.map((item) => (item.path === repository.path ? repository : item))
        : [...current, repository]
    })
    setActivePath(repository.path)
  }

  const runAction = async (action: () => Promise<Repository>) => {
    setBusy(true)
    setError('')
    try {
      updateRepository(await action())
    } catch (actionError) {
      setError(errorMessage(actionError))
    } finally {
      setBusy(false)
    }
  }

  const openRepository = async () => {
    setBusy(true)
    setError('')
    try {
      const repository = await SelectRepository()
      if (repository.path) updateRepository(repository)
    } catch (actionError) {
      setError(errorMessage(actionError))
    } finally {
      setBusy(false)
    }
  }

  const closeRepository = (path: string) => {
    const remaining = repositories.filter((repository) => repository.path !== path)
    setRepositories(remaining)
    if (activePath === path) setActivePath(remaining.at(-1)?.path ?? null)
  }

  const refreshRepository = () => {
    if (!activePath) return
    runAction(() => RefreshRepository(activePath))
  }

  const switchBranch = (branch: Branch) => {
    if (!activePath || branch.current || busy) return
    runAction(() => SwitchBranch(activePath, branch.name, branch.remote))
  }

  const runGitAction = (action: string) => {
    if (!activePath || busy) return
    runAction(() => GitAction(activePath, action))
  }

  const createBranch = () => {
    if (!activePath || !branchName.trim() || busy) return
    const path = activePath
    const name = branchName.trim()
    runAction(async () => {
      const repository = await CreateBranch(path, name)
      setCreatingBranch(false)
      setBranchName('')
      return repository
    })
  }

  const stageChange = (change: Change) => {
    if (!activePath || busy) return
    runAction(() => StageChange(activePath, change.path, change.originalPath))
  }

  const stageDirectory = (directory: string) => {
    if (!activePath || busy || !directory) return
    runAction(() => StageChange(activePath, `${directory}/`, ''))
  }

  const unstageChange = (change: Change) => {
    if (!activePath || busy) return
    runAction(() => UnstageChange(activePath, change.path, change.originalPath))
  }

  const unstageDirectory = (directory: string) => {
    if (!activePath || busy || !directory) return
    runAction(() => UnstageChange(activePath, `${directory}/`, ''))
  }

  const commitChanges = () => {
    if (!activePath || busy) return
    const path = activePath
    const subject = (commitSubjects[path] ?? '').trim()
    const description = (commitDescriptions[path] ?? '').trim()
    if (!subject) {
      setError('提交主题不能为空')
      return
    }
    const message = [subject, description].filter(Boolean).join('\n\n')
    runAction(async () => {
      const repository = await Commit(path, message)
      setCommitSubjects((drafts) => ({ ...drafts, [path]: '' }))
      setCommitDescriptions((drafts) => ({ ...drafts, [path]: '' }))
      return repository
    })
  }

  const localBranches = activeRepository?.branches.filter((branch) => !branch.remote) ?? []
  const remoteBranches = activeRepository?.branches.filter((branch) => branch.remote) ?? []
  const aheadCommits = activeRepository?.aheadCommits ?? 0
  const currentBranchHasUpstream = localBranches.some(
    (branch) => branch.current && branch.upstream,
  )
  const stagedChanges = activeRepository?.changes.filter((change) => change.staged) ?? []
  const unstagedChanges = activeRepository?.changes.filter((change) => change.unstaged) ?? []
  const changeEntries = activeRepository
    ? [
        ...unstagedChanges.map((change) => ({ change, staged: false, key: `unstaged:${change.path}` })),
        ...stagedChanges.map((change) => ({ change, staged: true, key: `staged:${change.path}` })),
      ]
    : []
  const selectedChange = changeEntries.find((entry) => entry.key === selectedChangeKey) ?? changeEntries[0] ?? null
  const selectedCommit = activeRepository?.commits.find((commit) => commit.hash === selectedCommitHash)
    ?? activeRepository?.commits[0]
    ?? null

  useEffect(() => {
    if (!activePath || workspaceView !== 'local' || !selectedChange) {
      setChangeDiff('')
      setDetailLoading(false)
      return
    }
    let cancelled = false
    setDetailLoading(true)
    GetChangeDiff(
      activePath,
      selectedChange.change.path,
      selectedChange.change.originalPath,
      selectedChange.change.status,
      selectedChange.staged,
    )
      .then((diff) => {
        if (!cancelled) setChangeDiff(diff)
      })
      .catch((diffError) => {
        if (!cancelled) setError(errorMessage(diffError))
      })
      .finally(() => {
        if (!cancelled) setDetailLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [
    activePath,
    workspaceView,
    selectedChange?.change.path,
    selectedChange?.change.originalPath,
    selectedChange?.change.status,
    selectedChange?.staged,
  ])

  useEffect(() => {
    if (!activePath || workspaceView !== 'commits' || !selectedCommit) {
      setCommitDetail(null)
      setDetailLoading(false)
      return
    }
    let cancelled = false
    setDetailLoading(true)
    GetCommitDetail(activePath, selectedCommit.hash)
      .then((detail) => {
        if (!cancelled) setCommitDetail(detail)
      })
      .catch((detailError) => {
        if (!cancelled) setError(errorMessage(detailError))
      })
      .finally(() => {
        if (!cancelled) setDetailLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activePath, workspaceView, selectedCommit?.hash])

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background text-foreground">
      {/* 按系统预留窗口控制按钮：macOS 左侧红绿灯，Windows 右侧最小化/最大化/关闭。 */}
      <header
        className={`flex h-11 shrink-0 items-end gap-1 border-b bg-muted/40 ${
          platform === 'darwin'
            ? 'pl-28 pr-2'
            : platform === 'windows'
              ? 'pl-2 pr-48'
              : 'pl-28 pr-48'
        }`}
        style={windowDragStyle}
      >
        <Tabs
          value={activePath ?? 'home'}
          onValueChange={(value) => setActivePath(value === 'home' ? null : String(value))}
          className="min-w-0 self-center"
        >
          <TabsList className="min-w-0 justify-start overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            <TabsTrigger
              value="home"
              className={activePath === null ? 'w-32 shrink-0 justify-start' : 'w-9 shrink-0 justify-center'}
              style={noDragStyle}
            >
              <Home className="size-4 shrink-0 text-primary" />
              {activePath === null && <span className="truncate">Dashboard</span>}
            </TabsTrigger>
            {repositories.map((repository) => (
              <div
                key={repository.path}
                className="relative flex h-full w-48 shrink-0 items-center"
              >
                <TabsTrigger
                  value={repository.path}
                  className="min-w-0 flex-1 justify-start pr-7"
                  title={repository.path}
                  style={noDragStyle}
                >
                  <FolderGit2 className="size-4 shrink-0 text-primary" />
                  <span className="truncate">{repository.name}</span>
                </TabsTrigger>
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute right-1 top-1/2 size-6 -translate-y-1/2 rounded p-1"
                  onClick={() => closeRepository(repository.path)}
                  title="关闭仓库"
                  style={noDragStyle}
                >
                  <X className="size-3.5" />
                </Button>
              </div>
            ))}
          </TabsList>
        </Tabs>

      </header>

      {error && (
        <div
          role="alert"
          className="flex shrink-0 items-center justify-between border-b bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          <span className="truncate">{error}</span>
          <Button variant="ghost" size="icon" className="size-7" onClick={() => setError('')} title="关闭错误">
            <X />
          </Button>
        </div>
      )}

      {!activeRepository ? (
        <OpenRepositoryPage busy={busy} onOpen={openRepository} />
      ) : (
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1">
            <aside className="flex w-64 shrink-0 flex-col border-r">
            <Tabs
              value={workspaceView}
              onValueChange={(value) => setWorkspaceView(value as 'local' | 'commits')}
              className="shrink-0 border-b p-1"
            >
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="local" className="gap-1.5">
                  <FileText className="size-4" />
                  Local Changes
                </TabsTrigger>
                <TabsTrigger value="commits" className="gap-1.5">
                  <History className="size-4" />
                  Commits
                </TabsTrigger>
              </TabsList>
            </Tabs>

            <div className="flex h-11 shrink-0 items-center justify-between border-b px-3">
              <div className="flex items-center gap-2">
                <GitBranch className="size-4 text-primary" />
                <span className="text-sm font-semibold">分支</span>
                <Badge variant="secondary">{localBranches.length + remoteBranches.length}</Badge>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  onClick={() => setCreatingBranch((value) => !value)}
                  disabled={busy}
                  title="新建分支"
                >
                  <Plus />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  onClick={refreshRepository}
                  disabled={busy}
                  title="刷新仓库"
                >
                  <RefreshCw className={busy ? 'animate-spin' : undefined} />
                </Button>
              </div>
            </div>

            {creatingBranch && (
              <div className="flex shrink-0 gap-1 border-b p-2">
                <Input
                  autoFocus
                  value={branchName}
                  onChange={(event) => setBranchName(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') createBranch()
                    if (event.key === 'Escape') setCreatingBranch(false)
                  }}
                  placeholder="新分支名称"
                />
                <Button size="icon" className="size-9" onClick={createBranch} disabled={!branchName.trim() || busy} title="创建分支">
                  <Check />
                </Button>
              </div>
            )}

            <ScrollArea className="min-h-0 flex-1">
              <div className="space-y-4 p-2">
                <BranchGroup
                  title="本地分支"
                  icon={<GitBranch className="size-3.5" />}
                  branches={localBranches}
                  busy={busy}
                  aheadCommits={aheadCommits}
                  onSelect={switchBranch}
                />
                <BranchGroup
                  title="远程分支"
                  icon={<Cloud className="size-3.5" />}
                  branches={remoteBranches}
                  busy={busy}
                  aheadCommits={0}
                  onSelect={switchBranch}
                />
              </div>
            </ScrollArea>
          </aside>

          <main className="flex min-w-0 flex-1 flex-col bg-muted/30">
          {activeRepository && (
            <div className="flex h-[56px] shrink-0 items-center justify-center gap-10 border-b px-2 py-1" style={noDragStyle}>
            <div className="flex items-center rounded-md">
              {toolbarActions.map((item, index) => (
                <Fragment key={item.action}>
                  {index > 0 && (
                    <Separator
                      orientation="vertical"
                      className="mx-0.5 h-5"
                      style={{ alignSelf: 'center' }}
                    />
                  )}
                  <Button
                    variant="ghost"
                    className={`h-11 ${
                      item.action === 'push' && currentBranchHasUpstream ? 'w-16' : 'w-10'
                    } flex-col gap-0.5 rounded-md px-1 text-[10px] text-muted-foreground hover:bg-transparent hover:text-foreground dark:hover:bg-transparent`}
                    onClick={() => runGitAction(item.action)}
                    disabled={busy}
                    title={
                      item.action === 'push' && currentBranchHasUpstream
                        ? `${item.label}（${aheadCommits} 个未推送提交）`
                        : item.label
                    }
                  >
                    <span className="relative flex h-7 w-7 items-center justify-center rounded-md p-1 group-hover/button:bg-muted">
                      <item.icon className="size-5" />
                      {item.action === 'push' && currentBranchHasUpstream && (
                        <Badge className="absolute -right-2.5 top-1 h-3.5 min-w-3.5 rounded-full px-1 text-[9px] leading-3">
                          {aheadCommits}
                        </Badge>
                      )}
                    </span>
                    <span>
                      {item.label}
                    </span>
                  </Button>
                </Fragment>
              ))}
            </div>

            <div
              className="mx-2 flex h-9 min-w-64 max-w-xs flex-1 items-center justify-center gap-1 rounded-md border bg-background px-2"
              title={activeRepository.path}
            >
              <Menu.Root>
                <Menu.Trigger
                  render={(props) => (
                    <Button
                      {...props}
                      variant="ghost"
                      size="icon"
                      className="size-6"
                      title="选择分支"
                    >
                      <List className="size-3.5" />
                    </Button>
                  )}
                />
                <Menu.Portal>
                  <Menu.Positioner align="start">
                    <Menu.Popup className="max-h-72 min-w-48 overflow-auto rounded-lg border bg-popover p-1 shadow-lg">
                      {localBranches.map((branch) => (
                        <Menu.Item
                          key={branch.name}
                          className="cursor-pointer rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted"
                          onSelect={() => switchBranch(branch)}
                        >
                          {branch.name}
                        </Menu.Item>
                      ))}
                    </Menu.Popup>
                  </Menu.Positioner>
                </Menu.Portal>
              </Menu.Root>
              <div className="flex min-w-0 flex-1 items-center justify-center gap-1 text-sm">
                <span className="truncate font-medium">{activeRepository.name}</span>
                <ChevronDown className="size-3 shrink-0 text-muted-foreground" />
                <span className="truncate text-muted-foreground">{activeRepository.currentBranch || 'main'}</span>
              </div>
            </div>

            <Menu.Root>
              <Menu.Trigger
                render={(props) => (
                  <Button
                    {...props}
                    variant="ghost"
                    className="h-9 flex-col gap-0.5 px-2 text-[10px] text-muted-foreground hover:text-foreground"
                    title="Open In"
                  >
                    <Monitor className="size-3.5" />
                    <span>Open In</span>
                  </Button>
                )}
              />
              <Menu.Portal>
                <Menu.Positioner align="end">
                  <Menu.Popup className="min-w-40 rounded-lg border bg-popover p-1 shadow-lg">
                    <Menu.Item
                      className="cursor-pointer rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted"
                      onSelect={() => window.open(`file://${activeRepository.path}`)}
                    >
                      Finder
                    </Menu.Item>
                  </Menu.Popup>
                </Menu.Positioner>
              </Menu.Portal>
            </Menu.Root>
          </div>
        )}
            {workspaceView === 'local' ? (
              <div className="flex min-h-0 flex-1">
                <section className="flex w-[42%] min-w-64 max-w-[520px] shrink-0 flex-col border-r">
                  <div className="flex h-11 shrink-0 items-center justify-between border-b px-4">
                    <div className="flex items-center gap-2">
                      <FileText className="size-4 text-primary" />
                      <span className="text-sm font-semibold">文件更改</span>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {unstagedChanges.length} 未暂存 · {stagedChanges.length} 已暂存
                    </span>
                  </div>
                  {activeRepository.changes.length === 0 ? (
                    <div className="p-6 text-center text-sm text-muted-foreground">工作区干净</div>
                  ) : (
                    <div className="flex min-h-0 flex-1 flex-col">
                      <div className="flex min-h-0 flex-[4_1_0%]">
                        <ChangeGroup
                          title="Unstaged"
                          changes={unstagedChanges}
                          staged={false}
                          selectedKey={selectedChange?.key ?? ''}
                          busy={busy}
                          onSelect={(change) => setSelectedChangeKey(`unstaged:${change.path}`)}
                          onStage={stageChange}
                          onStageDirectory={stageDirectory}
                          onUnstage={unstageChange}
                          onUnstageDirectory={unstageDirectory}
                        />
                      </div>
                      <div className="flex min-h-0 flex-[3_1_0%] border-t">
                        <ChangeGroup
                          title="Staged"
                          changes={stagedChanges}
                          staged
                          selectedKey={selectedChange?.key ?? ''}
                          busy={busy}
                          onSelect={(change) => setSelectedChangeKey(`staged:${change.path}`)}
                          onStage={stageChange}
                          onStageDirectory={stageDirectory}
                          onUnstage={unstageChange}
                          onUnstageDirectory={unstageDirectory}
                        />
                      </div>
                    </div>
                  )}
                </section>

                <section className="flex min-w-0 flex-1 flex-col">
                  <div className="flex h-11 shrink-0 items-center justify-between border-b px-4">
                    <div className="flex min-w-0 items-center gap-2">
                      <FileText className="size-4 shrink-0 text-primary" />
                      <span className="truncate text-sm font-semibold">
                        {selectedChange ? selectedChange.change.path : '文件更改'}
                      </span>
                    </div>
                    {selectedChange && (
                      <Badge variant={selectedChange.staged ? 'secondary' : 'outline'}>
                        {selectedChange.staged ? 'Staged' : 'Unstaged'}
                      </Badge>
                    )}
                  </div>
                  <ScrollArea className="min-h-0 flex-[1.2]">
                    {!selectedChange ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">选择文件查看更改</div>
                    ) : detailLoading ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">读取更改...</div>
                    ) : changeDiff ? (
                      <DiffViewer diff={changeDiff} />
                    ) : (
                      <div className="p-6 text-center text-sm text-muted-foreground">无文本更改</div>
                    )}
                  </ScrollArea>

                  <div className="flex min-h-0 flex-1 flex-col gap-3 border-t p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Check className="size-4 text-primary" />
                        <span className="text-sm font-medium">Commit</span>
                      </div>
                      <span className="text-xs text-muted-foreground">{stagedChanges.length} 个暂存文件</span>
                    </div>
                    <label className="space-y-1.5">
                      <span className="text-xs font-medium text-muted-foreground">Commit Subject</span>
                      <Input
                        value={commitSubjects[activeRepository.path] ?? ''}
                        onChange={(event) =>
                          setCommitSubjects((drafts) => ({
                            ...drafts,
                            [activeRepository.path]: event.target.value,
                          }))
                        }
                        placeholder="简短描述本次提交"
                        disabled={busy}
                      />
                    </label>
                    <label className="flex min-h-0 flex-1 flex-col space-y-1.5">
                      <span className="text-xs font-medium text-muted-foreground">Description</span>
                      <textarea
                        className="min-h-20 flex-1 resize-none rounded-md border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                        value={commitDescriptions[activeRepository.path] ?? ''}
                        onChange={(event) =>
                          setCommitDescriptions((drafts) => ({
                            ...drafts,
                            [activeRepository.path]: event.target.value,
                          }))
                        }
                        placeholder="可选的详细说明"
                        disabled={busy}
                      />
                    </label>
                    <div className="flex items-center justify-between gap-3">
                      <span className="truncate text-xs text-muted-foreground">
                        {stagedChanges.length === 0 ? '暂存文件后可提交' : '提交将包含暂存区内容'}
                      </span>
                      <Button
                        onClick={commitChanges}
                        disabled={busy || stagedChanges.length === 0 || !(commitSubjects[activeRepository.path] ?? '').trim()}
                      >
                        <Check />
                        提交
                      </Button>
                    </div>
                  </div>
                </section>
              </div>
            ) : (
              <div className="flex min-h-0 flex-1">
                <section className="flex w-[57.142857%] shrink-0 flex-col border-r">
                  <div className="flex h-11 shrink-0 items-center justify-between border-b px-4">
                    <div className="flex items-center gap-2">
                      <History className="size-4 text-primary" />
                      <span className="text-sm font-semibold">提交历史</span>
                      <Badge variant="outline">
                        {activeRepository.detached ? '分离 HEAD' : activeRepository.currentBranch || '未初始化分支'}
                      </Badge>
                    </div>
                    <span className="text-xs text-muted-foreground">{activeRepository.commits.length} 条</span>
                  </div>
                  <ScrollArea className="min-h-0 flex-1">
                    {activeRepository.commits.length === 0 ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">暂无提交</div>
                    ) : (
                      <div>
                        {activeRepository.commits.map((commit, index) => (
                          <CommitRow
                            key={commit.hash}
                            commit={commit}
                            selected={commit.hash === selectedCommit?.hash}
                            showDecorations={index > 0}
                            onSelect={() => setSelectedCommitHash(commit.hash)}
                          />
                        ))}
                      </div>
                    )}
                  </ScrollArea>
                </section>

                <Tabs
                  value={detailTab}
                  onValueChange={(value) => setDetailTab(String(value))}
                  className="min-h-0 min-w-0 flex-1"
                >
                  <div className="flex h-11 shrink-0 items-center justify-between border-b px-4">
                    <TabsList variant="line">
                      <TabsTrigger value="info">提交信息</TabsTrigger>
                      <TabsTrigger value="files">更改文件</TabsTrigger>
                    </TabsList>
                    <span className="font-mono text-xs text-muted-foreground">
                      {commitDetail?.shortHash ?? selectedCommit?.shortHash ?? ''}
                    </span>
                  </div>

                  <TabsContent value="info" className="min-h-0 flex-1 overflow-hidden">
                    <ScrollArea className="h-full">
                      {!commitDetail ? (
                        <div className="p-6 text-center text-sm text-muted-foreground">
                          {detailLoading ? '读取提交信息...' : '暂无提交信息'}
                        </div>
                      ) : (
                        <div className="max-w-3xl space-y-5 p-6">
                          <div className="space-y-1">
                            <div className="text-xs font-medium uppercase text-muted-foreground">Subject</div>
                            <div className="text-base font-semibold">{commitDetail.subject}</div>
                          </div>
                          <div className="space-y-1">
                            <div className="text-xs font-medium uppercase text-muted-foreground">Description</div>
                            <div className="whitespace-pre-wrap text-sm leading-6">
                              {commitDetail.body || '无详细说明'}
                            </div>
                          </div>
                          <div className="grid gap-4 text-sm sm:grid-cols-2">
                            <div className="space-y-1">
                              <div className="text-xs font-medium uppercase text-muted-foreground">Author</div>
                              <div>{commitDetail.author}</div>
                            </div>
                            <div className="space-y-1">
                              <div className="text-xs font-medium uppercase text-muted-foreground">Date</div>
                              <div>{formatDate(commitDetail.date)}</div>
                            </div>
                            <div className="space-y-1 sm:col-span-2">
                              <div className="text-xs font-medium uppercase text-muted-foreground">Hash</div>
                              <div className="break-all font-mono text-xs">{commitDetail.hash}</div>
                            </div>
                          </div>
                        </div>
                      )}
                    </ScrollArea>
                  </TabsContent>

                  <TabsContent value="files" className="min-h-0 flex-1 overflow-hidden">
                    <ScrollArea className="h-full">
                      {!commitDetail ? (
                        <div className="p-6 text-center text-sm text-muted-foreground">
                          {detailLoading ? '读取更改文件...' : '暂无更改文件'}
                        </div>
                      ) : commitDetail.files.length === 0 ? (
                        <div className="p-6 text-center text-sm text-muted-foreground">无更改文件</div>
                      ) : (
                        <div className="divide-y">
                          {commitDetail.files.map((file) => (
                            <CommitFileRow key={`${file.status}:${file.path}`} file={file} />
                          ))}
                        </div>
                      )}
                    </ScrollArea>
                  </TabsContent>
                </Tabs>
              </div>
            )}
          </main>
          </div>
        </div>
      )}
    </div>
  )
}

function OpenRepositoryPage({ busy, onOpen }: { busy: boolean; onOpen: () => void }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 text-muted-foreground">
      <FolderGit2 className="size-10 text-primary" />
      <div className="text-center">
        <p className="font-medium text-foreground">打开一个 Git 仓库</p>
        <p className="mt-1 text-sm">选择本机上的 Git 仓库目录</p>
      </div>
      <Button onClick={onOpen} disabled={busy}>
        <FolderGit2 />
        打开仓库
      </Button>
    </div>
  )
}

function BranchGroup({
  title,
  icon,
  branches,
  busy,
  aheadCommits,
  onSelect,
}: {
  title: string
  icon: ReactNode
  branches: Branch[]
  busy: boolean
  aheadCommits: number
  onSelect: (branch: Branch) => void
}) {
  return (
    <div>
      <div className="mb-1 flex items-center gap-1.5 px-2 text-xs font-medium uppercase text-muted-foreground">
        {icon}
        {title}
      </div>
      <div className="space-y-0.5">
        {branches.length === 0 && <div className="px-2 py-1 text-xs text-muted-foreground">无</div>}
        {branches.map((branch) => (
          <button
            key={branch.ref}
            type="button"
            disabled={busy || branch.current}
            onDoubleClick={() => onSelect(branch)}
            className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors ${
              branch.current
                ? 'bg-primary/10 font-medium text-primary'
                : 'text-foreground hover:bg-muted'
            }`}
            title={branch.upstream ? `上游: ${branch.upstream}` : branch.ref}
          >
            <GitBranch className="size-3.5 shrink-0" />
            <span className="min-w-0 flex-1 truncate">{branch.name}</span>
            {branch.current && branch.upstream && (
              <Badge variant="secondary" className="h-4 min-w-4 rounded-full px-1 text-[10px]">
                {aheadCommits}
              </Badge>
            )}
            {branch.current && <Check className="size-3.5 shrink-0" />}
          </button>
        ))}
      </div>
    </div>
  )
}

function CommitFileRow({ file }: { file: CommitFileItem }) {
  return (
    <div className="flex items-center gap-3 px-4 py-3">
      <Badge variant="outline" className="w-16 shrink-0 justify-center">
        {statusLabel(file.status)}
      </Badge>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm">{file.path}</div>
        {file.originalPath && (
          <div className="truncate text-xs text-muted-foreground">原路径: {file.originalPath}</div>
        )}
      </div>
    </div>
  )
}

function CommitRow({
  commit,
  selected,
  showDecorations,
  onSelect,
}: {
  commit: CommitItem
  selected: boolean
  showDecorations: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`flex w-full items-start gap-3 px-4 py-1 text-left text-xs transition-colors ${
        selected ? 'bg-[#0064E1] text-white' : 'hover:bg-muted/60'
      }`}
    >
      <span className={`mt-0.5 w-16 shrink-0 font-mono ${selected ? 'text-white' : 'text-primary'}`}>
        {commit.shortHash}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            <Badge
              variant="secondary"
              className={`max-w-32 truncate ${
                selected ? 'border-transparent bg-white/20 !text-white' : ''
              }`}
            >
              {commit.author}
            </Badge>
            <span className="truncate font-medium">{commit.subject}</span>
          </div>
          <span className={`shrink-0 ${selected ? 'text-white' : 'text-muted-foreground'}`}>
            {formatDate(commit.date)}
          </span>
        </div>
        {showDecorations && commit.decorations && (
          <div className={`mt-1 truncate ${selected ? 'text-white' : 'text-primary'}`}>
            {commit.decorations}
          </div>
        )}
      </div>
    </button>
  )
}

function ChangeGroup({
  title,
  changes,
  staged,
  selectedKey,
  busy,
  onSelect,
  onStage,
  onStageDirectory,
  onUnstage,
  onUnstageDirectory,
}: {
  title: string
  changes: Change[]
  staged: boolean
  selectedKey: string
  busy: boolean
  onSelect: (change: Change) => void
  onStage: (change: Change) => void
  onStageDirectory: (directory: string) => void
  onUnstage: (change: Change) => void
  onUnstageDirectory: (directory: string) => void
}) {
  const directories = new Map<string, Change[]>()
  changes.forEach((change) => {
    const separator = change.path.lastIndexOf('/')
    const directory = separator < 0 ? '' : change.path.slice(0, separator)
    directories.set(directory, [...(directories.get(directory) ?? []), change])
  })

  const listItems: ChangeListItem[] = []
  directories.forEach((group, directory) => {
    if (directory) listItems.push({ type: 'directory', directory })
    group.forEach((change) => listItems.push({ type: 'change', directory, change }))
  })

  const scrollRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: listItems.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 24,
    overscan: 10,
  })

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center justify-between px-3 text-xs font-medium uppercase text-muted-foreground">
        <span>{title}</span>
        <Badge variant="outline">{changes.length}</Badge>
      </div>
      {changes.length === 0 ? (
        <div className="px-3 text-xs text-muted-foreground">无文件</div>
      ) : (
        <ScrollArea className="min-h-0 flex-1 select-none" viewportRef={scrollRef}>
          <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
            {virtualizer.getVirtualItems().map((virtualItem) => {
              const item = listItems[virtualItem.index]
              return (
                <div
                  key={virtualItem.key}
                  className="absolute left-0 right-0"
                  style={{ top: virtualItem.start, height: virtualItem.size }}
                >
                  {item.type === 'directory' ? (
                    <div
                      className="flex h-6 cursor-pointer items-center gap-2 px-3 text-xs font-medium text-muted-foreground"
                      onDoubleClick={() => (staged ? onUnstageDirectory : onStageDirectory)(item.directory)}
                    >
                      <Folder className="size-3" />
                      <span className="truncate">{item.directory}</span>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        className="ml-auto"
                        onClick={() => (staged ? onUnstageDirectory : onStageDirectory)(item.directory)}
                        disabled={busy}
                        title={staged ? '取消暂存目录' : '暂存目录'}
                      >
                        {staged ? <Minus /> : <Plus />}
                      </Button>
                    </div>
                  ) : (
                    <div className="px-1">
                      <ChangeRow
                        key={`${staged ? 'staged' : 'unstaged'}:${item.change.path}`}
                        change={item.change}
                        staged={staged}
                        selected={selectedKey === `${staged ? 'staged' : 'unstaged'}:${item.change.path}`}
                        displayPath={
                          item.directory ? item.change.path.slice(item.directory.length + 1) : item.change.path
                        }
                        busy={busy}
                        onSelect={() => onSelect(item.change)}
                        onStage={() => onStage(item.change)}
                        onDoubleClick={() => (staged ? onUnstage : onStage)(item.change)}
                        onUnstage={() => onUnstage(item.change)}
                      />
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </ScrollArea>
      )}
    </div>
  )
}

function ChangeRow({
  change,
  staged,
  selected,
  displayPath,
  busy,
  onSelect,
  onStage,
  onDoubleClick,
  onUnstage,
}: {
  change: Change
  staged: boolean
  selected: boolean
  displayPath: string
  busy: boolean
  onSelect: () => void
  onStage: () => void
  onDoubleClick: () => void
  onUnstage: () => void
}) {
  const mark = statusMark(change.status)

  return (
    <div
      onDoubleClick={onDoubleClick}
      className={`flex h-6 items-center gap-2 rounded-md px-2 transition-colors ${
        selected ? 'bg-[#0064E1] text-white' : 'hover:bg-muted/60'
      }`}
    >
      <button
        type="button"
        onClick={onSelect}
        className="flex min-w-0 flex-1 items-center gap-2 text-left"
        title={change.path}
      >
        <Badge
          variant="outline"
          className={`${mark ? 'h-4 w-4 px-0 text-[10px] leading-none' : 'w-16'} ${mark?.className ?? ''} shrink-0 justify-center ${
            selected ? `${mark ? '' : 'border-white/40 bg-white/10'} !text-white` : ''
          }`}
        >
          {mark?.label ?? statusLabel(change.status)}
        </Badge>
        <span className="min-w-0 flex-1 truncate text-xs">{displayPath}</span>
      </button>
      <Button
        variant="ghost"
        size="icon-xs"
        onClick={staged ? onUnstage : onStage}
        disabled={busy}
        title={staged ? '取消暂存' : '暂存更改'}
      >
        {staged ? <Minus /> : <Plus />}
      </Button>
    </div>
  )
}

export default App
