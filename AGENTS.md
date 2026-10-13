# 仓库指南

Git Power 是基于 Wails v2 的桌面 Git 客户端。Go 后端通过 Wails 生成的绑定向 React 前端提供仓库操作。

## 项目结构与模块组织

- 根目录 Go 文件：`main.go` 负责启动 Wails；`app.go` 定义 `App` 绑定对象与设置、窗口、命令日志；`app_repository.go`、`app_dashboard.go`、`files.go` 是其余导出方法的薄转发层。
- Go 内部包：`internal/git` 是 Git 领域层（类型、命令执行、解析、全部仓库操作）；`internal/store` 用 SQLite 持久化仓库与目录；`internal/config` 负责设置读写与数据目录路径；`internal/scan` 扫描目录同步仓库；`internal/testutil` 提供跨包共用的测试辅助函数。
- Go 测试：`app_test.go` 及各包的 `*_test.go` 使用临时仓库编写单元测试和流程测试。
- 前端源码：主要界面位于 `frontend/src/App.tsx`；可复用控件位于 `frontend/src/components/ui/`；主题定义位于 `frontend/src/index.css`。
- 生成绑定：`frontend/wailsjs/go/` 包含由导出 Go 方法生成的 TypeScript 绑定，禁止手动编辑。
- 项目配置：`wails.json` 配置应用和开发服务器；`Makefile` 定义桌面构建目标。

## 构建、测试与开发命令

```bash
wails dev
go test ./...
npm --prefix frontend run build
make
```

- `wails dev` 启动桌面应用，启用前端热更新，并刷新生成绑定。
- `go test ./...` 运行全部 Go 测试。
- `npm --prefix frontend run build` 对前端进行类型检查和构建。
- `make` 构建 macOS 和 Windows 可执行文件。使用 `make mac`、`make windows` 或 `make clean` 执行单项任务。

## 编码风格与命名约定

- 遵循 `frontend/src` 中的 TypeScript 和 React 约定：两空格缩进、单引号，并使用已配置的 `@/` 导入别名。
- 组件尽可能使用用shadcn 组件。按钮、输入框、对话框、选项卡、徽章等界面组件禁止自行实现。
- 遵循 Wails 约定：导出 Go 方法使用 PascalCase，JSON 字段使用 camelCase。
- Go 测试函数命名为 `Test<行为>`，例如 `TestParseStatus`。提交前运行 `go test ./...`。
- 前端未配置测试运行器或 lint 工具；前端验收命令为 `npm --prefix frontend run build`。

## 提交与 Pull Request 规范

- Commit message 使用中文，格式为 `<类型>: <描述>`，例如 `修复: 分支切换状态未刷新`。
- Pull request 保持聚焦。说明问题和解决方案，列出验证命令，并关联相关 issue。

## 测试与验收

-  非必要无需使用浏览器或者Computer use类功能进行测试验收
