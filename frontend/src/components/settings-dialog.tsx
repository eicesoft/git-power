import { useEffect, useState, type ReactNode } from 'react'
import { Folder, FolderOpen, Plug, X } from 'lucide-react'
import { cn } from 'cn'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
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
import { toastError, toastSuccess } from '@/components/ui/toast'
import { DiffViewer } from '@/components/diff-viewer'
import { errorMessage } from '@/lib/utils'
import { GetGitGlobalConfig, SelectScanDirectory, SetGitGlobalConfig } from '../../wailsjs/go/main/App'
import type { config, main } from '../../wailsjs/go/models'

type Settings = config.Settings
type GitInfo = main.GitInfo

// Select 用空字符串表示未设置，这里用一个不会与真实配置值冲突的占位值承载。
const unsetOption = '__unset__'

const gitConfigFields: { key: string; hint: string; options?: string[] }[] = [
  { key: 'core.autocrlf', hint: '换行符转换策略', options: ['true', 'input', 'false'] },
  { key: 'core.quotepath', hint: '是否转义非 ASCII 文件名', options: ['true', 'false'] },
  { key: 'pull.rebase', hint: 'git pull 的默认合并方式', options: ['true', 'false', 'merges'] },
  {
    key: 'push.default',
    hint: 'git push 的默认推送范围',
    options: ['nothing', 'current', 'upstream', 'simple', 'matching'],
  },
  { key: 'init.defaultBranch', hint: 'git init 的默认分支名' },
  { key: 'core.editor', hint: '编辑提交信息使用的编辑器' },
  { key: 'credential.helper', hint: '凭据存储方式，例如 osxkeychain' },
]

// 保存时统一编辑的全局配置项，同时用于脏检查与写入。
const configKeys = ['user.name', 'user.email', ...gitConfigFields.map((field) => field.key)]

const diffPreview = `@@ -1,3 +1,3 @@
 func main() {
-	println("hello")
+	println("hello, Git Power")
 }
`

// clamp 把编辑中的数字收敛到有效区间，非法或清空的输入回落到默认值。
function clamp(value: number, min: number, max: number, fallback: number) {
  if (!Number.isFinite(value) || Math.round(value) <= 0) return fallback
  return Math.min(max, Math.max(min, Math.round(value)))
}

function SettingsSection({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: ReactNode
}) {
  return (
    <section className="rounded-lg border p-4">
      <h3 className="text-sm font-medium">{title}</h3>
      {description && <p className="mt-1 text-xs text-muted-foreground">{description}</p>}
      <div className="mt-3 space-y-3">{children}</div>
    </section>
  )
}

