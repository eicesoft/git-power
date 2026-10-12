import {
  useEffect,
  useRef,
  useState,
  type ComponentProps,
  type ReactNode,
} from 'react'
import { Folder, FolderPlus, Plug, X } from 'lucide-react'
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
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { toastError } from '@/components/ui/toast'
import { DiffViewer } from '@/components/diff-viewer'
import { errorMessage } from '@/lib/utils'
import { GetGitGlobalConfig, SelectScanDirectory, SetGitGlobalConfig } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

type Settings = main.Settings
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

const diffPreview = `@@ -1,3 +1,3 @@
 func main() {
-	println("hello")
+	println("hello, Git Power")
 }
`

// DraftInput 维持输入过程中的文本，失焦时才提交，避免编辑中的中间值写入设置。
function DraftInput({ value, onCommit, ...props }: { value: string; onCommit: (value: string) => void } & Omit<ComponentProps<typeof Input>, 'value' | 'onChange' | 'onBlur'>) {
  const [text, setText] = useState(value)
  useEffect(() => setText(value), [value])

  return (
    <Input
      {...props}
      value={text}
      onChange={(event) => setText(event.target.value)}
      onBlur={() => onCommit(text)}
    />
  )
}

// NumberField 同样在失焦时提交，并把越界或非法输入收敛到 [min, max] 区间内。
function NumberField({
  value,
  min,
  max,
  onCommit,
}: {
  value: number
  min: number
  max: number
  onCommit: (value: number) => void
}) {
  const [text, setText] = useState(String(value))
  useEffect(() => setText(String(value)), [value])

  const commit = () => {
    const clamped = Math.min(max, Math.max(min, Number(text) || value))
    setText(String(clamped))
    onCommit(clamped)
  }

  return (
    <Input
      type="number"
      min={min}
      max={max}
      className="w-24"
      value={text}
      onChange={(event) => setText(event.target.value)}
      onBlur={commit}
    />
  )
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
  onChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  settings: Settings
  gitInfo: GitInfo | null
  onChange: (patch: Partial<Settings>) => void
}) {
  const [gitConfig, setGitConfig] = useState<Record<string, string>>({})
  const savedGitConfig = useRef<Record<string, string>>({})

  const loadGitConfig = () => {
    GetGitGlobalConfig()
      .then((entries) => {
        const values = Object.fromEntries(entries.map((entry) => [entry.key, entry.value]))
        savedGitConfig.current = values
        setGitConfig(values)
      })
      .catch((configError) => toastError(errorMessage(configError)))
  }

  useEffect(() => {
    if (open) loadGitConfig()
  }, [open])

  const commitGitConfig = (key: string, value: string) => {
    setGitConfig((current) => ({ ...current, [key]: value }))
    if ((savedGitConfig.current[key] ?? '') === value) return
    savedGitConfig.current = { ...savedGitConfig.current, [key]: value }
    SetGitGlobalConfig(key, value).catch((configError) => {
      toastError(errorMessage(configError))
      // 写入失败时回退为磁盘上的真实配置。
      loadGitConfig()
    })
  }

  const addScanDirectory = async () => {
    try {
      const directory = await SelectScanDirectory()
      if (directory && !settings.scanDirectories.includes(directory)) {
        onChange({ scanDirectories: [...settings.scanDirectories, directory] })
      }
    } catch (selectError) {
      toastError(errorMessage(selectError))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>设置</DialogTitle>
        </DialogHeader>
        <Tabs defaultValue="basic" className="gap-4">
          <TabsList variant="line">
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
                    description="配置默认扫描的磁盘目录，用于自动发现本地仓库。扫描功能尚未实现，此处仅保存配置。"
                  >
                    {settings.scanDirectories.length === 0 ? (
                      <p className="text-xs text-muted-foreground">尚未添加目录</p>
                    ) : (
                      settings.scanDirectories.map((directory) => (
                        <div
                          key={directory}
                          className="flex items-center gap-2 rounded-md border bg-muted/40 px-2 py-1"
                        >
                          <Folder className="size-3.5 shrink-0 text-muted-foreground" />
                          <span className="min-w-0 flex-1 truncate text-xs" title={directory}>
                            {directory}
                          </span>
                          <button
                            type="button"
                            title="移除"
                            className="rounded p-0.5 text-muted-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground"
                            onClick={() =>
                              onChange({
                                scanDirectories: settings.scanDirectories.filter(
                                  (item) => item !== directory,
                                ),
                              })
                            }
                          >
                            <X className="size-3.5" />
                          </button>
                        </div>
                      ))
                    )}
                    <Button variant="outline" size="sm" onClick={addScanDirectory}>
                      <FolderPlus className="size-3.5" />
                      添加目录
                    </Button>
                  </SettingsSection>

                  <SettingsSection title="Diff 视图字体">
                    <SettingsField label="字体">
                      <DraftInput
                        value={settings.diffFontFamily}
                        placeholder="默认等宽字体"
                        onCommit={(value) => onChange({ diffFontFamily: value })}
                      />
                    </SettingsField>
                    <SettingsField label="字号">
                      <NumberField
                        value={settings.diffFontSize}
                        min={8}
                        max={32}
                        onCommit={(value) => onChange({ diffFontSize: value })}
                      />
                    </SettingsField>
                    <div className="overflow-hidden rounded-md border">
                      <DiffViewer
                        diff={diffPreview}
                        fontFamily={settings.diffFontFamily}
                        fontSize={settings.diffFontSize}
                      />
                    </div>
                  </SettingsSection>

                  <SettingsSection title="通用">
                    <SettingsField label="最近仓库条数" hint="Dashboard 左侧 Recent 列表最多显示的仓库数量。">
                      <NumberField
                        value={settings.recentLimit}
                        min={1}
                        max={50}
                        onCommit={(value) => onChange({ recentLimit: value })}
                      />
                    </SettingsField>
                    <label className="flex cursor-pointer items-center gap-2 text-sm">
                      <Checkbox
                        checked={settings.confirmDiscard}
                        onCheckedChange={(checked) => onChange({ confirmDiscard: checked === true })}
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
                      <DraftInput
                        value={settings.gitPath}
                        placeholder="git"
                        onCommit={(value) => onChange({ gitPath: value })}
                      />
                    </SettingsField>
                  </SettingsSection>

                  <SettingsSection
                    title="Global User"
                    description="对应全局配置中的 user.name 与 user.email，提交时记录为作者信息。"
                  >
                    <SettingsField label="用户名" mono>
                      <DraftInput
                        value={gitConfig['user.name'] ?? ''}
                        placeholder="未设置"
                        onCommit={(value) => commitGitConfig('user.name', value)}
                      />
                    </SettingsField>
                    <SettingsField label="邮箱" mono>
                      <DraftInput
                        value={gitConfig['user.email'] ?? ''}
                        placeholder="未设置"
                        onCommit={(value) => commitGitConfig('user.email', value)}
                      />
                    </SettingsField>
                  </SettingsSection>

                  <SettingsSection
                    title="常用配置"
                    description="写入 git config --global，留空表示删除该配置项。"
                  >
                    {gitConfigFields.map((field) => {
                      const value = gitConfig[field.key] ?? ''
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
                                commitGitConfig(field.key, selected === unsetOption ? '' : selected)
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
                            <DraftInput
                              value={value}
                              placeholder="未设置"
                              onCommit={(next) => commitGitConfig(field.key, next)}
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
                      <DraftInput
                        value={settings.agentBaseUrl}
                        placeholder="https://api.example.com/v1"
                        onCommit={(value) => onChange({ agentBaseUrl: value })}
                      />
                    </SettingsField>
                    <SettingsField label="API Key" hint="以明文保存在本地设置文件中。">
                      <DraftInput
                        value={settings.agentApiKey}
                        placeholder="sk-..."
                        onCommit={(value) => onChange({ agentApiKey: value })}
                      />
                    </SettingsField>
                    <SettingsField label="模型名">
                      <DraftInput
                        value={settings.agentModel}
                        placeholder="gpt-4o-mini"
                        onCommit={(value) => onChange({ agentModel: value })}
                      />
                    </SettingsField>
                  </SettingsSection>
                </TabsContent>
              </div>
            </ScrollArea>
          </div>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

export { SettingsDialog }
