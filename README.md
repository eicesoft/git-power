# Git Power

一个使用 Wails v2（Go + React）构建的桌面 Git 客户端。

> **项目状态：后端已实现，前端仍为 UI 原型。**
> Go 侧的真实 Git 能力（仓库扫描、`git status`、提交历史、分支、stage / unstage / commit）已经实现并通过 `options.App.Bind` 导出，`frontend/wailsjs/go/` 已有自动生成的绑定。**但前端尚未接入**——`App.tsx` 未 import 任何绑定，界面上展示的仓库、文件变更、分支数据全部是前端硬编码的 mock 数据。请勿将其当作可用的 Git 客户端。

---

## 当前进度

已完成的部分：

- Wails v2 应用骨架，macOS 窗口 1440x900，使用 `TitleBarHiddenInset`（隐藏标题栏、保留红绿灯）
- 前端三栏布局：顶栏 / 左侧仓库列表 / 主区 Tabs
- 6 个 shadcn 风格基础组件（Tabs、ScrollArea、Badge、Separator、Button、Input），底层为 `@base-ui/react`
- 基于 CSS 变量的主题体系（`oklch` 色彩，light / dark 两套变量均已定义；dark 套已定义但尚未接入切换逻辑，当前固定为 light）
- **Go 侧真实 Git 能力**（`app.go`）：仓库扫描与读取、分支枚举（含远程 / upstream）、提交历史（`git log`）、工作区更改（`git status --porcelain -z`，含重命名原路径）、stage / unstage / commit，以及打开系统目录选择器
- 6 个绑定方法已通过 `options.App.Bind` 注册，`frontend/wailsjs/go/` 生成了对应的 TypeScript 绑定与数据模型
- Go 单元测试（`app_test.go`）：覆盖 `git status` 解析与仓库读写流程

尚未完成的部分：

- **前端未接入任何 wailsjs 绑定**：`App.tsx` 无 `import`，界面上仍是硬编码 mock 数据，真实数据链路尚未打通
- 顶栏的「刷新 / 搜索 / 设置」、左栏的「添加仓库」按钮均只有 UI，无事件处理
- 「变更」页的筛选输入框未接过滤逻辑
- 提交、分支切换等 Go 方法已就绪，但前端没有对应的触发入口
- 仓库目前尚无任何 git commit

---

## 已实现的界面功能

以下均为**纯前端渲染**，数据来源是 `App.tsx` 中的硬编码常量。Go 侧虽已导出真实数据，但前端尚未调用：

| 区域 | 内容 |
| --- | --- |
| 顶栏 | 项目图标、`git-power` 标题、分支徽章 `main`、刷新 / 搜索 / 设置图标按钮 |
| 左栏 | 标题「仓库」、添加按钮、单条硬编码仓库项 `git-power` |
| 主区 Tab「变更」 | 文件筛选输入框、文件计数、3 条硬编码文件（`main.go` M、`App.tsx` M、`package.json` A）并按状态渲染徽章 |
| 主区 Tab「提交记录」 | 分支名 `main` + 空状态「暂无提交记录」 |
| 主区 Tab「分支」 | 分支 `main` + 「当前分支」徽章 |

---

## 技术栈

**后端**

