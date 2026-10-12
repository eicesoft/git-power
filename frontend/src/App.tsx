import {
  Fragment,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type Dispatch,
  type DragEvent as ReactDragEvent,
  type MouseEvent as ReactMouseEvent,
  type ReactNode,
  type SetStateAction,
} from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Menu } from '@base-ui/react/menu'
import { Popover } from '@base-ui/react/popover'
import { ContextMenu } from '@base-ui/react/context-menu'
import {
  ChevronDown,
  ChevronRight,
  Archive,
  Check,
  Cloud,
  Copy,
  ExternalLink,
  EyeOff,
  FileText,
  Folder,
  FolderGit2,
  FolderX,
  GitBranch,
  Home,
  History,
  Monitor,
  Download,
  Upload,
  Minus,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Settings,
  Tag as TagIcon,
  Terminal,
  Trash2,
  X,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Toaster, toastError, toastSuccess } from '@/components/ui/toast'
import { DiffViewer } from '@/components/diff-viewer'
import { SettingsDialog } from '@/components/settings-dialog'
import { errorMessage } from '@/lib/utils'
import { Environment, ClipboardSetText } from '../wailsjs/runtime/runtime'
import {
  AddDashboardFolder,
  ClearGitCommandLogs,
  Commit,
  CreateBranch,
  DeleteDashboardFolder,
  DiscardChange,
  GetChangePath,
  GetGitCommandLogs,
  GetChangeDiff,
  GetCommitDetail,
  GetGitInfo,
  GetProjectSummary,
  GetSettings,
  IgnoreChange,
  ListDashboardRepositories,
  ListRecentProjects,
  MoveDashboardProject,
  OpenChange,
  Pull,
  Push,
  RecordProject,
  RefreshRepository,
  GitAction,
  RenameDashboardFolder,
  SaveSettings,
  SelectRepository,
  StageChange,
  SwitchBranch,
  SyncRepositories,
  UnstageChange,
  WindowClose,
  WindowIsMaximised,
  WindowMinimise,
  WindowToggleMaximise,
} from '../wailsjs/go/main/App'

type Repository = Awaited<ReturnType<typeof SelectRepository>>
type Branch = Repository['branches'][number]
type CommitItem = Repository['commits'][number]
type CommitDetailItem = Awaited<ReturnType<typeof GetCommitDetail>>
type CommitFileItem = CommitDetailItem['files'][number]
type Change = Repository['changes'][number]
type GitCommandLogItem = Awaited<ReturnType<typeof GetGitCommandLogs>>[number]
type ProjectInfoItem = Awaited<ReturnType<typeof ListRecentProjects>>[number]
type FolderItem = Awaited<ReturnType<typeof ListDashboardRepositories>>['folders'][number]
type ProjectSummaryItem = Awaited<ReturnType<typeof GetProjectSummary>>
type Settings = Awaited<ReturnType<typeof GetSettings>>
type GitInfoItem = Awaited<ReturnType<typeof GetGitInfo>>

type ChangeListItem =
  | { type: 'directory'; directory: string }
  | { type: 'change'; directory: string; change: Change }

const windowDragStyle = { '--wails-draggable': 'drag' } as CSSProperties
function startPaneDrag(
  event: ReactMouseEvent<HTMLDivElement>,
  onDrag: (delta: number, containerSize: number) => void,
  axis: 'x' | 'y' = 'x',
) {
  event.preventDefault()
  const startPos = axis === 'x' ? event.clientX : event.clientY
  const rect = (event.currentTarget.parentElement as HTMLElement).getBoundingClientRect()
  const containerSize = axis === 'x' ? rect.width : rect.height
  const onMove = (move: MouseEvent) => {
    const pos = axis === 'x' ? move.clientX : move.clientY
    onDrag(pos - startPos, containerSize)
  }
  const onUp = () => {
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}

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

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function formatRelativeTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const now = new Date()
  const minutes = Math.floor((now.getTime() - date.getTime()) / 60000)
  if (minutes < 10) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const dayDiff = Math.floor((startOfDay(now) - startOfDay(date)) / 86400000)
  if (dayDiff === 1) return '昨天'
  if (dayDiff === 2) return '前天'
  if (dayDiff < 7) return `${dayDiff}天前`
  return formatDate(value)
}

function statusLabel(status: string) {
  return statusLabels[status.trim()[0]] ?? status
}

function statusMark(status: string) {
  return statusMarks[status.trim()[0]]
}