function SettingsField({
  label,
  hint,
  mono,
  children,
}: {
  label: string
  hint?: ReactNode
  mono?: boolean
  children: ReactNode
}) {
  return (
    <div className="grid grid-cols-[8rem_1fr] items-start gap-3">
      <span className={cn('pt-1.5 text-sm text-muted-foreground', mono && 'font-mono text-xs')}>
        {label}
      </span>
      <div className="min-w-0 space-y-1">
        {children}
        {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
      </div>
    </div>
  )
}

function SettingsDialog({
  open,
  onOpenChange,
  settings,
  gitInfo,
  onSave,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  settings: Settings
  gitInfo: GitInfo | null
  onSave: (settings: Settings) => Promise<void>
}) {
  // draft 是编辑中的应用设置，configDraft 是编辑中的全局 Git 配置，
  // configOriginal 是打开时读到的基准，两者一起判断是否还有未保存修改。
  const [draft, setDraft] = useState(settings)
  const [configDraft, setConfigDraft] = useState<Record<string, string>>({})
  const [configOriginal, setConfigOriginal] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [confirming, setConfirming] = useState(false)

  const loadGitConfig = (scope: 'draft' | 'original' | 'both' = 'both') => {
    GetGitGlobalConfig()
      .then((entries) => {
        const values = Object.fromEntries(entries.map((entry) => [entry.key, entry.value]))
        if (scope !== 'original') setConfigDraft(values)
        if (scope !== 'draft') setConfigOriginal(values)
      })
      .catch((configError) => toastError(errorMessage(configError)))
  }

  useEffect(() => {
    if (!open) return
    setDraft(settings)
    loadGitConfig()
  }, [open])

  const update = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setDraft((current) => ({ ...current, [key]: value }))

  const settingsDirty = JSON.stringify(draft) !== JSON.stringify(settings)
  const configDirty = configKeys.some(
    (key) => (configDraft[key] ?? '') !== (configOriginal[key] ?? ''),
  )
  const dirty = settingsDirty || configDirty

  // save 写入应用设置与有变化的全局 Git 配置，全部成功返回 true。
  const save = async (): Promise<boolean> => {
    setSaving(true)
    try {
      const normalized: Settings = {
        ...draft,
        diffFontSize: clamp(draft.diffFontSize, 8, 32, 12),
        recentLimit: clamp(draft.recentLimit, 1, 50, 10),
      }
      await onSave(normalized)
      setDraft(normalized)

      const failures: string[] = []
      for (const key of configKeys) {
        const next = configDraft[key] ?? ''
        if (next === (configOriginal[key] ?? '')) continue
        try {
          await SetGitGlobalConfig(key, next)
        } catch (configError) {
          failures.push(`${key}: ${errorMessage(configError)}`)
        }
      }
      if (failures.length > 0) {
        toastError(failures.join('\n'), '保存失败')
        // 只刷新基准值，保留草稿以便用户重试未写入的配置项。
        loadGitConfig('original')
        return false
      }

      setConfigOriginal(configDraft)
      toastSuccess('设置已保存')
      return true
    } catch (saveError) {
      toastError(errorMessage(saveError))
      return false
    } finally {
      setSaving(false)
    }
  }

  const requestClose = () => {
    if (dirty) setConfirming(true)
    else onOpenChange(false)
  }

  // 关闭按钮、关闭图标、Esc 与遮罩点击都会走到这里，有未保存修改时先确认。
  const handleOpenChange = (next: boolean) => {
    if (next) onOpenChange(true)
    else if (!saving) requestClose()
  }

  const handleSave = () => {
    save().catch(() => {})
  }

  const saveAndClose = () => {
    save()
      .then((saved) => {
        if (saved) onOpenChange(false)
      })
      .catch(() => {})
  }

  const pickScanDirectory = async () => {
    try {
      const directory = await SelectScanDirectory()
      if (directory) update('scanDirectories', [directory])
    } catch (selectError) {
      toastError(errorMessage(selectError))
    }
  }

  const scanDirectory = draft.scanDirectories[0] ?? ''

  return (
    <>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>设置</DialogTitle>
          </DialogHeader>
          <Tabs defaultValue="basic" className="gap-4">
            <TabsList>
              <TabsTrigger value="basic">基础</TabsTrigger>
              <TabsTrigger value="git">Git</TabsTrigger>
              <TabsTrigger value="integration">集成</TabsTrigger>
              <TabsTrigger value="agent">智能体</TabsTrigger>
            </TabsList>
            <div className="h-[26rem] min-w-0">
              <ScrollArea className="h-full">
                <div className="pr-4">
                  <TabsContent value="basic" className="space-y-4">
                    <SettingsSection
                      title="扫描目录"
                      description="配置默认扫描的磁盘目录，用于自动发现本地仓库。在 Dashboard 的 Repositories 点击同步后，会把该目录下的 Git 仓库按原目录结构登记进去。"
                    >
                      <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-2 py-1">
                        <Folder className="size-3.5 shrink-0 text-muted-foreground" />
                        <span
                          className={`min-w-0 flex-1 truncate text-xs${scanDirectory ? '' : ' text-muted-foreground'}`}
                          title={scanDirectory || undefined}
                        >
                          {scanDirectory || '尚未选择目录'}
                        </span>
                        <button
                          type="button"
                          title="选择目录"
                          className="rounded p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
                          onClick={pickScanDirectory}
                        >
                          <FolderOpen className="size-3.5" />
                        </button>
                        {scanDirectory && (
                          <button
                            type="button"
                            title="清除"
                            className="rounded p-1 text-muted-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground"
                            onClick={() => update('scanDirectories', [])}
                          >
                            <X className="size-3.5" />
                          </button>
                        )}
                      </div>
                    </SettingsSection>

                    <SettingsSection title="Diff 视图字体">
                      <SettingsField label="字体">
                        <Input
                          value={draft.diffFontFamily}
                          placeholder="默认等宽字体"
                          onChange={(event) => update('diffFontFamily', event.target.value)}
                        />
                      </SettingsField>
                      <SettingsField label="字号">
                        <Input
                          type="number"
                          min={8}
                          max={32}
                          className="w-24"
                          value={draft.diffFontSize}
                          onChange={(event) => update('diffFontSize', Number(event.target.value))}
                        />
                      </SettingsField>
                      <div className="overflow-hidden rounded-md border">
                        <DiffViewer
                          diff={diffPreview}
                          fontFamily={draft.diffFontFamily}
                          fontSize={draft.diffFontSize}
                        />
                      </div>
                    </SettingsSection>

                    <SettingsSection title="通用">
                      <SettingsField label="最近仓库条数" hint="Dashboard 左侧 Recent 列表最多显示的仓库数量。">
                        <Input
                          type="number"
                          min={1}
                          max={50}
                          className="w-24"
                          value={draft.recentLimit}
                          onChange={(event) => update('recentLimit', Number(event.target.value))}
                        />
                      </SettingsField>
                      <label className="flex cursor-pointer items-center gap-2 text-sm">
                        <Checkbox
                          checked={draft.confirmDiscard}
                          onCheckedChange={(checked) => update('confirmDiscard', checked === true)}
                        />
                        丢弃更改前弹出确认
                      </label>
                    </SettingsSection>
                  </TabsContent>

                  <TabsContent value="git" className="space-y-4">
                    <SettingsSection title="Git 客户端" description="留空则使用系统 PATH 中的 git。">
                      <SettingsField
                        label="可执行文件路径"
                        hint={
                          gitInfo
                            ? `当前使用：${gitInfo.path}（${gitInfo.version}）`
                            : '未能执行当前配置的 Git 客户端'
                        }
                      >
                        <Input
                          value={draft.gitPath}
                          placeholder="git"
                          onChange={(event) => update('gitPath', event.target.value)}
                        />
                      </SettingsField>
                    </SettingsSection>

                    <SettingsSection
                      title="Global User"
                      description="对应全局配置中的 user.name 与 user.email，提交时记录为作者信息。"
                    >
                      <SettingsField label="用户名" mono>
                        <Input
                          value={configDraft['user.name'] ?? ''}
                          placeholder="未设置"
                          onChange={(event) =>
                            setConfigDraft((current) => ({ ...current, 'user.name': event.target.value }))
                          }
                        />
                      </SettingsField>
                      <SettingsField label="邮箱" mono>
                        <Input
                          value={configDraft['user.email'] ?? ''}
                          placeholder="未设置"
                          onChange={(event) =>
                            setConfigDraft((current) => ({ ...current, 'user.email': event.target.value }))
                          }
                        />
                      </SettingsField>
                    </SettingsSection>

                    <SettingsSection
                      title="常用配置"
                      description="写入 git config --global，留空表示删除该配置项。"
                    >
                      {gitConfigFields.map((field) => {
                        const value = configDraft[field.key] ?? ''
                        return (
                          <SettingsField key={field.key} label={field.key} hint={field.hint} mono>
                            {field.options ? (
                              <Select
                                value={value || unsetOption}
                                items={[
                                  { value: unsetOption, label: '未设置' },
                                  ...field.options.map((option) => ({ value: option, label: option })),
                                ]}
                                onValueChange={(next) => {
                                  const selected = String(next)
                                  setConfigDraft((current) => ({
                                    ...current,
                                    [field.key]: selected === unsetOption ? '' : selected,
                                  }))
                                }}
                              >
                                <SelectTrigger className="w-40">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value={unsetOption}>未设置</SelectItem>
                                  {field.options.map((option) => (
                                    <SelectItem key={option} value={option}>
                                      {option}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            ) : (
                              <Input
                                value={value}
                                placeholder="未设置"
                                onChange={(event) =>
                                  setConfigDraft((current) => ({
                                    ...current,
                                    [field.key]: event.target.value,
                                  }))
                                }
                              />
                            )}
                          </SettingsField>
                        )
                      })}
                    </SettingsSection>
                  </TabsContent>

                  <TabsContent value="integration">
                    <div className="flex flex-col items-center gap-2 py-20 text-center">
                      <Plug className="size-6 text-muted-foreground" />
                      <p className="text-sm font-medium">集成</p>
                      <p className="text-xs text-muted-foreground">集成功能尚未开放，敬请期待。</p>
                    </div>
                  </TabsContent>

                  <TabsContent value="agent" className="space-y-4">
                    <SettingsSection
                      title="AI Compatible"
                      description="配置兼容 OpenAI 协议的服务端点，供后续的智能体功能使用。"
                    >
                      <SettingsField label="Base URL">
                        <Input
                          value={draft.agentBaseUrl}
                          placeholder="https://api.example.com/v1"
                          onChange={(event) => update('agentBaseUrl', event.target.value)}
                        />
                      </SettingsField>
                      <SettingsField label="API Key" hint="以明文保存在本地设置文件中。">
                        <Input
                          value={draft.agentApiKey}
                          placeholder="sk-..."
                          onChange={(event) => update('agentApiKey', event.target.value)}
                        />
                      </SettingsField>
                      <SettingsField label="模型名">
                        <Input
                          value={draft.agentModel}
                          placeholder="gpt-4o-mini"
                          onChange={(event) => update('agentModel', event.target.value)}
                        />
                      </SettingsField>
                    </SettingsSection>
                  </TabsContent>
                </div>
              </ScrollArea>
            </div>
          </Tabs>
          <DialogFooter>
            <Button variant="outline" onClick={requestClose} disabled={saving}>
              关闭
            </Button>
            <Button onClick={handleSave} disabled={!dirty || saving}>
              {saving ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>有未保存的修改</AlertDialogTitle>
            <AlertDialogDescription>关闭前要保存对设置所做的修改吗？</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={() => onOpenChange(false)}>
              放弃修改
            </AlertDialogAction>
            <AlertDialogAction onClick={saveAndClose}>保存</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

export { SettingsDialog }