- Go 1.25.0
- [Wails v2](https://wails.io/) v2.14.0（`github.com/wailsapp/wails/v2`）

**前端**

- React 19.3
- TypeScript 5.9
- Vite 8.3
- Tailwind CSS 4.3（通过 `@tailwindcss/vite` 插件接入，无 `tailwind.config.js`，主题在 `index.css` 的 `@theme inline` 中定义）
- shadcn 风格组件，底层 `@base-ui/react` 1.8，配置见 `frontend/components.json`（style: `base-nova`）
- `lucide-react` 1.53 图标
- 系统原生中英文字体栈（SF Pro / Segoe UI / 苹方 / 微软雅黑）

---

## 目录结构

```
.
├── main.go                # Wails 应用入口，1440x900，macOS TitleBarHiddenInset，Bind 注册 app
├── app.go                 # Go 侧真实 Git 能力：仓库读取、分支、提交历史、stage/unstage/commit
├── app_test.go            # Go 单元测试（status 解析 + 仓库读写流程）
├── go.mod / go.sum
├── Makefile               # 构建封装（mac / windows / installer / clean）
├── wails.json             # 项目名 git-power，产物 git-power，devServer localhost:34116
├── bin/                   # 手工 go build 的遗留产物（wails build 不会生成到此处，Makefile 也不重建）
├── build/                 # Wails 构建目录（Info.plist、Info.dev.plist、icons.icns、appicon.png、darwin/、bin/）
└── frontend/
    ├── index.html         # title: Git Power
    ├── package.json       # name 为 wails 模板默认值 frontend，private: true，version 0.0.0
    ├── package-lock.json  # 锁定依赖版本，保证可复现安装
    ├── vite.config.ts     # 路径别名 @ → ./src，server.strictPort: false（端口自动分配）
    ├── tsconfig.json      # 路径别名 @/* → ./src/*
    ├── components.json    # shadcn 配置
    ├── node_modules/      # 前端依赖（未忽略）
    ├── public/            # wails.png（favicon）
    ├── src/
    │   ├── App.tsx        # 唯一页面：三栏布局 + 全部 mock 数据（尚未 import wailsjs 绑定）
    │   ├── main.tsx
    │   ├── App.css        # 模板遗留，当前未被任何文件 import
    │   ├── index.css      # Tailwind 入口 + @theme 主题变量（light / dark 两套）
    │   ├── vite-env.d.ts  # Vite 类型声明
    │   ├── lib/utils.ts   # cn() 工具
    │   ├── components/ui/ # badge / button / input / scroll-area / separator / tabs
    │   └── assets/        # 模板遗留资源
    ├── wailsjs/           # Wails 自动生成的 JS 绑定（go/ 有 6 个方法绑定 + models.ts；runtime/ 为运行时）
    └── dist/              # 前端构建产物，被 main.go 通过 go:embed 嵌入
```

---

## 环境要求

- **Go 1.25.0+**（`go.mod` 声明 `go 1.25.0`）
- **Wails CLI v2**（本仓库不包含其安装记录，需自行安装）
- **Node.js + npm**（用于前端构建；仓库中未声明具体版本要求）
- Windows 交叉编译额外需要 **mingw-w64**
- 打 Windows NSIS 安装包需要 `nsis`（macOS 上可通过 `brew install nsis`）

---

## 快速开始

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 安装前端依赖

```bash
cd frontend
npm install
```

### 开发模式

在项目根目录执行（Wails 会自行启动 Vite dev server 并反向代理；Vite 实际端口由 Vite 自动分配，`wails.json` 中的 `devServer: localhost:34116` 是 Wails 内置 dev server 自身的监听地址）：

```bash
wails dev
```

Makefile 未封装 dev 目标，需手动执行上面的命令。

### 仅跑前端（纯浏览器调试）

```bash
cd frontend
npm run dev
```

> 注意：此模式下没有 Go 后端，且当前代码未使用任何 wailsjs 绑定，界面可以正常渲染。

---

## 构建

### 前端构建

```bash
cd frontend
npm run build
```

等价于 `tsc --noEmit && vite build`，产物输出到 `frontend/dist`。

### 桌面应用构建（通过 Makefile）

产物名统一为 `git-power`。

```bash
make            # 编译 mac + windows
make mac        # 只编译 mac（darwin/arm64）
make windows    # 只编译 windows（windows/amd64）
make installer  # 打 Windows NSIS 安装包（需 brew install nsis）
make clean      # 清理 build/bin
```

| Make 目标 | 命令 | 产物 |
| --- | --- | --- |
| `mac` | `wails build -platform darwin/arm64` | `build/bin/git-power.app` |
| `windows` | `wails build -platform windows/amd64` | `build/bin/git-power.exe` |
| `installer` | `wails build -platform windows/amd64 -nsis` | `build/bin/git-power-amd64-installer.exe` |

可通过变量覆盖目标平台：

```bash
make mac MAC_PLATFORM=darwin/universal
make windows WIN_PLATFORM=windows/386
```

> Makefile 头部注释中出现的 `agent-router.app` / `agent-router.exe` 是历史遗留的文案，与实际产物名（`git-power`）不符；同一注释里的 `*-setup.exe` 同样与实际不符——Wails 内置 NSIS 模板（`pkg/buildassets/build/windows/installer/project.nsi`）的 `OutFile` 为 `${INFO_PROJECTNAME}-${ARCH}-installer.exe`，单 amd64 目标下即 `git-power-amd64-installer.exe`。

### 直接使用 wails CLI

```bash
wails build -platform darwin/arm64
```

### 运行 Go 测试

```bash
go test ./...
```

`app_test.go` 覆盖 `git status --porcelain` 解析（含重命名原路径、暂存 / 未暂存 / 未跟踪三种状态）与仓库读写流程（在 `t.TempDir()` 中真实调用 `git`）。

---

## 架构简述

```
┌─────────────────────────────────────────┐
│  main.go (Go)                           │
│  wails.Run(&options.App{...})           │
│    ├─ //go:embed all:frontend/dist     │
│    │     → assets embed.FS             │
│    └─ Bind: []interface{}{app}          │
│         app.go → 6 个 Git 方法          │
└──────────────┬──────────────────────────┘
               │ assetserver 托管静态资源
               ▼
┌─────────────────────────────────────────┐
│  WebView                                │
│  frontend/dist (React 19 + TS)          │
│    └─ wailsjs 绑定已生成，但前端尚未调用  │
└─────────────────────────────────────────┘
```

- **资源嵌入**：`main.go` 使用 `//go:embed all:frontend/dist` 把前端构建产物编译进二进制，因此 `wails build` 前必须先有 `frontend/dist`（`wails build` 会依据 `wails.json` 自动触发）。
- **前后端桥接**：Wails 依据 `options.App.Bind` 中注册的 struct 自动生成 `frontend/wailsjs/go/` 下的 TypeScript 绑定。**绑定方法本身不需要声明 `context.Context` 参数**（Wails 会把 ctx 当成前端需传入的普通参数，导致调用直接报错）；官方推荐做法是把 `OnStartup` 收到的 ctx 存入 struct 字段，供 `pkg/runtime` 调用——本项目 `app.go` 即 `App{ctx}` + `startup(ctx)`，`SelectRepository()` 内部用 `a.ctx` 调 `runtime.OpenDirectoryDialog`。本项目已注册 6 个方法：`SelectRepository` / `SwitchBranch` / `CreateBranch` / `StageChange` / `UnstageChange` / `Commit`，生成了 `go/main/App.d.ts`、`App.js` 与 `models.ts`。绑定在 `wails dev` / `wails build` 时自动重新生成。
- **窗口配置**：标题 `Git Power`，1440x900，背景色纯白，macOS 使用 `TitleBarHiddenInset`。

---

## 路线图（计划 / 设想）

以下均为**计划中的能力，不代表已实现**。Go 侧的 Git 读取与操作方法（`app.go`）已实现并生成绑定，剩余工作集中在前端接入与工程化：

- **前端接入绑定**（当前最主要缺口）
  - 在 `App.tsx` 中 import `wailsjs/go/main/App`，用真实数据替换现有 mock 常量
  - 接通「添加仓库」（调用 `SelectRepository`）、刷新、分支切换 / 新建、stage / unstage / commit 等交互入口
  - 提交记录页加载真实历史并按分支过滤；变更页接上筛选过滤逻辑
- **仓库管理**：左栏支持多仓库切换与状态刷新
- **顶栏功能**：实现「刷新」「搜索」「设置」三个按钮的行为
- **主题**：`.dark` 变量已定义但无切换逻辑，需补 `classList` 切换并接入「设置」
- **工程化**：补充 `.gitignore`（当前仓库缺失，`build/`、`bin/`、`frontend/node_modules`、`frontend/dist` 均处于未忽略状态）、扩充测试覆盖、CI

---

## License

尚未选择开源协议。仓库中当前不存在 LICENSE 文件。
