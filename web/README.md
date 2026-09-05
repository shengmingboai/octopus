# Octopus Web

Octopus 的管理前端使用 React 19、TypeScript 6、Vite 8 和 Tailwind CSS 4。TanStack Query 管理 API 数据，Zustand 保存页面与本地偏好，页面按需加载；日志和分组运行状态通过 SSE 更新。

功能与客户端接入说明见 [项目中文文档](../README_zh.md)。

## 环境与启动

- Node.js 24.11.0 或更高版本，满足当前 Babel 等依赖的 `engines` 要求。
- pnpm 11.x，CI 使用 11.10.0。
- 后端需要 Go 1.26.4 或更高版本，以 [go.mod](../go.mod) 为准。

在项目根目录启动后端：

```bash
go run . start
```

另开终端，从项目根目录启动前端：

```bash
cd web
pnpm install --frozen-lockfile
pnpm run dev
```

访问 `http://localhost:5173`。Vite 监听 `0.0.0.0`，将 `/api` 请求（包括日志与分组事件流）代理到 `http://127.0.0.1:8080`。开发服务没有配置 `/v1` 代理，LLM 客户端应直接连接后端。

如需使用其他后端地址，在 `web` 目录启动时设置 `VITE_PROXY_TARGET`：

```bash
VITE_PROXY_TARGET="http://127.0.0.1:9090" pnpm run dev
```

PowerShell：

```powershell
$env:VITE_PROXY_TARGET = "http://127.0.0.1:9090"
pnpm run dev
```

## 构建与检查

在 `web` 目录执行：

| 命令 | 作用 |
|------|------|
| `pnpm run dev` | 启动开发服务与热更新 |
| `pnpm exec tsc --noEmit` | 仅检查 TypeScript 类型 |
| `pnpm run lint` | 运行 ESLint |
| `pnpm run build` | 先检查类型，再执行生产构建 |
| `pnpm run preview` | 启动 Vite 的构建预览服务 |

生产构建直接清空并写入 `../static/out`，无需手动复制。HTML、CSS、JavaScript、JSON 和 SVG 等资源会生成 gzip 文件并移除对应原文件。Go 静态服务会按 `Accept-Encoding` 返回 gzip，或为不支持 gzip 的客户端解压；完整部署验证应使用 Go 后端提供页面。

```bash
pnpm run build
cd ..
go run . start
```

此时访问 `http://localhost:8080`。前端资源通过 `go:embed` 嵌入后端，修改后必须重新构建前端，再重新编译或执行 `go run`。

`static/out/README.md` 是仓库跟踪的占位文件，Vite 清空输出目录时也会删除它。若工作区只因构建出现这项删除，可在根目录执行 `git restore -- static/out/README.md` 恢复。

生产页面加载后注册 Service Worker；开发模式不注册。设置页支持清除 Octopus 浏览器缓存并刷新。开发模式或后端版本为 `dev` 时，不显示前后端版本不匹配及新版本提醒。

## 环境变量

| 变量 | 默认值 | 用途 |
|------|--------|------|
| `VITE_PROXY_TARGET` | `http://127.0.0.1:8080` | 开发服务的 `/api` 代理目标 |
| `DISABLE_HMR` | 未设置 | 设为 `true` 时关闭热更新和文件监听 |
| `VITE_APP_VERSION` | `unknown` | 构建时写入前端版本；发布流程注入与后端一致的版本号 |
| `VITE_GITHUB_REPO` | `https://github.com/shengmingboai/octopus` | 设置页显示的项目仓库链接 |

后端的 `OCTOPUS_*` 配置变量见项目 README，与前端构建变量分开设置。

## 代码导航

| 位置 | 职责 |
|------|------|
| `src/main.tsx` | 主题、语言、QueryClient 等 Provider 和 Service Worker 注册 |
| `src/components/app.tsx` | 管理员/API Key 登录分流、启动数据预取和页面懒加载 |
| `src/components/app-shell.tsx`、`src/stores/app.ts` | 导航、页面切换和布局；固定页面由 Zustand 管理 |
| `src/api/client.ts`、`src/api/queries.ts` | 统一请求、错误处理及共享查询定义 |
| `src/api/channel.ts`、`src/api/group.ts`、`src/api/log.ts` | 渠道/授权类型、分组查询与 SSE、请求日志事件流 |
| `src/components/modules` | 首页、渠道、分组、模型价格、日志、设置及 API Key 用量页面 |
| `src/components/ui`、`src/components/common` | UI 基础组件和跨页面组件 |
| `src/lib/channel-presets.tsx` | 服务商地址和协议路径预设 |
| `src/provider`、`src/locales` | 主题与简体中文、繁体中文、英文文案 |

修改数据结构时对照后端 [数据模型](../internal/model) 和 [HTTP 处理器](../internal/server/handlers)。渠道编辑提交整份配置，未提交的凭据、模型或授权会被删除；分组成员以授权 ID 引用，提交顺序就是优先级。协议位值会落库，必须保持前后端一致。

分组列表与详情共享事件连接，重连后重新拉取数据对齐；日志订阅先接收快照，再接收增量。较大的请求/响应正文按需获取，不能假定事件流会包含正文或跨重启历史。