function App() {
  const [repositories, setRepositories] = useState<Repository[]>([])
  const [activePath, setActivePath] = useState<string | null>(null)
  const [workspaceView, setWorkspaceView] = useState<'local' | 'commits'>('local')
  const [selectedChangeKey, setSelectedChangeKey] = useState('')
  const [sidebarSelected, setSidebarSelected] = useState('')
  const [selectedCommitHash, setSelectedCommitHash] = useState('')
  const [changeDiff, setChangeDiff] = useState('')
  const [commitDetail, setCommitDetail] = useState<CommitDetailItem | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailTab, setDetailTab] = useState('info')
  const [commitSubjects, setCommitSubjects] = useState<Record<string, string>>({})
  const [sidebarWidth, setSidebarWidth] = useState(256)
  const [changesPaneRatio, setChangesPaneRatio] = useState(0.42)

  const startSidebarDrag = (event: ReactMouseEvent<HTMLDivElement>) => {
    startPaneDrag(event, (deltaX) => {
      setSidebarWidth(Math.min(480, Math.max(160, Math.round(sidebarWidth + deltaX))))
    })
  }

  const startChangesPaneDrag = (event: ReactMouseEvent<HTMLDivElement>) => {
    startPaneDrag(event, (deltaX, containerWidth) => {
      setChangesPaneRatio(Math.min(0.7, Math.max(0.2, changesPaneRatio + deltaX / containerWidth)))
    })
  }

  const [diffPaneRatio, setDiffPaneRatio] = useState(0.55)

  const startDiffPaneDrag = (event: ReactMouseEvent<HTMLDivElement>) => {
    startPaneDrag(event, (deltaY, containerHeight) => {
      setDiffPaneRatio(Math.min(0.8, Math.max(0.2, diffPaneRatio + deltaY / containerHeight)))
    }, 'y')
  }

  const [commitsPaneRatio, setCommitsPaneRatio] = useState(0.5714)

  const startCommitsPaneDrag = (event: ReactMouseEvent<HTMLDivElement>) => {
    startPaneDrag(event, (deltaX, containerWidth) => {
      setCommitsPaneRatio(Math.min(0.8, Math.max(0.2, commitsPaneRatio + deltaX / containerWidth)))
    })
  }
  const [commitDescriptions, setCommitDescriptions] = useState<Record<string, string>>({})
  const [branchName, setBranchName] = useState('')
  const [creatingBranch, setCreatingBranch] = useState(false)
  const [busy, setBusy] = useState(false)
  const [syncing, setSyncing] = useState(false)
  // Pull/Push 弹窗关闭后仍在后台执行，用它驱动仓库信息框底部的进度条。
  const [remoteBusy, setRemoteBusy] = useState(false)
  // Repositories 目录树的展开状态由 App 持有：Dashboard 切换到仓库视图时会卸载，
  // 这样切回来仍保留原来的展开情况，且初始为空即默认全部折叠。
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(() => new Set())
  const [platform, setPlatform] = useState('')
  const [windowMaximised, setWindowMaximised] = useState(false)
  const [gitCommandLogs, setGitCommandLogs] = useState<GitCommandLogItem[]>([])
  const [gitLogOpen, setGitLogOpen] = useState(false)
  const [gitLogAnchor, setGitLogAnchor] = useState<{ x: number; y: number } | null>(null)
  const [expandedLogIds, setExpandedLogIds] = useState<Set<number>>(new Set())
  const [pendingDiscard, setPendingDiscard] = useState<Change | null>(null)
  const [pullOpen, setPullOpen] = useState(false)
  const [pullRemote, setPullRemote] = useState('')
  const [pullBranch, setPullBranch] = useState('')
  const [pullRebase, setPullRebase] = useState(false)
  const [pullStash, setPullStash] = useState(true)
  const [pushOpen, setPushOpen] = useState(false)
  const [pushBranch, setPushBranch] = useState('')
  const [pushTarget, setPushTarget] = useState('')
  const [pushTracking, setPushTracking] = useState(true)
  const [pushTags, setPushTags] = useState(false)
  const [pushForce, setPushForce] = useState(false)
  const [branchDialogOpen, setBranchDialogOpen] = useState(false)
  const [checkoutAfterCreate, setCheckoutAfterCreate] = useState(true)
  const [recentProjects, setRecentProjects] = useState<ProjectInfoItem[]>([])
  const [storedFolders, setStoredFolders] = useState<FolderItem[]>([])
  const [storedProjects, setStoredProjects] = useState<ProjectInfoItem[]>([])
  const [settings, setSettings] = useState<Settings | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [gitInfo, setGitInfo] = useState<GitInfoItem | null>(null)

  const activeRepository = useMemo(
    () => repositories.find((repository) => repository.path === activePath) ?? null,
    [activePath, repositories],
  )

  useEffect(() => {
    setCreatingBranch(false)
    setBranchName('')
    setSelectedChangeKey('')
    setSelectedCommitHash('')
    setSidebarSelected('')
    setChangeDiff('')
    setCommitDetail(null)
    setDetailTab('info')
    setPullOpen(false)
  }, [activePath])

  useEffect(() => {
    Environment()
      .then((environment) => setPlatform(environment.platform))
      .catch(() => setPlatform(''))
  }, [])

  const refreshGitInfo = () => {
    GetGitInfo()
      .then(setGitInfo)
      .catch(() => setGitInfo(null))
  }

  useEffect(() => {
    GetSettings()
      .then(setSettings)
      .catch((settingsError) => toastError(errorMessage(settingsError)))
  }, [])

  useEffect(() => {
    // 首次挂载与打开设置页时都刷新，避免提示停留在未检测状态。
    refreshGitInfo()
  }, [settingsOpen])

  // saveSettings 写入完整设置，成功后才刷新界面状态；失败时抛给设置页提示。
  const saveSettings = async (next: Settings) => {
    const gitPathChanged = settings?.gitPath !== next.gitPath
    await SaveSettings(next)
    setSettings(next)
    if (gitPathChanged) refreshGitInfo()
  }

  const refreshGitCommandLogs = () => {
    GetGitCommandLogs()
      .then(setGitCommandLogs)
      .catch((logError) => toastError(errorMessage(logError)))
  }

  useEffect(() => {
    refreshGitCommandLogs()
  }, [])

  const refreshStoredProjects = () => {
    ListRecentProjects(settings?.recentLimit ?? 10)
      .then(setRecentProjects)
      .catch(() => {})
    return ListDashboardRepositories()
      .then((data) => {
        setStoredFolders(data.folders)
        setStoredProjects(data.projects)
      })
      .catch(() => {})
  }

  useEffect(() => {
    if (!settings) return
    refreshStoredProjects()
  }, [settings?.recentLimit])

  const addDashboardFolder = async (parentId: number) => {
    try {
      const folder = await AddDashboardFolder('新建文件夹', parentId)
      await refreshStoredProjects()
      return folder
    } catch (folderError) {
      toastError(errorMessage(folderError))
      return null
    }
  }

  const renameDashboardFolder = (id: number, name: string) => {
    RenameDashboardFolder(id, name)
      .then(refreshStoredProjects)
      .catch((folderError) => toastError(errorMessage(folderError)))
  }

  const deleteDashboardFolder = (id: number) => {
    DeleteDashboardFolder(id)
      .then(refreshStoredProjects)
      .catch((folderError) => toastError(errorMessage(folderError)))
  }

  const moveDashboardProject = (path: string, folderId: number) => {
    MoveDashboardProject(path, folderId)
      .then(refreshStoredProjects)
      .catch((moveError) => toastError(errorMessage(moveError)))
  }

  // 同步扫描设置中的目录，把发现的仓库按目录结构登记进 Repositories。
  const syncRepositories = async () => {
    if (busy || syncing) return
    setSyncing(true)
    try {
      const result = await SyncRepositories()
      await refreshStoredProjects()
      toastSuccess(
        `发现 ${result.found} 个仓库，新增 ${result.added} 个，更新 ${result.updated} 个`,
        '同步完成',
      )
    } catch (syncError) {
      toastError(errorMessage(syncError))
    } finally {
      setSyncing(false)
    }
  }

  const updateRepository = (repository: Repository) => {
    if (!repository.path) return
    setRepositories((current) => {
      const exists = current.some((item) => item.path === repository.path)
      return exists
        ? current.map((item) => (item.path === repository.path ? repository : item))
        : [...current, repository]
    })
    setActivePath(repository.path)
    RecordProject(repository.path, repository.name)
      .then(refreshStoredProjects)
      .catch(() => {})
  }

  const openStoredProject = (path: string) => {
    if (busy) return
    runAction(() => RefreshRepository(path))
  }

  const runAction = async (action: () => Promise<Repository>) => {
    setBusy(true)
    try {
      updateRepository(await action())
    } catch (actionError) {
      toastError(errorMessage(actionError))
    } finally {
      setBusy(false)
      refreshGitCommandLogs()
    }
  }

  const openRepository = async () => {
    setBusy(true)
    try {
      const repository = await SelectRepository()
      if (repository.path) updateRepository(repository)
    } catch (actionError) {
      toastError(errorMessage(actionError))
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

  const defaultPullBranch = (repository: Repository, remote: string) => {
    const current = repository.branches.find((branch) => branch.current && !branch.remote)
    if (current?.upstream?.startsWith(`${remote}/`)) {
      return current.upstream.slice(remote.length + 1)
    }
    const first = repository.branches.find((branch) => branch.remote && branch.name.startsWith(`${remote}/`))
    return first ? first.name.slice(remote.length + 1) : ''
  }

  const openPullDialog = () => {
    if (!activeRepository || busy) return
    if (activeRepository.remotes.length === 0) {
      toastError('当前仓库没有配置远程仓库，无法 Pull', '无法 Pull')
      return
    }
    const remote = activeRepository.remotes.includes('origin') ? 'origin' : activeRepository.remotes[0]
    setPullRemote(remote)
    setPullBranch(defaultPullBranch(activeRepository, remote))
    setPullRebase(false)
    setPullStash(true)
    setPullOpen(true)
  }

  const pullFromRemote = () => {
    if (!activePath || busy || !pullRemote || !pullBranch) return
    const path = activePath
    setPullOpen(false)
    setRemoteBusy(true)
    runAction(() => Pull(path, pullRemote, pullBranch, pullRebase, pullStash)).finally(() =>
      setRemoteBusy(false),
    )
  }

  const openPushDialog = () => {
    if (!activeRepository || busy) return
    if (activeRepository.remotes.length === 0) {
      toastError('当前仓库没有配置远程仓库，无法 Push', '无法 Push')
      return
    }
    const remote = activeRepository.remotes.includes('origin') ? 'origin' : activeRepository.remotes[0]
    const current = activeRepository.branches.find((branch) => branch.current && !branch.remote)
    const branch = current?.name ?? ''
    let target = `${remote}/${branch}`
    if (current?.upstream?.startsWith(`${remote}/`)) {
      target = current.upstream
    }
    setPushBranch(branch)
    setPushTarget(target)
    setPushTracking(true)
    setPushTags(false)
    setPushForce(false)
    setPushOpen(true)
  }

  const pushToRemote = () => {
    if (!activePath || busy || !pushBranch || !pushTarget) return
    const remote = activeRepository?.remotes.find((candidate) => pushTarget.startsWith(`${candidate}/`))
    if (!remote) return
    const target = pushTarget.slice(remote.length + 1)
    const path = activePath
    setPushOpen(false)
    setRemoteBusy(true)
    runAction(() => Push(path, remote, pushBranch, target, pushTracking, pushTags, pushForce)).finally(
      () => setRemoteBusy(false),
    )
  }

  const createBranch = (checkout: boolean) => {
    if (!activePath || !branchName.trim() || busy) return
    const path = activePath
    const name = branchName.trim()
    runAction(async () => {
      const repository = await CreateBranch(path, name, checkout)
      setCreatingBranch(false)
      setBranchDialogOpen(false)
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

  const stageAll = () => {
    if (!activePath || busy) return
    runAction(() => StageChange(activePath, '.', ''))
  }

  const unstageAll = () => {
    if (!activePath || busy) return
    runAction(() => UnstageChange(activePath, '.', ''))
  }

  const openChange = (change: Change) => {
    if (!activePath) return
    OpenChange(activePath, change.path).catch((openError) => toastError(errorMessage(openError)))
  }

  const copyChangePath = (change: Change) => {
    if (!activePath) return
    GetChangePath(activePath, change.path)
      .then((absolutePath) => ClipboardSetText(absolutePath))
      .catch((copyError) => toastError(errorMessage(copyError)))
  }

  const ignoreChange = (change: Change) => {
    if (!activePath || busy) return
    runAction(() => IgnoreChange(activePath, change.path))
  }

  const discard = (change: Change) => {
    if (!activePath || busy) return
    runAction(() => DiscardChange(activePath, change.path, change.originalPath, change.status))
  }

  const requestDiscard = (change: Change) => {
    if (settings?.confirmDiscard === false) {
      discard(change)
      return
    }
    setPendingDiscard(change)
  }

  const discardChange = () => {
    if (!pendingDiscard) return
    const change = pendingDiscard
    setPendingDiscard(null)
    discard(change)
  }

  const commitChanges = () => {
    if (!activePath || busy) return
    const path = activePath
    const subject = (commitSubjects[path] ?? '').trim()
    const description = (commitDescriptions[path] ?? '').trim()
    if (!subject) {
      toastError('提交主题不能为空', '无法提交')
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
  const tags = activeRepository?.tags ?? []
  const stashes = activeRepository?.stashes ?? []
  const pullRemoteBranches = (activeRepository?.branches ?? [])
    .filter((branch) => branch.remote && branch.name.startsWith(`${pullRemote}/`))
    .map((branch) => branch.name.slice(pullRemote.length + 1))
  const pushRemoteBranches = activeRepository?.branches.filter((branch) => branch.remote).map((branch) => branch.name) ?? []
  const pushTargetExists = pushRemoteBranches.includes(pushTarget)
  const pushTargetOptions = pushTargetExists ? pushRemoteBranches : [pushTarget, ...pushRemoteBranches]
  const aheadCommits = activeRepository?.aheadCommits ?? 0
  const behindCommits = activeRepository?.behindCommits ?? 0
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
  const commitGraph = useMemo(
    () => buildCommitGraph(activeRepository?.commits ?? []),
    [activeRepository?.commits],
  )

  useEffect(() => {
    if (!activePath || workspaceView !== 'local' || !selectedChange) {
      setChangeDiff('')
      setDetailLoading(false)
      return
    }
    // 未跟踪目录（以 / 结尾）无法生成 diff，跳过以免 git diff --no-index 报错。
    if (selectedChange.change.path.endsWith('/')) {
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
        if (!cancelled) toastError(errorMessage(diffError))
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
        if (!cancelled) toastError(errorMessage(detailError))
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
      <header
        className={`flex h-11 shrink-0 select-none items-stretch gap-1 border-b bg-muted/40 ${platform === 'darwin' ? 'pl-28 pr-2' : 'pl-2 pr-2'}`}
        style={windowDragStyle}
      >
        <Tabs
          value={activePath ?? 'home'}
          onValueChange={(value) => setActivePath(value === 'home' ? null : String(value))}
          className="min-w-0 flex-1 self-center"
        >
          <TabsList className="min-w-0 h-9! justify-start overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            <TabsTrigger
              value="home"
              className={`${activePath === null ? 'w-32 justify-start' : 'w-9 justify-center'} shrink-0 text-[13px]`}
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
                  className="min-w-0 flex-1 justify-start pr-7 font-normal text-[13px]"
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
        {platform === 'windows' && (
          <div className="flex shrink-0 items-stretch" style={noDragStyle}>
            <button
              type="button"
              className="flex w-12 items-center justify-center text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              onClick={() => WindowMinimise()}
              title="最小化"
            >
              <Minus className="size-4" />
            </button>
            <button
              type="button"
              className="flex w-12 items-center justify-center text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              onClick={() =>
                WindowToggleMaximise().then(() => WindowIsMaximised().then(setWindowMaximised))
              }
              title={windowMaximised ? '还原' : '最大化'}
            >
              {windowMaximised ? (
                <svg className="size-3.5" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="0.9">
                  <rect x="2.5" y="0.5" width="7" height="7" rx="0.5" />
                  <path d="M0.5 2.5v6a1 1 0 0 0 1 1h6" />
                </svg>
              ) : (
                <svg className="size-3.5" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="0.9">
                  <rect x="0.75" y="0.75" width="8.5" height="8.5" rx="0.5" />
                </svg>
              )}
            </button>
            <button
              type="button"
              className="flex w-12 items-center justify-center text-muted-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground"
              onClick={() => WindowClose()}
              title="关闭"
            >
              <X className="size-4" />
            </button>
          </div>
        )}
      </header>

      {!activeRepository ? (
        <Dashboard
          busy={busy}
          syncing={syncing}
          expandedFolders={expandedFolders}
          onExpandedFoldersChange={setExpandedFolders}
          recentProjects={recentProjects}
          folders={storedFolders}
          projects={storedProjects}
          onOpen={openRepository}
          onOpenProject={openStoredProject}
          onAddFolder={addDashboardFolder}
          onRenameFolder={renameDashboardFolder}
          onDeleteFolder={deleteDashboardFolder}
          onMoveProject={moveDashboardProject}
          onSync={syncRepositories}
          onOpenSettings={() => setSettingsOpen(true)}
        />
      ) : (
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1">
            <aside className="flex min-w-40 shrink-0 flex-col" style={{ width: sidebarWidth }}>
            <Tabs
              value={workspaceView}
              onValueChange={(value) => setWorkspaceView(value as 'local' | 'commits')}
              className="shrink-0 p-1"
            >
              <TabsList className="h-auto w-full gap-1 rounded-lg bg-muted/50 p-1">
                <TabsTrigger
                  value="local"
                  className="h-7 flex-1 justify-center gap-1.5 rounded-md text-xs dark:data-active:border-transparent"
                >
                  <FileText className="size-3.5" />
                  Local Changes
                </TabsTrigger>
                <TabsTrigger
                  value="commits"
                  className="relative h-7 flex-1 justify-center gap-1.5 rounded-md text-xs after:absolute after:inset-y-1 after:left-0 after:w-px after:bg-border after:opacity-100 after:content-[''] dark:data-active:border-transparent"
                >
                  <History className="size-3.5" />
                  Commits
                </TabsTrigger>
              </TabsList>
            </Tabs>

            <div className="flex h-8 shrink-0 items-center justify-between border-b px-2">
              <div className="flex items-center gap-1.5">
                <FolderGit2 className="size-3.5 text-primary" />
                <span className="text-xs font-semibold">{activeRepository?.name ?? '仓库'}</span>
                <Badge variant="secondary" className="h-4 px-1.5 text-[10px]">
                  {localBranches.length + remoteBranches.length + tags.length + stashes.length}
                </Badge>
              </div>
              <div className="flex items-center gap-0.5">
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => setCreatingBranch((value) => !value)}
                  disabled={busy}
                  title="新建分支"
                >
                  <Plus />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-xs"
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
                    if (event.key === 'Enter') createBranch(true)
                    if (event.key === 'Escape') setCreatingBranch(false)
                  }}
                  placeholder="新分支名称"
                />
                <Button
                  size="icon"
                  className="size-9"
                  onClick={() => createBranch(true)}
                  disabled={!branchName.trim() || busy}
                  title="创建分支"
                >
                  <Check />
                </Button>
              </div>
            )}

            <ScrollArea className="min-h-0 flex-1">
              <div className="space-y-3 p-2">
                <SidebarSection
                  title="Branches"
                  icon={<GitBranch className="size-3.5" />}
                  count={localBranches.length}
                >
                  {localBranches.map((branch) => (
                    <SidebarRow
                      key={branch.ref}
                      icon={<GitBranch className="size-3 shrink-0" />}
                      label={branch.name}
                      detail={branch.upstream ? `上游: ${branch.upstream}` : branch.ref}
                      badge={
                        <>
                          {branch.upstream && branch.aheadCommits > 0 && (
                            <Badge
                              variant="secondary"
                              className="h-3.5 min-w-3.5 rounded-full px-1 text-[9px]"
                              title={`${branch.aheadCommits} 个未推送提交`}
                            >
                              ↑{branch.aheadCommits}
                            </Badge>
                          )}
                          {branch.upstream && branch.behindCommits > 0 && (
                            <Badge
                              className="h-3.5 min-w-3.5 rounded-full px-1 text-[9px]"
                              title={`${branch.behindCommits} 个未拉取提交`}
                            >
                              ↓{branch.behindCommits}
                            </Badge>
                          )}
                          {branch.current && <Check className="size-3 shrink-0" />}
                        </>
                      }
                      current={branch.current}
                      selected={sidebarSelected === branch.ref}
                      disabled={busy || branch.current}
                      onClick={() => setSidebarSelected(branch.ref)}
                      onDoubleClick={() => switchBranch(branch)}
                    />
                  ))}
                </SidebarSection>

                <SidebarSection
                  title="Remotes"
                  icon={<Cloud className="size-3.5" />}
                  count={remoteBranches.length}
                >
                  <RemoteBranchNodes
                    branches={remoteBranches}
                    busy={busy}
                    selected={sidebarSelected}
                    onSelectItem={setSidebarSelected}
                    onOpen={switchBranch}
                  />
                </SidebarSection>

                <SidebarSection title="Tags" icon={<TagIcon className="size-3.5" />} count={tags.length}>
                  {tags.map((tag) => (
                    <SidebarRow
                      key={tag.ref}
                      icon={<TagIcon className="size-3 shrink-0" />}
                      label={tag.name}
                      detail={`${tag.ref} · ${tag.hash}`}
                      selected={sidebarSelected === tag.ref}
                      onClick={() => setSidebarSelected(tag.ref)}
                    />
                  ))}
                </SidebarSection>

                <SidebarSection title="Stashes" icon={<Archive className="size-3.5" />} count={stashes.length}>
                  {stashes.map((stash) => (
                    <SidebarRow
                      key={stash.ref}
                      icon={<Archive className="size-3 shrink-0" />}
                      label={stash.message}
                      detail={`${stash.ref} · ${stash.hash} · ${stash.message}`}
                      selected={sidebarSelected === stash.ref}
                      onClick={() => setSidebarSelected(stash.ref)}
                    />
                  ))}
                </SidebarSection>
              </div>
            </ScrollArea>
          </aside>

          <div
            role="separator"
            aria-orientation="vertical"
            className="relative w-px shrink-0 cursor-col-resize bg-border transition-colors before:absolute before:inset-y-0 before:-left-1 before:w-2 before:content-[''] hover:bg-primary/50 active:bg-primary"
            style={noDragStyle}
            onMouseDown={startSidebarDrag}
          />

          <main className="flex min-w-0 flex-1 flex-col bg-muted/30">
          {activeRepository && (
            <div className="flex h-[64px] shrink-0 items-center border-b px-2 py-1" style={noDragStyle}>
            <div className="flex min-w-0 flex-1 items-center justify-center gap-10">
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
                    className={`h-12 ${
                      (item.action === 'push' && currentBranchHasUpstream) ||
                      (item.action === 'pull' && currentBranchHasUpstream && behindCommits > 0)
                        ? 'w-16'
                        : 'w-10'
                    } flex-col gap-0.5 rounded-md px-1 text-[10px] text-muted-foreground hover:bg-transparent hover:text-foreground dark:hover:bg-transparent`}
                    onClick={() =>
                      item.action === 'pull'
                        ? openPullDialog()
                        : item.action === 'push'
                          ? openPushDialog()
                          : runGitAction(item.action)}
                    disabled={busy}
                    title={
                      item.action === 'push' && currentBranchHasUpstream
                        ? `${item.label}（${aheadCommits} 个未推送提交）`
                        : item.action === 'pull' && currentBranchHasUpstream
                          ? `${item.label}（${behindCommits} 个未拉取提交）`
                          : item.label
                    }
                  >
                    <span className="relative flex h-8 w-8 items-center justify-center rounded-md p-1 group-hover/button:bg-muted">
                      <item.icon className="size-6" />
                      {item.action === 'push' && currentBranchHasUpstream && (
                        <Badge className="absolute -right-2.5 top-1 h-3.5 min-w-3.5 rounded-full px-1 text-[9px] leading-3">
                          {aheadCommits}
                        </Badge>
                      )}
                      {item.action === 'pull' && currentBranchHasUpstream && behindCommits > 0 && (
                        <Badge className="absolute -right-2.5 top-1 h-3.5 min-w-3.5 rounded-full px-1 text-[9px] leading-3">
                          {behindCommits}
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
              className="relative mx-2 flex h-12 min-w-64 max-w-xs flex-1 items-center gap-1 rounded-md bg-[#F5F5F5] px-2 dark:bg-[#262626]"
              title={activeRepository.path}
            >
              {remoteBusy && (
                <div className="absolute inset-x-0 bottom-0 h-[2px] overflow-hidden rounded-b-md">
                  <div className="h-full w-1/4 bg-primary remote-progress" />
                </div>
              )}
              <Popover.Root
                open={gitLogOpen}
                onOpenChange={(open) => {
                  setGitLogOpen(open)
                  if (open) refreshGitCommandLogs()
                }}
              >
                <Popover.Trigger
                  render={(props) => (
                    <Button
                      {...props}
                      variant="ghost"
                      size="icon"
                      className={`absolute left-1 top-1/2 -translate-y-1/2 hover:bg-transparent hover:text-muted-foreground dark:hover:bg-transparent dark:hover:text-muted-foreground ${gitLogOpen ? 'text-primary' : ''}`}
                      title="Git 命令行日志"
                      onClick={(event) => {
                        setGitLogAnchor({ x: event.clientX, y: event.clientY })
                        props.onClick?.(event as ReactMouseEvent<HTMLButtonElement>)
                      }}
                    >
                      <span className="flex h-8 w-8 items-center justify-center rounded-md">
                        <Terminal className="size-4" />
                      </span>
                    </Button>
                  )}
                />
                <Popover.Portal>
                  <Popover.Positioner
                    anchor={() => (gitLogAnchor ? { getBoundingClientRect: () => ({
                      x: gitLogAnchor.x,
                      y: gitLogAnchor.y,
                      top: gitLogAnchor.y,
                      left: gitLogAnchor.x,
                      right: gitLogAnchor.x,
                      bottom: gitLogAnchor.y,
                      width: 0,
                      height: 0,
                    }) } : null)}
                    side="bottom"
                    align="center"
                    sideOffset={16}
                    collisionPadding={8}
                  >
                    <Popover.Popup className="relative w-[min(760px,calc(100vw-2rem))] animate-in fade-in-0 slide-in-from-bottom-2 select-none rounded-xl border bg-popover p-0 shadow-xl outline-none">
                      <div aria-hidden className="absolute -top-[5px] left-1/2 z-0 size-3 -translate-x-1/2 rotate-45 rounded-[2px] border-t border-l border-border bg-popover" />
                      <div className="flex h-10 items-center justify-between border-b px-3">
                        <div className="flex items-center gap-2 text-sm font-medium">
                          <Terminal className="size-4 text-primary" />
                          Git 命令行日志
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            ClearGitCommandLogs()
                              .then(setGitCommandLogs)
                              .catch((logError) => toastError(errorMessage(logError)))
                          }}
                          disabled={gitCommandLogs.length === 0}
                        >
                          清空
                        </Button>
                      </div>
                      <ScrollArea className="h-80">
                        {gitCommandLogs.length === 0 ? (
                          <div className="p-6 text-center text-sm text-muted-foreground">暂无 Git 命令日志</div>
                        ) : (
                          <div className="divide-y">
                            {[...gitCommandLogs].reverse().map((log) => (
                              <div key={log.id} className="space-y-1 px-3 py-2">
                                <div className="flex items-start gap-2">
                                  <button
                                    type="button"
                                    className="mt-0.5 shrink-0 text-muted-foreground hover:text-foreground"
                                    title={expandedLogIds.has(log.id) ? '收起输出' : '展开输出'}
                                    onClick={() =>
                                      setExpandedLogIds((prev) => {
                                        const next = new Set(prev)
                                        if (next.has(log.id)) next.delete(log.id)
                                        else next.add(log.id)
                                        return next
                                      })
                                    }
                                  >
                                    <ChevronRight
                                      className={`size-3.5 transition-transform ${expandedLogIds.has(log.id) ? 'rotate-90' : ''}`}
                                    />
                                  </button>
                                  <span
                                    className={`mt-1 size-1.5 shrink-0 rounded-full ${
                                      log.success ? 'bg-green-500' : 'bg-red-500'
                                    }`}
                                  />
                                  <Tooltip>
                                    <TooltipTrigger
                                      render={(props) => (
                                        <code
                                          {...props}
                                          className="min-w-0 flex-1 cursor-pointer truncate font-mono text-[11px]"
                                          onClick={() =>
                                            setExpandedLogIds((prev) => {
                                              const next = new Set(prev)
                                              if (next.has(log.id)) next.delete(log.id)
                                              else next.add(log.id)
                                              return next
                                            })
                                          }
                                        />
                                      )}
                                    >
                                      {log.command}
                                    </TooltipTrigger>
                                    <TooltipContent className="max-w-[min(640px,calc(100vw-2rem))] break-all font-mono">
                                      {log.command}
                                    </TooltipContent>
                                  </Tooltip>
                                  <span className="shrink-0 font-mono text-[10px] text-muted-foreground">
                                    {log.duration} ms
                                  </span>
                                </div>
                                {log.output && expandedLogIds.has(log.id) && (
                                  <pre className="scroll-on-hover max-h-40 overflow-auto whitespace-pre-wrap break-all rounded bg-muted/40 p-2 font-mono text-[10px] text-muted-foreground">
                                    {log.output}
                                  </pre>
                                )}
                                {log.error && expandedLogIds.has(log.id) && (
                                  <pre className="scroll-on-hover max-h-40 overflow-auto whitespace-pre-wrap break-all rounded bg-destructive/10 p-2 font-mono text-[10px] text-destructive">
                                    {log.error}
                                  </pre>
                                )}
                              </div>
                            ))}
                          </div>
                        )}
                      </ScrollArea>
                    </Popover.Popup>
                  </Popover.Positioner>
                </Popover.Portal>
              </Popover.Root>
              <div className="flex min-w-0 flex-1 flex-col items-center justify-center gap-0.5 px-8">
              <span
                className="w-full truncate text-center text-xs"
                title={activeRepository.name}
              >
                {activeRepository.name}
              </span>
              <Menu.Root>
                <Menu.Trigger
                  render={(props) => (
                    <Button
                      {...props}
                      variant="ghost"
                      className="h-6 min-w-0 max-w-48 shrink justify-center gap-1 px-2 text-xs font-normal hover:bg-transparent dark:hover:bg-transparent"
                      title="选择分支"
                    >
                      <span className="flex h-6 items-center gap-1 rounded-md px-1.5 text-muted-foreground group-hover/button:text-foreground">
                        <GitBranch className="size-3 shrink-0" />
                        <span className="min-w-0 truncate">{activeRepository.currentBranch || 'main'}</span>
                        <ChevronDown className="size-3 shrink-0" />
                      </span>
                    </Button>
                  )}
                />
                <Menu.Portal>
                  <Menu.Positioner align="center">
                    <Menu.Popup className="scroll-on-hover max-h-72 min-w-72 overflow-auto rounded-lg border bg-popover p-1 shadow-lg">
                      <div className="px-2 py-1 text-xs font-medium text-muted-foreground">
                        本地分支
                      </div>
                      {localBranches.map((branch) => (
                        <Menu.Item
                          key={branch.name}
                          className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted"
                                                    onClick={() => switchBranch(branch)}
                          disabled={branch.current}
                        >
                          <GitBranch className="size-3.5 shrink-0 text-muted-foreground" />
                          <span className="min-w-0 flex-1 truncate">{branch.name}</span>
                          {branch.upstream && branch.aheadCommits > 0 && (
                            <Badge
                              variant="secondary"
                              className="h-4 rounded-full px-1 text-[9px]"
                              title={`${branch.aheadCommits} 个未推送提交`}
                            >
                              ↑{branch.aheadCommits}
                            </Badge>
                          )}
                          {branch.upstream && branch.behindCommits > 0 && (
                            <Badge
                              className="h-4 rounded-full px-1 text-[9px]"
                              title={`${branch.behindCommits} 个未拉取提交`}
                            >
                              ↓{branch.behindCommits}
                            </Badge>
                          )}
                          {branch.current && <Check className="size-3.5 shrink-0 text-primary" />}
                        </Menu.Item>
                      ))}
                      {remoteBranches.length > 0 && (
                        <>
                          <Separator className="my-1" />
                          <div className="px-2 py-1 text-xs font-medium text-muted-foreground">
                            远程分支
                          </div>
                          {remoteBranches.map((branch) => (
                            <Menu.Item
                              key={branch.ref}
                              className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted"
                                                        onClick={() => switchBranch(branch)}
                            >
                              <Cloud className="size-3.5 shrink-0 text-muted-foreground" />
                              <span className="min-w-0 flex-1 truncate">{branch.name}</span>
                            </Menu.Item>
                          ))}
                        </>
                      )}
                    </Menu.Popup>
                  </Menu.Positioner>
                </Menu.Portal>
              </Menu.Root>
              </div>
            </div>

            <Button
              variant="ghost"
              className="h-12 w-10 flex-col gap-0.5 rounded-md px-1 text-[10px] text-muted-foreground hover:bg-transparent hover:text-foreground dark:hover:bg-transparent"
              onClick={() => {
                setBranchName('')
                setCreatingBranch(false)
                setCheckoutAfterCreate(true)
                setBranchDialogOpen(true)
              }}
              disabled={busy}
              title="Branch"
            >
              <span className="relative flex h-8 w-8 items-center justify-center rounded-md p-1 group-hover/button:bg-muted">
                <GitBranch className="size-6" />
              </span>
              <span>Branch</span>
            </Button>

            </div>

            <div className="flex shrink-0 items-center gap-10">
              <Menu.Root>
                <Menu.Trigger
                  render={(props) => (
                    <Button
                      {...props}
                      variant="ghost"
                      className="h-12 min-w-10 flex-col gap-0.5 rounded-md px-1 text-[10px] text-muted-foreground hover:bg-transparent hover:text-foreground dark:hover:bg-transparent"
                      title="Open In"
                    >
                      <span className="relative flex h-8 w-8 items-center justify-center rounded-md p-1 group-hover/button:bg-muted">
                        <Monitor className="size-6" />
                      </span>
                      <span>Open In</span>
                    </Button>
                  )}
                />
                <Menu.Portal>
                  <Menu.Positioner align="end">
                    <Menu.Popup className="min-w-40 rounded-lg border bg-popover p-1 shadow-lg">
                      <Menu.Item
                        className="cursor-pointer rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted"
                        onClick={() => window.open(`file://${activeRepository.path}`)}
                      >
                        Finder
                      </Menu.Item>
                    </Menu.Popup>
                  </Menu.Positioner>
                </Menu.Portal>
              </Menu.Root>
            </div>
          </div>
        )}
            {workspaceView === 'local' ? (
              <div className="flex min-h-0 flex-1">
                <section
                  className="flex min-w-64 shrink-0 flex-col"
                  style={{ width: `${changesPaneRatio * 100}%` }}
                >
                  <div className="flex h-8 shrink-0 items-center justify-between border-b px-3">
                    <div className="flex items-center gap-1.5">
                      <FileText className="size-3.5 text-primary" />
                      <span className="text-xs font-semibold">文件更改</span>
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
                          onStageAll={stageAll}
                          selectedKey={selectedChange?.key ?? ''}
                          busy={busy}
                          onSelect={(change) => setSelectedChangeKey(`unstaged:${change.path}`)}
                          onStage={stageChange}
                          onStageDirectory={stageDirectory}
                          onUnstage={unstageChange}
                          onUnstageDirectory={unstageDirectory}
                          onOpen={openChange}
                          onCopyPath={copyChangePath}
                          onIgnore={ignoreChange}
                          onDiscard={requestDiscard}
                        />
                      </div>
                      <div className="flex min-h-0 flex-[3_1_0%] border-t">
                        <ChangeGroup
                          title="Staged"
                          changes={stagedChanges}
                          staged
                          onUnstageAll={unstageAll}
                          selectedKey={selectedChange?.key ?? ''}
                          busy={busy}
                          onSelect={(change) => setSelectedChangeKey(`staged:${change.path}`)}
                          onStage={stageChange}
                          onStageDirectory={stageDirectory}
                          onUnstage={unstageChange}
                          onUnstageDirectory={unstageDirectory}
                          onOpen={openChange}
                          onCopyPath={copyChangePath}
                          onIgnore={ignoreChange}
                          onDiscard={requestDiscard}
                        />
                      </div>
                    </div>
                  )}
                </section>

                <div
                  role="separator"
                  aria-orientation="vertical"
                  className="relative w-px shrink-0 cursor-col-resize bg-border transition-colors before:absolute before:inset-y-0 before:-left-1 before:w-2 before:content-[''] hover:bg-primary/50 active:bg-primary"
                  style={noDragStyle}
                  onMouseDown={startChangesPaneDrag}
                />

                <section className="flex min-w-0 flex-1 flex-col">
                  <div className="flex h-8 shrink-0 items-center justify-between border-b px-3">
                    <div className="flex min-w-0 items-center gap-1.5">
                      <FileText className="size-3.5 shrink-0 text-primary" />
                      <span className="truncate text-xs font-semibold">
                        {selectedChange ? selectedChange.change.path : '文件更改'}
                      </span>
                    </div>
                    {selectedChange && (
                      <Badge
                        variant={selectedChange.staged ? 'secondary' : 'outline'}
                        className="h-4 px-1.5 text-[10px]"
                      >
                        {selectedChange.staged ? 'Staged' : 'Unstaged'}
                      </Badge>
                    )}
                  </div>
                  <ScrollArea className="min-h-0" style={{ flex: `${diffPaneRatio} 1 0%` }}>
                    {!selectedChange ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">选择文件查看更改</div>
                    ) : detailLoading ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">读取更改...</div>
                    ) : changeDiff ? (
                      <DiffViewer
                        diff={changeDiff}
                        fontFamily={settings?.diffFontFamily}
                        fontSize={settings?.diffFontSize}
                      />
                    ) : (
                      <div className="p-6 text-center text-sm text-muted-foreground">无文本更改</div>
                    )}
                  </ScrollArea>

                  <div
                    role="separator"
                    aria-orientation="horizontal"
                    className="relative h-px shrink-0 cursor-row-resize bg-border transition-colors before:absolute before:inset-x-0 before:-top-1 before:h-2 before:content-[''] hover:bg-primary/50 active:bg-primary"
                    style={noDragStyle}
                    onMouseDown={startDiffPaneDrag}
                  />

                  <div
                    className="flex min-h-0 flex-col gap-3 p-4"
                    style={{ flex: `${1 - diffPaneRatio} 1 0%` }}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Check className="size-4 text-primary" />
                        <span className="text-sm font-medium">Commit</span>
                      </div>
                      <span className="text-xs text-muted-foreground">{stagedChanges.length} 个暂存文件</span>
                    </div>
                    <label className="block">
                      <Input
                        value={commitSubjects[activeRepository.path] ?? ''}
                        onChange={(event) =>
                          setCommitSubjects((drafts) => ({
                            ...drafts,
                            [activeRepository.path]: event.target.value,
                          }))
                        }
                        placeholder="Commit Subject"
                        disabled={busy}
                      />
                    </label>
                    <label className="flex min-h-0 flex-1 flex-col">
                      <textarea
                        className="min-h-20 flex-1 resize-none rounded-md border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                        value={commitDescriptions[activeRepository.path] ?? ''}
                        onChange={(event) =>
                          setCommitDescriptions((drafts) => ({
                            ...drafts,
                            [activeRepository.path]: event.target.value,
                          }))
                        }
                        placeholder="Description（可选）"
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
                <section
                  className="flex min-w-64 shrink-0 flex-col"
                  style={{ width: `${commitsPaneRatio * 100}%` }}
                >
                  <div className="flex h-8 shrink-0 items-center justify-between border-b px-3">
                    <div className="flex items-center gap-1.5">
                      <History className="size-3.5 text-primary" />
                      <span className="text-xs font-semibold">提交历史</span>
                      <Badge variant="outline" className="h-4 max-w-40 truncate px-1.5 text-[10px]">
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
                            graph={commitGraph.rows.get(commit.hash)}
                            graphWidth={commitGraph.width}
                            selected={commit.hash === selectedCommit?.hash}
                            showDecorations={index > 0}
                            onSelect={() => setSelectedCommitHash(commit.hash)}
                          />
                        ))}
                      </div>
                    )}
                  </ScrollArea>
                </section>

                <div
                  role="separator"
                  aria-orientation="vertical"
                  className="relative w-px shrink-0 cursor-col-resize bg-border transition-colors before:absolute before:inset-y-0 before:-left-1 before:w-2 before:content-[''] hover:bg-primary/50 active:bg-primary"
                  style={noDragStyle}
                  onMouseDown={startCommitsPaneDrag}
                />

                <Tabs
                  value={detailTab}
                  onValueChange={(value) => setDetailTab(String(value))}
                  className="min-h-0 min-w-0 flex-1"
                >
                  <div className="flex h-8 shrink-0 items-center justify-between border-b px-3">
                    <TabsList variant="line" className="h-7!">
                      <TabsTrigger value="info" className="text-xs">
                        提交信息
                      </TabsTrigger>
                      <TabsTrigger value="files" className="text-xs">
                        更改文件
                      </TabsTrigger>
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
                        <div className="max-w-3xl space-y-3 p-4">
                          <div className="space-y-0.5">
                            <div className="text-[11px] font-medium uppercase text-muted-foreground">Subject</div>
                            <div className="text-sm font-semibold leading-5">{commitDetail.subject}</div>
                          </div>
                          <div className="space-y-0.5">
                            <div className="text-[11px] font-medium uppercase text-muted-foreground">Description</div>
                            <div className="whitespace-pre-wrap text-xs leading-5">
                              {commitDetail.body || '无详细说明'}
                            </div>
                          </div>
                          <div className="grid gap-2 text-xs sm:grid-cols-2">
                            <div className="space-y-0.5">
                              <div className="text-[11px] font-medium uppercase text-muted-foreground">Author</div>
                              <div>{commitDetail.author}</div>
                            </div>
                            <div className="space-y-0.5">
                              <div className="text-[11px] font-medium uppercase text-muted-foreground">Date</div>
                              <div>{formatDate(commitDetail.date)}</div>
                            </div>
                            <div className="space-y-0.5 sm:col-span-2">
                              <div className="text-[11px] font-medium uppercase text-muted-foreground">Hash</div>
                              <div className="break-all font-mono text-[11px]">{commitDetail.hash}</div>
                            </div>
                          </div>
                        </div>
                      )}
                    </ScrollArea>
                  </TabsContent>

                  <TabsContent value="files" className="min-h-0 flex-1 overflow-hidden">
                    {!commitDetail ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">
                        {detailLoading ? '读取更改文件...' : '暂无更改文件'}
                      </div>
                    ) : commitDetail.files.length === 0 ? (
                      <div className="p-6 text-center text-sm text-muted-foreground">无更改文件</div>
                    ) : (
                      <CommitFileList files={commitDetail.files} />
                    )}
                  </TabsContent>
                </Tabs>
              </div>
            )}
          </main>
          </div>
        </div>
      )}

      {activeRepository && (
        <>
        <Dialog open={branchDialogOpen} onOpenChange={setBranchDialogOpen}>
          <DialogContent className="sm:max-w-md" showCloseButton={false}>
            <DialogHeader>
              <DialogTitle>Create Branch</DialogTitle>
              <DialogDescription>Create new branch</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2.5">
              <span className="text-right text-sm text-muted-foreground">Create branch at:</span>
              <div className="flex min-w-0 items-center gap-1.5 text-sm">
                <GitBranch className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate">{activeRepository.currentBranch || 'main'}</span>
              </div>
              <span className="text-right text-sm text-muted-foreground">Branch name:</span>
              <div className="flex min-w-0 items-center gap-1.5">
                <GitBranch className="size-3.5 shrink-0 text-muted-foreground" />
                <Input
                  autoFocus
                  value={branchName}
                  onChange={(event) => setBranchName(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') createBranch(checkoutAfterCreate)
                  }}
                  placeholder="新分支名称"
                />
              </div>
              <label className="col-start-2 flex cursor-pointer items-center gap-2 text-sm">
                <Checkbox
                  checked={checkoutAfterCreate}
                  onCheckedChange={(checked) => setCheckoutAfterCreate(checked === true)}
                />
                Check out after create
              </label>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setBranchDialogOpen(false)} disabled={busy}>
                Cancel
              </Button>
              <Button
                onClick={() => createBranch(checkoutAfterCreate)}
                disabled={busy || !branchName.trim()}
              >
                {checkoutAfterCreate ? 'Create and Checkout' : 'Create'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog open={pullOpen} onOpenChange={setPullOpen}>
          <DialogContent className="sm:max-w-md" showCloseButton={false}>
            <DialogHeader>
              <DialogTitle>Pull</DialogTitle>
              <DialogDescription>
                Pull remote changes and merges them into your local branch
              </DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2.5">
              <span className="text-right text-sm text-muted-foreground">Remote:</span>
              <Select
                value={pullRemote}
                items={activeRepository.remotes.map((remote) => ({ value: remote, label: remote }))}
                onValueChange={(value) => {
                  const remote = String(value)
                  setPullRemote(remote)
                  setPullBranch(defaultPullBranch(activeRepository, remote))
                }}
              >
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {activeRepository.remotes.map((remote) => (
                    <SelectItem key={remote} value={remote}>
                      {remote}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <span className="text-right text-sm text-muted-foreground">Branch:</span>
              <Select
                value={pullBranch}
                items={pullRemoteBranches.map((branch) => ({ value: branch, label: branch }))}
                onValueChange={(value) => setPullBranch(String(value))}
              >
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {pullRemoteBranches.map((branch) => (
                    <SelectItem key={branch} value={branch}>
                      {branch}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <span className="text-right text-sm text-muted-foreground">Into:</span>
              <div className="flex min-w-0 items-center gap-1.5 text-sm">
                <GitBranch className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate">{activeRepository.currentBranch || '未初始化分支'}</span>
              </div>
              <div className="col-start-2 flex flex-col gap-2.5">
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <Checkbox
                    checked={pullRebase}
                    onCheckedChange={(checked) => setPullRebase(checked === true)}
                  />
                  Rebase instead of merge
                </label>
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <Checkbox
                    checked={pullStash}
                    onCheckedChange={(checked) => setPullStash(checked === true)}
                  />
                  Stash and reapply local changes
                </label>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setPullOpen(false)} disabled={busy}>
                Cancel
              </Button>
              <Button
                onClick={pullFromRemote}
                disabled={busy || !pullRemote || !pullBranch || pullRemoteBranches.length === 0}
              >
                Pull
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <Dialog open={pushOpen} onOpenChange={setPushOpen}>
          <DialogContent className="sm:max-w-md" showCloseButton={false}>
            <DialogHeader>
              <DialogTitle>Push</DialogTitle>
              <DialogDescription>Push your local changes to remote repository</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2.5">
              <span className="text-right text-sm text-muted-foreground">Branch:</span>
              <Select
                value={pushBranch}
                items={localBranches.map((branch) => ({ value: branch.name, label: branch.name }))}
                onValueChange={(value) => {
                  const branch = String(value)
                  setPushBranch(branch)
                  const remote = activeRepository.remotes.find((candidate) =>
                    pushTarget.startsWith(`${candidate}/`),
                  )
                  if (remote) setPushTarget(`${remote}/${branch}`)
                }}
              >
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {localBranches.map((branch) => (
                    <SelectItem key={branch.name} value={branch.name}>
                      {branch.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <span className="text-right text-sm text-muted-foreground">To:</span>
              <Select
                value={pushTarget}
                items={pushTargetOptions.map((option) => ({
                  value: option,
                  label: pushRemoteBranches.includes(option) ? option : `new (${option})`,
                }))}
                onValueChange={(value) => setPushTarget(String(value))}
              >
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {pushTargetOptions.map((option) => (
                    <SelectItem key={option} value={option}>
                      {pushRemoteBranches.includes(option) ? option : `new (${option})`}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <div className="col-start-2 flex flex-col gap-2.5">
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <Checkbox
                    checked={pushTracking}
                    onCheckedChange={(checked) => setPushTracking(checked === true)}
                  />
                  Create tracking reference
                </label>
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <Checkbox
                    checked={pushTags}
                    onCheckedChange={(checked) => setPushTags(checked === true)}
                  />
                  Push all tags
                </label>
                <label className="flex cursor-pointer items-center gap-2 text-sm">
                  <Checkbox
                    checked={pushForce}
                    onCheckedChange={(checked) => setPushForce(checked === true)}
                  />
                  Force push
                </label>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setPushOpen(false)} disabled={busy}>
                Cancel
              </Button>
              <Button onClick={pushToRemote} disabled={busy || !pushBranch || !pushTarget}>
                Push
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        </>
      )}

      {settings && (
        <SettingsDialog
          open={settingsOpen}
          onOpenChange={setSettingsOpen}
          settings={settings}
          gitInfo={gitInfo}
          onSave={saveSettings}
        />
      )}

      <AlertDialog
        open={pendingDiscard !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDiscard(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>丢弃更改</AlertDialogTitle>
            <AlertDialogDescription>
              确定丢弃 “{pendingDiscard?.path}” 的所有未提交更改吗？未跟踪文件将被直接删除，此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={discardChange}>
              丢弃
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

type DashboardTreeNode = {
  folderId: number | null
  name: string
  project: ProjectInfoItem | null
  children: DashboardTreeNode[]
}

// buildDashboardTree 按用户自定义目录组织仓库，未归入目录的仓库展示在根级。
function buildDashboardTree(folders: FolderItem[], projects: ProjectInfoItem[]): DashboardTreeNode[] {
  const folderNodes = new Map<number, DashboardTreeNode>()
  for (const folder of folders) {
    folderNodes.set(folder.id, { folderId: folder.id, name: folder.name, project: null, children: [] })
  }
  const tree: DashboardTreeNode[] = []
  for (const folder of folders) {
    const node = folderNodes.get(folder.id)!
    const parent = folder.parentId ? folderNodes.get(folder.parentId) : undefined
    if (parent) parent.children.push(node)
    else tree.push(node)
  }
  for (const project of projects) {
    const node: DashboardTreeNode = { folderId: null, name: project.name, project, children: [] }
    const parent = project.folderId ? folderNodes.get(project.folderId) : undefined
    if (parent) parent.children.push(node)
    else tree.push(node)
  }
  return tree
}

const projectDragKey = 'application/x-gitpower-project'

function Dashboard({
  busy,
  syncing,
  expandedFolders,
  onExpandedFoldersChange,
  recentProjects,
  folders,
  projects,
  onOpen,
  onOpenProject,
  onAddFolder,
  onRenameFolder,
  onDeleteFolder,
  onMoveProject,
  onSync,
  onOpenSettings,
}: {
  busy: boolean
  syncing: boolean
  expandedFolders: Set<string>
  onExpandedFoldersChange: Dispatch<SetStateAction<Set<string>>>
  recentProjects: ProjectInfoItem[]
  folders: FolderItem[]
  projects: ProjectInfoItem[]
  onOpen: () => void
  onOpenProject: (path: string) => void
  onAddFolder: (parentId: number) => Promise<FolderItem | null>
  onRenameFolder: (id: number, name: string) => void
  onDeleteFolder: (id: number) => void
  onMoveProject: (path: string, folderId: number) => void
  onSync: () => void
  onOpenSettings: () => void
}) {
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [summary, setSummary] = useState<ProjectSummaryItem | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)
  const [sideWidth, setSideWidth] = useState(224)
  const [renaming, setRenaming] = useState<{ id: number; value: string } | null>(null)
  const [deletingFolder, setDeletingFolder] = useState<FolderItem | null>(null)
  const [dropTarget, setDropTarget] = useState<number | 'root' | null>(null)
  const tree = useMemo(() => buildDashboardTree(folders, projects), [folders, projects])
  const renamingRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    renamingRef.current?.focus()
    renamingRef.current?.select()
  }, [renaming?.id])

  const folderKey = (id: number) => `folder:${id}`

  // expandFolderChain 展开指定目录及其全部上层目录，保证新建或移入的目录可见。
  const expandFolderChain = (folderId: number) => {
    const parentById = new Map(folders.map((folder) => [folder.id, folder.parentId ?? 0]))
    onExpandedFoldersChange((prev) => {
      const next = new Set(prev)
      let current = folderId
      while (current) {
        next.add(folderKey(current))
        current = parentById.get(current) ?? 0
      }
      return next
    })
  }

  const startAddFolder = async (parentId: number) => {
    const folder = await onAddFolder(parentId)
    if (folder) {
      expandFolderChain(parentId)
      setRenaming({ id: folder.id, value: folder.name })
    }
  }

  const commitRename = () => {
    if (renaming) {
      const name = renaming.value.trim()
      const folder = folders.find((item) => item.id === renaming.id)
      if (name && folder && name !== folder.name) onRenameFolder(renaming.id, name)
    }
    setRenaming(null)
  }

  const dropHandlers = (folderId: number | 'root') => ({
    onDragOver: (event: ReactDragEvent) => {
      if (event.dataTransfer.types.includes(projectDragKey)) {
        event.preventDefault()
        event.stopPropagation()
        event.dataTransfer.dropEffect = 'move'
        setDropTarget(folderId)
      }
    },
    onDragLeave: () => setDropTarget((current) => (current === folderId ? null : current)),
    onDrop: (event: ReactDragEvent) => {
      const path = event.dataTransfer.getData(projectDragKey)
      if (!path) return
      event.preventDefault()
      event.stopPropagation()
      setDropTarget(null)
      if (folderId !== 'root') expandFolderChain(folderId)
      onMoveProject(path, folderId === 'root' ? 0 : folderId)
    },
  })

  const dropHighlight = (folderId: number | 'root') =>
    dropTarget === folderId ? 'rounded-md bg-primary/10 outline-dashed outline-1 outline-primary' : ''

  const startSideDrag = (event: ReactMouseEvent<HTMLDivElement>) => {
    startPaneDrag(event, (deltaX) => {
      setSideWidth(Math.min(480, Math.max(180, Math.round(sideWidth + deltaX))))
    })
  }

  const toggle = (key: string) =>
    onExpandedFoldersChange((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })

  const selectProject = (path: string) => {
    setSelectedPath(path)
    setSummaryLoading(true)
    GetProjectSummary(path)
      .then(setSummary)
      .catch((summaryError) => {
        setSummary(null)
        toastError(errorMessage(summaryError))
      })
      .finally(() => setSummaryLoading(false))
  }

  const projectItemClass = (path: string, extra = '') =>
    `flex w-full items-center gap-2 rounded-md py-1.5 pr-2 text-left text-sm transition-colors disabled:opacity-50 ${
      selectedPath === path
        ? 'bg-primary/10 font-medium text-primary'
        : 'text-foreground hover:bg-muted'
    } ${extra}`

  // rowIndent 目录行按层级缩进；仓库行额外让出展开箭头与间距的宽度，
  // 使根级仓库与根级目录的图标对齐，目录内的仓库比所在目录再深一层。
  const rowIndent = (depth: number, isFolder: boolean) => ({
    paddingLeft: depth * 18 + 8 + (isFolder ? 0 : 18),
  })

  const renderNodes = (nodes: DashboardTreeNode[], depth: number): ReactNode[] =>
    nodes.map((node) => {
      if (node.project) {
        const project = node.project
        return (
          <button
            key={project.path}
            type="button"
            disabled={busy}
            draggable
            onDragStart={(event) => {
              event.dataTransfer.setData(projectDragKey, project.path)
              event.dataTransfer.effectAllowed = 'move'
            }}
            onDragEnd={() => setDropTarget(null)}
            onClick={() => selectProject(project.path)}
            onDoubleClick={() => onOpenProject(project.path)}
            className={projectItemClass(project.path)}
            style={rowIndent(depth, false)}
            title={
              project.missing
                ? `${project.path}\n仓库已不在磁盘上`
                : `${project.path}\n双击打开仓库，可拖拽到目录`
            }
          >
            {project.missing ? (
              <FolderX className="size-3.5 shrink-0 text-muted-foreground" />
            ) : (
              <FolderGit2 className="size-3.5 shrink-0 text-primary" />
            )}
            <span className="min-w-0 flex-1 truncate">{node.name}</span>
          </button>
        )
      }

      const key = folderKey(node.folderId!)
      // 展开状态默认为空，目录树初始全部折叠，点开的目录由 App 记住。
      const isCollapsed = !expandedFolders.has(key)
      const isRenaming = renaming?.id === node.folderId
      return (
        <div key={key}>
          <div
            className={`group/folder flex w-full items-center gap-1 rounded-md py-1.5 pr-1 text-sm text-foreground transition-colors hover:bg-muted ${dropHighlight(node.folderId!)}`}
            style={rowIndent(depth, true)}
            {...dropHandlers(node.folderId!)}
          >
            {isRenaming ? (
              <>
                <Folder className="size-3.5 shrink-0 text-muted-foreground" />
                <Input
                  ref={renamingRef}
                  value={renaming.value}
                  onChange={(event) => setRenaming({ id: node.folderId!, value: event.target.value })}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') commitRename()
                    if (event.key === 'Escape') setRenaming(null)
                  }}
                  onBlur={commitRename}
                  className="h-6 min-w-0 flex-1 px-1 py-0 text-sm"
                />
              </>
            ) : (
              <>
                <button
                  type="button"
                  onClick={() => toggle(key)}
                  className="flex min-w-0 flex-1 items-center gap-1 text-left"
                >
                  {isCollapsed ? (
                    <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" />
                  ) : (
                    <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
                  )}
                  <Folder className="size-3.5 shrink-0 text-muted-foreground" />
                  <span className="min-w-0 flex-1 truncate font-medium">{node.name}</span>
                </button>
                <span className="hidden shrink-0 items-center gap-0.5 group-hover/folder:flex">
                  <button
                    type="button"
                    title="新建子文件夹"
                    className="rounded p-0.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                    onClick={() => startAddFolder(node.folderId!)}
                  >
                    <Plus className="size-3.5" />
                  </button>
                  <button
                    type="button"
                    title="重命名"
                    className="rounded p-0.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                    onClick={() => setRenaming({ id: node.folderId!, value: node.name })}
                  >
                    <Pencil className="size-3" />
                  </button>
                  <button
                    type="button"
                    title="删除文件夹"
                    className="rounded p-0.5 text-muted-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground"
                    onClick={() => setDeletingFolder(folders.find((item) => item.id === node.folderId) ?? null)}
                  >
                    <Trash2 className="size-3" />
                  </button>
                </span>
              </>
            )}
          </div>
          {!isCollapsed && renderNodes(node.children, depth + 1)}
        </div>
      )
    })

  return (
    <div className="flex min-h-0 flex-1 select-none">
      <aside className="flex min-w-44 shrink-0 flex-col" style={{ width: sideWidth }}>
        <div className="flex h-11 shrink-0 items-center gap-2 border-b px-3">
          <Home className="size-4 text-primary" />
          <span className="text-sm font-semibold">GitPower</span>
          <button
            type="button"
            title="设置"
            onClick={onOpenSettings}
            className="ml-auto rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          >
            <Settings className="size-4" />
          </button>
        </div>
        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-4 p-2">
            <div>
              <div className="mb-1 flex items-center gap-1.5 px-2 text-xs font-medium uppercase text-muted-foreground">
                <History className="size-3.5" />
                Recent
              </div>
              <div className="space-y-0.5">
                {recentProjects.length === 0 && (
                  <div className="px-2 py-1 text-xs text-muted-foreground">暂无记录</div>
                )}
                {recentProjects.map((project) => (
                  <button
                    key={project.path}
                    type="button"
                    disabled={busy}
                    onClick={() => selectProject(project.path)}
                    onDoubleClick={() => onOpenProject(project.path)}
                    className={projectItemClass(project.path, 'px-2')}
                    title={`${project.path}\n双击打开仓库`}
                  >
                    <FolderGit2 className="size-3.5 shrink-0 text-primary" />
                    <span className="min-w-0 flex-1 truncate">{project.name}</span>
                    <span className="shrink-0 text-xs text-muted-foreground">
                      {formatRelativeTime(project.lastOpenedAt)}
                    </span>
                  </button>
                ))}
              </div>
            </div>
            <div>
              <div className="mb-1 flex items-center gap-1.5 px-2 text-xs font-medium uppercase text-muted-foreground">
                <FolderGit2 className="size-3.5" />
                Repositories
                <button
                  type="button"
                  title={syncing ? '同步中...' : '同步扫描目录'}
                  disabled={busy || syncing}
                  onClick={onSync}
                  className="ml-auto rounded p-0.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
                >
                  <RefreshCw className={`size-3.5${syncing ? ' animate-spin' : ''}`} />
                </button>
                <button
                  type="button"
                  title="新建文件夹"
                  className="rounded p-0.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                  onClick={() => startAddFolder(0)}
                >
                  <Plus className="size-3.5" />
                </button>
              </div>
              <div
                className={`min-h-6 space-y-0.5 ${dropHighlight('root')}`}
                {...dropHandlers('root')}
              >
                {tree.length === 0 ? (
                  <div className="px-2 py-1 text-xs text-muted-foreground">
                    暂无仓库，点击同步扫描目录或新建文件夹整理
                  </div>
                ) : (
                  renderNodes(tree, 0)
                )}
              </div>
            </div>
          </div>
        </ScrollArea>
      </aside>

      <div
        role="separator"
        aria-orientation="vertical"
        className="relative w-px shrink-0 cursor-col-resize bg-border transition-colors before:absolute before:inset-y-0 before:-left-1 before:w-2 before:content-[''] hover:bg-primary/50 active:bg-primary"
        style={noDragStyle}
        onMouseDown={startSideDrag}
      />

      <main className="flex min-w-0 flex-1 flex-col items-center justify-center gap-4 text-muted-foreground">
        {summaryLoading ? (
          <div className="text-sm">读取仓库信息...</div>
        ) : summary ? (
          <div className="w-full max-w-xl space-y-4 p-8">
            <div className="flex items-center gap-3">
              <FolderGit2 className="size-8 shrink-0 text-primary" />
              <div className="min-w-0">
                <div className="truncate text-lg font-semibold text-foreground">{summary.name}</div>
                <div className="truncate font-mono text-xs" title={summary.path}>{summary.path}</div>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
              <SummaryRow label="当前分支" value={summary.detached ? `分离 HEAD (${summary.currentBranch})` : summary.currentBranch || '无'} />
              <SummaryRow label="远程仓库" value={summary.remotes.join(', ') || '无'} />
              <SummaryRow label="分支" value={`本地 ${summary.localBranches} · 远程 ${summary.remoteBranches}`} />
              <SummaryRow label="提交数" value={String(summary.commits)} />
              <SummaryRow label="未提交更改" value={String(summary.changes)} />
              <SummaryRow label="未推送提交" value={String(summary.aheadCommits)} />
            </div>
            {summary.lastCommit && (
              <div className="space-y-1 rounded-lg border p-3">
                <div className="text-[11px] font-medium uppercase text-muted-foreground">最近一次提交</div>
                <div className="flex items-center gap-2 text-sm">
                  <span className="shrink-0 font-mono text-primary">{summary.lastCommit.shortHash}</span>
                  <Badge variant="secondary" className="h-auto shrink-0 px-1.5 py-0 text-[10px] leading-4">
                    {summary.lastCommit.author}
                  </Badge>
                  <span className="min-w-0 flex-1 truncate font-medium text-foreground">
                    {summary.lastCommit.subject}
                  </span>
                  <Tooltip>
                    <TooltipTrigger
                      render={(props) => (
                        <span {...props} className="shrink-0 text-xs" />
                      )}
                    >
                      {formatRelativeTime(summary.lastCommit.date)}
                    </TooltipTrigger>
                    <TooltipContent>{formatDate(summary.lastCommit.date)}</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            )}
            <div className="flex justify-end">
              <Button onClick={() => onOpenProject(summary.path)} disabled={busy}>
                <FolderGit2 />
                打开仓库
              </Button>
            </div>
          </div>
        ) : (
          <>
            <FolderGit2 className="size-10 text-primary" />
            <div className="text-center">
              <p className="font-medium text-foreground">打开一个 Git 仓库</p>
              <p className="mt-1 text-sm">单击左侧项目查看仓库信息，双击打开仓库，或选择本机上的 Git 仓库目录</p>
            </div>
            <Button onClick={onOpen} disabled={busy}>
              <FolderGit2 />
              打开仓库
            </Button>
          </>
        )}
      </main>

      <AlertDialog
        open={deletingFolder !== null}
        onOpenChange={(open) => {
          if (!open) setDeletingFolder(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除文件夹</AlertDialogTitle>
            <AlertDialogDescription>
              删除「{deletingFolder?.name}」后，其中的仓库与子文件夹会移到上一层。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (deletingFolder) onDeleteFolder(deletingFolder.id)
                setDeletingFolder(null)
              }}
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right font-medium text-foreground">{value}</span>
    </div>
  )
}

// SidebarSection 是侧边栏树的一级节点：折叠头 + 计数徽标。
function SidebarSection({
  title,
  icon,
  count,
  children,
}: {
  title: string
  icon: ReactNode
  count: number
  children: ReactNode
}) {
  const [open, setOpen] = useState(true)

  return (
    <div>
      <button
        type="button"
        aria-expanded={open}
        aria-label={title}
        onClick={() => setOpen((value) => !value)}
        className="flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-left text-xs font-medium text-muted-foreground transition-colors hover:bg-muted"
      >
        {open ? (
          <ChevronDown className="size-3 shrink-0" />
        ) : (
          <ChevronRight className="size-3 shrink-0" />
        )}
        {icon}
        <span className="min-w-0 flex-1 truncate">{title}</span>
        <Badge variant="secondary" className="h-4 px-1.5 text-[10px]">
          {count}
        </Badge>
      </button>
      {open && (
        <div className="mt-0.5 space-y-0.5">
          {count === 0 ? (
            <div className="px-2 py-0.5 text-[11px] text-muted-foreground">无</div>
          ) : (
            children
          )}
        </div>
      )}
    </div>
  )
}

// SidebarRow 是树的叶子行：单击选中，配合 onDoubleClick 执行切换分支。
// indent 默认跳过分组标题的下拉箭头（8 + 12 + 6）后再右移 8，避免与箭头对齐。
function SidebarRow({
  icon,
  label,
  detail,
  badge,
  current,
  selected,
  disabled,
  indent = 34,
  onClick,
  onDoubleClick,
}: {
  icon: ReactNode
  label: string
  detail?: string
  badge?: ReactNode
  current?: boolean
  selected?: boolean
  disabled?: boolean
  indent?: number
  onClick?: () => void
  onDoubleClick?: () => void
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      title={detail}
      style={{ paddingLeft: indent }}
      className={`flex w-full items-center gap-1.5 rounded-md py-1 pr-2 text-left text-xs transition-colors ${
        current
          ? 'bg-primary/10 font-medium text-primary'
          : selected
            ? 'bg-accent font-medium text-accent-foreground'
            : 'text-foreground hover:bg-muted'
      }`}
    >
      {icon}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {badge}
    </button>
  )
}

type RemoteBranchNode = {
  name: string
  path: string
  branch: Branch | null
  children: RemoteBranchNode[]
}

function buildRemoteBranchTree(branches: Branch[]): RemoteBranchNode[] {
  const root: RemoteBranchNode = { name: '', path: '', branch: null, children: [] }
  branches.forEach((branch) => {
    const segments = branch.name.split('/')
    let parent = root
    segments.forEach((segment, index) => {
      const path = segments.slice(0, index + 1).join('/')
      let node = parent.children.find((item) => item.name === segment && !item.branch)
      if (!node) {
        node = { name: segment, path, branch: null, children: [] }
        parent.children.push(node)
      }
      if (index === segments.length - 1) node.branch = branch
      parent = node
    })
  })
  return root.children
}

function countRemoteBranches(node: RemoteBranchNode): number {
  if (node.branch) return 1
  return node.children.reduce((sum, child) => sum + countRemoteBranches(child), 0)
}

// remoteTreeIndent 每层缩进 16px，首层图标从 34 起，与分组标题下的叶子行对齐；
// 目录行让出展开箭头与间距的宽度（12 + 4），使同层目录图标与分支图标对齐。
const remoteTreeIndent = (depth: number, isFolder: boolean) =>
  34 + depth * 16 - (isFolder ? 16 : 0)

// RemoteBranchNodes 渲染 Remotes 节点下的远程分支树，目录可折叠。
function RemoteBranchNodes({
  branches,
  busy,
  selected,
  onSelectItem,
  onOpen,
}: {
  branches: Branch[]
  busy: boolean
  selected: string
  onSelectItem: (ref: string) => void
  onOpen: (branch: Branch) => void
}) {
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set())
  const tree = useMemo(() => buildRemoteBranchTree(branches), [branches])

  const toggle = (path: string) =>
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })

  const renderNodes = (nodes: RemoteBranchNode[], depth: number): ReactNode[] =>
    nodes.map((node) => {
      if (node.branch) {
        const branch = node.branch
        return (
          <SidebarRow
            key={branch.ref}
            icon={<GitBranch className="size-3 shrink-0" />}
            label={node.name}
            detail={branch.upstream ? `上游: ${branch.upstream}` : branch.ref}
            selected={selected === branch.ref}
            disabled={busy}
            indent={remoteTreeIndent(depth, false)}
            onClick={() => onSelectItem(branch.ref)}
            onDoubleClick={() => onOpen(branch)}
          />
        )
      }

      const isCollapsed = collapsed.has(node.path)
      const NodeIcon = node.path.includes('/') ? Folder : Cloud
      return (
        <div key={node.path}>
          <button
            type="button"
            onClick={() => toggle(node.path)}
            style={{ paddingLeft: remoteTreeIndent(depth, true) }}
            className="flex w-full items-center gap-1 rounded-md py-1 pr-2 text-left text-xs text-foreground transition-colors hover:bg-muted"
          >
            {isCollapsed ? (
              <ChevronRight className="size-3 shrink-0 text-muted-foreground" />
            ) : (
              <ChevronDown className="size-3 shrink-0 text-muted-foreground" />
            )}
            <NodeIcon className="size-3 shrink-0 text-muted-foreground" />
            <span className="min-w-0 flex-1 truncate font-medium">{node.name}</span>
            <Badge variant="outline" className="h-3.5 min-w-3.5 justify-center rounded-full px-1 text-[9px]">
              {countRemoteBranches(node)}
            </Badge>
          </button>
          {!isCollapsed && renderNodes(node.children, depth + 1)}
        </div>
      )
    })

  return <>{renderNodes(tree, 0)}</>
}

type CommitFileListItem =
  | { type: 'directory'; directory: string }
  | { type: 'file'; directory: string; file: CommitFileItem }

function CommitFileList({ files }: { files: CommitFileItem[] }) {
  const directories = new Map<string, CommitFileItem[]>()
  files.forEach((file) => {
    const separator = file.path.lastIndexOf('/')
    const directory = separator < 0 ? '' : file.path.slice(0, separator)
    directories.set(directory, [...(directories.get(directory) ?? []), file])
  })

  const listItems: CommitFileListItem[] = []
  directories.forEach((group, directory) => {
    if (directory) listItems.push({ type: 'directory', directory })
    group.forEach((file) => listItems.push({ type: 'file', directory, file }))
  })

  const scrollRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: listItems.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 24,
    overscan: 10,
  })

  return (
    <ScrollArea className="h-full select-none" viewportRef={scrollRef}>
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
                <div className="flex h-6 items-center gap-2 px-3 text-xs font-medium text-muted-foreground">
                  <Folder className="size-3 shrink-0" />
                  <span className="min-w-0 flex-1 truncate">{item.directory}</span>
                </div>
              ) : (
                <div className="px-1">
                  <div
                    className="flex h-6 items-center gap-2 rounded-md px-2"
                    title={item.file.originalPath ? `${item.file.originalPath} → ${item.file.path}` : item.file.path}
                  >
                    <Badge
                      variant="outline"
                      className={`${
                        statusMark(item.file.status)
                          ? 'h-4 w-4 px-0 text-[10px] leading-none'
                          : 'w-16'
                      } ${statusMark(item.file.status)?.className ?? ''} shrink-0 justify-center`}
                    >
                      {statusMark(item.file.status)?.label ?? statusLabel(item.file.status)}
                    </Badge>
                    <span className="min-w-0 flex-1 truncate text-xs">
                      {item.directory ? item.file.path.slice(item.directory.length + 1) : item.file.path}
                    </span>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </ScrollArea>
  )
}

const GRAPH_COLORS = ['#3b82f6', '#f59e0b', '#10b981', '#a855f7', '#ef4444', '#14b8a6', '#f97316', '#8b5cf6']
const GRAPH_LANE_WIDTH = 12
const GRAPH_ROW_HEIGHT = 24

interface GraphLane {
  hash: string
  color: string
}

interface CommitGraphRow {
  lane: number
  color: string
  before: GraphLane[]
  after: GraphLane[]
  parentLanes: number[]
}

// buildCommitGraph 按提交顺序为每条提交分配泳道：首个父提交继承当前泳道，
// 其余父提交向右开辟新泳道；已存在于泳道中的父提交直接复用（合并线汇入）。
function buildCommitGraph(commits: CommitItem[]) {
  const rows = new Map<string, CommitGraphRow>()
  let lanes: GraphLane[] = []
  let colorCursor = 0
  let maxLanes = 1
  const nextColor = () => GRAPH_COLORS[colorCursor++ % GRAPH_COLORS.length]

  for (const commit of commits) {
    const parents = commit.parents ?? []
    const before = lanes
    const matchIndex = before.findIndex((item) => item.hash === commit.hash)

    let lane: number
    let color: string
    let after: GraphLane[]
    if (matchIndex >= 0) {
      lane = matchIndex
      color = before[matchIndex].color
      after = before.filter((_, index) => index !== matchIndex)
    } else {
      lane = before.length
      color = nextColor()
      after = [...before]
    }

    const parentLanes: number[] = []
    let extraInserted = 0
    parents.forEach((parent, index) => {
      const existing = after.findIndex((item) => item.hash === parent)
      if (existing >= 0) {
        parentLanes.push(existing)
        return
      }
      const insertAt = index === 0 ? lane : lane + 1 + extraInserted
      after.splice(insertAt, 0, { hash: parent, color: index === 0 ? color : nextColor() })
      parentLanes.push(insertAt)
      if (index > 0) extraInserted++
    })

    lanes = after
    maxLanes = Math.max(maxLanes, after.length)
    rows.set(commit.hash, { lane, color, before, after, parentLanes })
  }

  return { rows, width: Math.max(maxLanes, 1) * GRAPH_LANE_WIDTH }
}

function CommitGraphCell({ row, width }: { row: CommitGraphRow; width: number }) {
  const height = GRAPH_ROW_HEIGHT
  const mid = height / 2
  const laneX = (index: number) => index * GRAPH_LANE_WIDTH + GRAPH_LANE_WIDTH / 2
  const bezier = (x1: number, y1: number, x2: number, y2: number) =>
    `M ${x1} ${y1} C ${x1} ${(y1 + y2) / 2} ${x2} ${(y1 + y2) / 2} ${x2} ${y2}`

  const segments: { d: string; color: string }[] = []
  row.before.forEach((item, index) => {
    const x1 = laneX(index)
    if (index === row.lane) {
      segments.push({ d: `M ${x1} 0 L ${x1} ${mid}`, color: item.color })
      return
    }
    const target = row.after.findIndex((lane) => lane.hash === item.hash)
    const x2 = target >= 0 ? laneX(target) : x1
    if (x2 === x1) {
      segments.push({ d: `M ${x1} 0 L ${x1} ${height}`, color: item.color })
    } else {
      segments.push({ d: bezier(x1, 0, x2, height), color: item.color })
    }
  })
  row.parentLanes.forEach((laneIndex) => {
    const x1 = laneX(row.lane)
    const x2 = laneX(laneIndex)
    if (x2 === x1) {
      segments.push({ d: `M ${x1} ${mid} L ${x1} ${height}`, color: row.color })
    } else {
      segments.push({ d: bezier(x1, mid, x2, height), color: row.color })
    }
  })

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className="shrink-0 overflow-visible"
    >
      {segments.map((segment, index) => (
        <path key={index} d={segment.d} fill="none" stroke={segment.color} strokeWidth={1.5} />
      ))}
      <circle
        cx={laneX(row.lane)}
        cy={mid}
        r={3}
        fill={row.color}
        stroke="var(--background)"
        strokeWidth={1}
      />
    </svg>
  )
}

function CommitRow({
  commit,
  graph,
  graphWidth,
  selected,
  showDecorations,
  onSelect,
}: {
  commit: CommitItem
  graph?: CommitGraphRow
  graphWidth: number
  selected: boolean
  showDecorations: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`flex h-6 w-full items-center gap-2 px-4 text-left text-[11px] leading-5 transition-colors ${
        selected ? 'bg-[#0064E1] text-white' : 'hover:bg-muted/60'
      }`}
    >
      {graph ? (
        <CommitGraphCell row={graph} width={graphWidth} />
      ) : (
        <span className="shrink-0" style={{ width: graphWidth }} />
      )}
      <span className={`w-14 shrink-0 font-mono ${selected ? 'text-white' : 'text-primary'}`}>
        {commit.shortHash}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5">
            <Badge
              variant="secondary"
              className={`h-auto max-w-32 shrink-0 truncate px-1.5 py-0 text-[10px] leading-4 ${
                selected ? 'border-transparent bg-white/20 !text-white' : ''
              }`}
            >
              {commit.author}
            </Badge>
            {showDecorations && commit.decorations && (
              <span
                className={`shrink-0 truncate ${selected ? 'text-white' : 'text-muted-foreground'}`}
                title={commit.decorations}
              >
                {commit.decorations}
              </span>
            )}
            <span className="truncate font-medium">{commit.subject}</span>
          </div>
          <Tooltip>
            <TooltipTrigger
              render={(props) => (
                <span
                  {...props}
                  className={`shrink-0 ${selected ? 'text-white' : 'text-muted-foreground'}`}
                />
              )}
            >
              {formatRelativeTime(commit.date)}
            </TooltipTrigger>
            <TooltipContent>{formatDate(commit.date)}</TooltipContent>
          </Tooltip>
        </div>
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
  onStageAll,
  onUnstageAll,
  onOpen,
  onCopyPath,
  onIgnore,
  onDiscard,
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
  onStageAll?: () => void
  onUnstageAll?: () => void
  onOpen: (change: Change) => void
  onCopyPath: (change: Change) => void
  onIgnore: (change: Change) => void
  onDiscard: (change: Change) => void
}) {
  const directories = new Map<string, Change[]>()
  changes.forEach((change) => {
    // 未跟踪目录（如嵌套 git 仓库）以 / 结尾，去掉尾斜杠后取父目录，避免文件名显示为空。
    const normalizedPath = change.path.replace(/\/$/, '')
    const separator = normalizedPath.lastIndexOf('/')
    const directory = separator < 0 ? '' : normalizedPath.slice(0, separator)
    directories.set(directory, [...(directories.get(directory) ?? []), change])
  })

  const listItems: ChangeListItem[] = []
  const sortedDirectories = [...directories.keys()].sort((a, b) => {
    if (!a) return 1
    if (!b) return -1
    return a.localeCompare(b)
  })
  sortedDirectories.forEach((directory) => {
    if (directory) listItems.push({ type: 'directory', directory })
    directories.get(directory)!.forEach((change) => listItems.push({ type: 'change', directory, change }))
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
      <div
        className="flex items-center justify-between border-b px-3 py-0.5 text-xs font-medium uppercase text-muted-foreground"
        style={{ backgroundColor: '#FBFBFB', borderColor: '#E6E6E6' }}
      >
        <div className="flex items-center gap-1.5">
          <span>{title}</span>
          <Badge variant="outline" className="text-[10px]">{changes.length}</Badge>
        </div>
        {onStageAll && (
          <Button
            variant="ghost"
            size="icon-xs"
            title="全部暂存"
            disabled={busy || changes.length === 0}
            onClick={onStageAll}
          >
            <Plus />
          </Button>
        )}
        {onUnstageAll && (
          <Button
            variant="ghost"
            size="icon-xs"
            title="全部取消暂存"
            disabled={busy || changes.length === 0}
            onClick={onUnstageAll}
          >
            <Minus />
          </Button>
        )}
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
                    <DirectoryRow
                      key={`dir:${item.directory}`}
                      directory={item.directory}
                      untracked={listItems
                        .filter((entry) => entry.type === 'change' && entry.directory === item.directory)
                        .every((entry) => entry.type === 'change' && entry.change.status.startsWith('??'))}
                      staged={staged}
                      busy={busy}
                      onStageDirectory={() => onStageDirectory(item.directory)}
                      onUnstageDirectory={() => onUnstageDirectory(item.directory)}
                      onOpen={() => onOpen({ path: `${item.directory}/`, originalPath: '', status: '', staged: false, unstaged: true })}
                      onCopyPath={() => onCopyPath({ path: `${item.directory}/`, originalPath: '', status: '', staged: false, unstaged: true })}
                      onIgnore={() => onIgnore({ path: `${item.directory}/`, originalPath: '', status: '', staged: false, unstaged: true })}
                      onDiscard={() => onDiscard({ path: `${item.directory}/`, originalPath: '', status: '??', staged: false, unstaged: true })}
                    />
                  ) : (
                    <div className={item.directory ? 'ml-3 px-1' : 'px-1'}>
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
                        onOpen={() => onOpen(item.change)}
                        onCopyPath={() => onCopyPath(item.change)}
                        onIgnore={() => onIgnore(item.change)}
                        onDiscard={() => onDiscard(item.change)}
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
  onOpen,
  onCopyPath,
  onIgnore,
  onDiscard,
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
  onOpen: () => void
  onCopyPath: () => void
  onIgnore: () => void
  onDiscard: () => void
}) {
  const mark = statusMark(change.status)
  const deleted = change.status.trim().startsWith('D')
  const untracked = change.status.startsWith('??')
  const itemClassName =
    'flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted data-disabled:pointer-events-none data-disabled:opacity-50'

  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger
        onContextMenu={onSelect}
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
      </ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Positioner align="start" sideOffset={2}>
          <ContextMenu.Popup className="z-50 min-w-44 rounded-lg border bg-popover p-1 shadow-lg outline-none">
            <ContextMenu.Item
              className={itemClassName}
              disabled={busy || deleted}
              onClick={onOpen}
            >
              <ExternalLink className="size-3.5 shrink-0 text-muted-foreground" />
              打开
            </ContextMenu.Item>
            <ContextMenu.Item
              className={itemClassName}
              disabled={busy}
              onClick={staged ? onUnstage : onStage}
            >
              {staged ? <Minus className="size-3.5 shrink-0 text-muted-foreground" /> : <Plus className="size-3.5 shrink-0 text-muted-foreground" />}
              {staged ? '取消暂存' : '暂存'}
            </ContextMenu.Item>
            <Separator className="my-1" />
            <ContextMenu.Item
              className={itemClassName}
              disabled={busy}
              onClick={onDiscard}
            >
              <RotateCcw className="size-3.5 shrink-0 text-muted-foreground" />
              丢弃更改...
            </ContextMenu.Item>
            <ContextMenu.Item
              className={itemClassName}
              disabled={busy || !untracked}
              onClick={onIgnore}
              title={untracked ? undefined : '仅未跟踪的文件可以忽略'}
            >
              <EyeOff className="size-3.5 shrink-0 text-muted-foreground" />
              忽略
            </ContextMenu.Item>
            <Separator className="my-1" />
            <ContextMenu.Item className={itemClassName} onClick={onCopyPath}>
              <Copy className="size-3.5 shrink-0 text-muted-foreground" />
              复制路径
            </ContextMenu.Item>
          </ContextMenu.Popup>
        </ContextMenu.Positioner>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  )
}

function DirectoryRow({
  directory,
  untracked,
  staged,
  busy,
  onStageDirectory,
  onUnstageDirectory,
  onOpen,
  onCopyPath,
  onIgnore,
  onDiscard,
}: {
  directory: string
  untracked: boolean
  staged: boolean
  busy: boolean
  onStageDirectory: () => void
  onUnstageDirectory: () => void
  onOpen: () => void
  onCopyPath: () => void
  onIgnore: () => void
  onDiscard: () => void
}) {
  const itemClassName =
    'flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm outline-none data-highlighted:bg-muted data-disabled:pointer-events-none data-disabled:opacity-50'

  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger
        onDoubleClick={() => (staged ? onUnstageDirectory : onStageDirectory)()}
        className="flex h-6 cursor-pointer items-center gap-2 px-3 text-xs font-medium text-muted-foreground"
      >
        <Folder className="size-3 shrink-0" />
        <span className="min-w-0 flex-1 truncate">{directory}</span>
        <Button
          variant="ghost"
          size="icon-xs"
          className="ml-auto"
          onClick={() => (staged ? onUnstageDirectory : onStageDirectory)()}
          disabled={busy}
          title={staged ? '取消暂存目录' : '暂存目录'}
        >
          {staged ? <Minus /> : <Plus />}
        </Button>
      </ContextMenu.Trigger>
      <ContextMenu.Portal>
        <ContextMenu.Positioner align="start" sideOffset={2}>
          <ContextMenu.Popup className="z-50 min-w-44 rounded-lg border bg-popover p-1 shadow-lg outline-none">
            <ContextMenu.Item className={itemClassName} disabled={busy} onClick={onOpen}>
              <ExternalLink className="size-3.5 shrink-0 text-muted-foreground" />
              打开
            </ContextMenu.Item>
            <ContextMenu.Item
              className={itemClassName}
              disabled={busy}
              onClick={staged ? onUnstageDirectory : onStageDirectory}
            >
              {staged ? <Minus className="size-3.5 shrink-0 text-muted-foreground" /> : <Plus className="size-3.5 shrink-0 text-muted-foreground" />}
              {staged ? '取消暂存' : '暂存'}
            </ContextMenu.Item>
            {untracked && (
              <>
                <Separator className="my-1" />
                <ContextMenu.Item className={itemClassName} disabled={busy} onClick={onDiscard}>
                  <RotateCcw className="size-3.5 shrink-0 text-muted-foreground" />
                  丢弃更改...
                </ContextMenu.Item>
                <ContextMenu.Item className={itemClassName} disabled={busy} onClick={onIgnore}>
                  <EyeOff className="size-3.5 shrink-0 text-muted-foreground" />
                  忽略
                </ContextMenu.Item>
              </>
            )}
            <Separator className="my-1" />
            <ContextMenu.Item className={itemClassName} onClick={onCopyPath}>
              <Copy className="size-3.5 shrink-0 text-muted-foreground" />
              复制路径
            </ContextMenu.Item>
          </ContextMenu.Popup>
        </ContextMenu.Positioner>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  )
}

export default App
