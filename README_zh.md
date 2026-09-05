<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**为个人打造的简单、美观、优雅的 LLM API 聚合服务**

简体中文 | [English](README.md)

</div>


## ✨ 特性

- 🔀 **多渠道、多凭据** - 一个渠道管理多份凭据，按“模型 × 凭据”配置支持的协议
- 🔄 **协议互转** - 支持 OpenAI Chat / OpenAI Responses / Anthropic Messages，同协议优先透传
- 💰 **模型价格管理** - 同步参考价格，支持手动定价和重建渠道模型价格
- 🔃 **模型同步与自动入组** - 按凭据同步模型，保留下架授权，支持新模型加入已有同名分组
- 🛡️ **手动选路与故障转移** - 按成员优先级切换，支持全局重试策略、指数退避冷却和备用成员亲和
- 🔍 **实时请求与重试回放** - 展示每轮渠道、凭据、模型和错误，支持中止正在等待的上游调用
- 🚧 **响应提交前重试** - 上游调用失败时保持客户端请求并重试；流式响应开始后不再切换渠道
- 📊 **统计与分享** - 查看请求、Token、费用、首字耗时和速率，导出首页及渠道统计图片
- 🔑 **API Key 管理** - 支持自定义 Key、有效期、费用上限、分组限制和独立用量页面
- 🎨 **Web 管理面板** - 支持浅色/深色主题、简体中文、繁体中文和英文
- 📦 **轻量单文件部署** - 单个二进制文件即可运行，无需额外运行时依赖
- 🗄️ **多数据库支持** - 支持 SQLite、MySQL、PostgreSQL


## 🚀 快速开始

### 🐳 Docker 运行

直接运行

```bash
docker run -d --name octopus -v /path/to/data:/app/data -p 8080:8080 shengmingboai/octopus
```

或者使用 docker compose 运行

```bash
wget https://raw.githubusercontent.com/shengmingboai/octopus/refs/heads/master/docker-compose.yml
docker compose up -d
```


### 📦 从 Release 下载

从 [Releases](https://github.com/shengmingboai/octopus/releases) 下载对应平台的二进制文件，然后运行：

```bash
./octopus start
```

### 🛠️ 源码运行

**环境要求：**

- Go 1.26.4 或更高版本，以 [go.mod](go.mod) 为准
- Node.js 24.11.0 或更高版本，满足当前前端依赖的 `engines` 要求
- pnpm 11.x，CI 使用 11.10.0

```bash
# 克隆项目
git clone https://github.com/shengmingboai/octopus.git
cd octopus
# 构建前端
cd web
pnpm install --frozen-lockfile
pnpm run build
cd ..
# 从项目根目录启动后端服务
go run . start
```

> 💡 **提示**：前端构建产物直接写入 `static/out`，并嵌入 Go 二进制文件。通过后端访问管理页面前，需先构建前端；更新前端后也需重新构建或启动后端。

**开发模式**

在项目根目录打开两个终端：

```bash
# 终端一：后端
go run . start
```

```bash
# 终端二：前端
cd web
pnpm install --frozen-lockfile
pnpm run dev
```

访问 `http://localhost:5173`。Vite 将 `/api` 代理到 `http://127.0.0.1:8080`，LLM 客户端直接连接后端的 `8080` 端口。代理地址、生产资源和目录说明见 [前端开发文档](web/README.md)。

### 🔐 默认账户

首次启动后，访问 http://localhost:8080 使用以下默认账户登录管理面板：

- **用户名**：`admin`
- **密码**：`admin`

> ⚠️ **安全提示**：请在首次登录后立即修改默认密码。

### 📝 配置文件

配置文件默认位于 `data/config.json`，首次启动时自动生成。

可用 `./octopus start --config /path/to/config.json` 指定配置文件；Windows 使用 `.\octopus.exe start --config C:/path/to/config.json`。相对数据路径以启动时的工作目录为基准。

**完整配置示例：**

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "database": {
    "type": "sqlite",
    "path": "data/data.db"
  },
  "log": {
    "level": "info"
  }
}
```

**配置项说明：**

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.host` | 监听地址 | `0.0.0.0` |
| `server.port` | 服务端口 | `8080` |
| `database.type` | 数据库类型 | `sqlite` |
| `database.path` | 数据库连接地址 | `data/data.db` |
| `log.level` | 日志级别 | `info` |

**数据库配置：**

支持三种数据库：

| 类型 | `database.type` | `database.path` 格式 |
|------|-----------------|---------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL 配置示例：**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL 配置示例：**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **提示**：MySQL 和 PostgreSQL 需要先手动创建数据库，程序会自动创建表结构。

**环境变量：**

所有配置项均可通过环境变量覆盖，格式为 `OCTOPUS_` + 配置路径（用 `_` 连接）：

| 环境变量 | 对应配置项 |
|----------|-----------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_GITHUB_PAT` | 用于获取最新版本时的速率限制(可选) |


## 📸 界面预览

### 🖥️ 桌面端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="首页" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="渠道" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="分组" width="400"></td>
</tr>
<tr>
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="价格" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="日志" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="设置" width="400"></td>
</tr>
</table>
</div>

### 📱 移动端

<div align="center">
<table>
<tr>
<td align="center"><b>首页</b></td>
<td align="center"><b>渠道</b></td>
<td align="center"><b>分组</b></td>
<td align="center"><b>价格</b></td>
<td align="center"><b>日志</b></td>
<td align="center"><b>设置</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="移动端首页" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="移动端渠道" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="移动端分组" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="移动端价格" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="移动端日志" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="移动端设置" width="140"></td>
</tr>
</table>
</div>


## 📖 功能说明

### 📡 渠道管理

渠道保存上游地址、协议路径、代理、请求头和参数覆盖配置。同一渠道可以有多份凭据和多个模型：

| 概念 | 用途 |
|------|------|
| 凭据 | 有名称和启停状态的上游 Key，可分别统计用量 |
| 模型 | 上游实际接受的模型名称 |
| 授权 | 一个模型与一份凭据的组合，勾选该组合支持的一个或多个协议 |

分组选择的是授权，因此同一模型使用不同凭据时可以作为不同成员参与选路。停用渠道或凭据后，后续选路不再调用对应目标；删除模型或凭据会连带删除其授权和分组成员。

**地址与协议路径：**

服务商预设用于预填地址和路径，保存后可按上游要求修改。路径留空时使用以下默认值：

| 协议 | 默认路径 | Base URL 示例 | 完整请求地址示例 |
|------|----------|---------------|-----------------|
| OpenAI Chat Completions | `/v1/chat/completions` | `https://api.openai.com` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/v1/responses` | `https://api.openai.com` | `https://api.openai.com/v1/responses` |
| Anthropic Messages | `/v1/messages` | `https://api.anthropic.com` | `https://api.anthropic.com/v1/messages` |

使用默认路径时，Base URL 填服务根地址即可，避免重复包含 `/v1`。有自定义前缀的服务需要同时核对 Base URL 与协议路径。当前转发支持上述三种协议；Gemini 等模型需由上游提供其中一种兼容接口。

授权支持客户端协议时优先透传；否则按 Anthropic Messages、OpenAI Responses、OpenAI Chat Completions 的顺序选择可用协议进行转换。

开启代理后，优先使用渠道专用代理，留空时使用设置页的全局代理。参数覆盖使用 JSON 对象，不能覆盖 `model` 和 `stream`；自定义请求头不能替换转发器已设置的敏感认证头。

**模型同步与自动入组：**

- 拉取模型时按凭据分别访问 OpenAI 和 Anthropic 模型列表并合并结果，可用正则表达式过滤模型名称。
- OpenAI 列表探测结果默认标记为 Responses，Anthropic 列表结果标记为 Messages。列表探测不验证实际生成能力，需要根据上游能力确认协议；只支持 Chat 的目标需手动调整。
- 定时同步和设置页的“立即同步”处理已启用且开启自动同步的渠道，只探测其中启用的凭据。
- 新模型和授权会自动创建；已有授权只更新“上游消失”标记，保留手动设置的协议、分组位置和统计。模型重新出现后恢复原授权。
- 某份凭据探测失败或返回空列表时，保留它的现有授权；所有凭据都没有有效结果时，本渠道同步失败。
- “自动入组”仅针对本次同步新引入的模型，匹配已有分组并追加到成员末尾。匹配忽略大小写，先比较完整模型名，再尝试最后一个 `/` 后的名称；不会自动创建分组，也不会为已有模型补齐分组成员。

---

### 📁 分组管理

**分组名称就是客户端请求中的 `model`**，转发时会替换成所选成员的真实上游模型名。例如，建立 `octopus` 分组并加入多条授权，客户端统一使用 `model: "octopus"`。

| 模式 | 行为 |
|------|------|
| 手动（默认） | 只使用人工指定的成员；新建分组后需要选中一个成员。失败时继续等待并重试，可在界面切换成员 |
| 故障转移 | 按成员排列顺序选择目标，跳过停用的渠道/凭据及冷却中的成员；拖动成员可调整优先级 |

分组页实时显示当前成员、冷却倒计时和亲和状态。没有可选目标时，已进入转发流程的请求会等待配置或目标恢复，并重新选路。

**全局故障转移策略：**

在“设置 → 故障转移策略”中配置，所有故障转移分组共用，后续选路和重试读取最新值。

| 设置 | 默认值 | 含义 |
|------|--------|------|
| 总尝试次数 | 2 次 | 单成员包含首次调用的尝试次数，耗尽后换成员 |
| 重试间隔 | 3 秒 | 同一成员重试及无目标等待的间隔，手动模式也使用此值 |
| 基准冷却 | 30 秒 | 成员耗尽尝试后的初始冷却时间 |
| 最大冷却 | 600 秒 | 连续熔断按 30、60、120……秒指数退避后的上限 |
| 亲和时间 | 300 秒 | 故障切换成功后保持备用成员的时间，设为 0 则不保持 |

冷却到期后，同一分组最多放行一个恢复探测请求；探测成功会清除冷却并重置退避，亲和期间继续使用备用成员。渠道的“失败不冷却”开关只免除跨请求冷却，单次请求仍在耗尽尝试后换成员。总尝试次数、重试间隔和冷却时间至少为 1，亲和时间至少为 0。

“总尝试次数”不是整个客户端请求的总重试上限。Octopus 不设置上游响应等待超时；客户端及反向代理自身的超时仍然有效。需要打断等待时，可在日志中中止当前上游轮次，或由客户端取消整个请求。

只有尚未向客户端发送响应时才能重试或切换。流式响应开始输出后发生的错误会结束当前请求，不能通过更换渠道续接；鉴权失败、无权访问的分组和请求开始时不存在的分组会直接返回错误。

---

### 💰 价格管理

价格页保存用于费用统计的模型价格，包含输入、输出、缓存读取和缓存写入单价，单位为每百万 Token 的美元费用。

- 程序自带参考价格，并定期从 models.dev 更新参考数据。
- 创建、编辑渠道或同步新模型时，会为尚无价格记录的模型匹配参考价格；匹配不到时以零价创建，可在价格页手动填写。
- 价格页包含已有价格记录，并非只显示参考库中没有的模型。转发费用按真实上游模型名（忽略大小写）查询这里的价格。
- 普通价格同步只更新参考数据，不覆盖已保存的模型价格，手动定价因此得以保留。
- “设置 → 模型价格 → 重建价格”会清理不再被渠道引用的模型，并按当前参考价格覆盖剩余模型的价格；匹配不到的价格重置为零。重建会覆盖手动定价，需要先更新参考数据时应先点“立即更新”。

费用是依据返回用量和配置单价计算的统计值。缺少用量或价格会影响费用记录，历史统计不会因修改价格而重新计算。

---

### 🔍 日志与统计

日志页通过 SSE 实时展示请求状态、客户端与上游协议、目标渠道/凭据/模型，以及每轮失败原因。请求结束后也可展开历轮记录，按调用顺序回放重试过程，并查看原始请求和聚合后的响应正文。

- **首字耗时**：流式请求取首个有效事件的等待时间，非流式请求取完整响应的等待时间。
- **总耗时**：包含选路、等待、重试和响应传输。
- **速率**：输出 Token 数除以总耗时，单位 `t/s`，包含重试等待的影响。
- **中止轮次**：仅中止仍在等待响应的当前上游调用，随后重新选路；不会取消整个客户端请求，也不按渠道故障计数。

日志只保存在当前进程内，最多保留最近 50 条已结束请求；运行中的请求另外保留。重启会清空日志，“清空日志”仅移除已结束记录。

首页提供每日活动、今天/近 7 天/近 30 天趋势，以及可按指标排序的渠道/模型榜单。首页与渠道统计详情支持生成 PNG 图片，预览后可复制或下载。总体与 API Key 统计按客户端请求计数，渠道、模型和凭据统计包含各轮上游调用，因此重试时计数可能不同。

---

### 🔑 API Key

在设置页创建供客户端调用的 Octopus API Key，与渠道中保存的上游凭据分开管理：

- Key 值留空时自动生成 `sk-octopus-` 开头的值，也可自定义完整值，不要求固定前缀。
- 可设置启停、到期时间、费用上限和允许访问的分组；未选择分组表示不限制。
- 接口接受 `Authorization: Bearer <key>` 或 `x-api-key: <key>`。
- 在登录页使用 API Key 登录，可查看该 Key 的用量、费用和允许使用的模型，不获得管理员权限。

---

### ⚙️ 设置

基础监听地址和数据库连接放在 `data/config.json`；以下运行设置在 Web 设置页维护并保存到数据库：

| 设置 | 默认值 | 说明 |
|------|--------|------|
| 统计保存周期 | 10 分钟 | 将内存统计批量写入数据库，修改后重启生效 |
| 模型价格更新间隔 | 24 小时 | 更新参考价格，启用时启动即执行一次 |
| 渠道自动同步间隔 | 24 小时 | 同步已启用且开启自动同步的渠道，启用时启动即执行一次 |
| 全局代理地址 | 空 | 供开启代理且未指定专用代理的渠道使用 |
| CORS 跨域白名单 | 空 | 空表示禁止跨域，支持逗号分隔的来源或域名，`*` 表示允许所有来源 |

价格或渠道同步间隔设为 0 会关闭对应定时任务。从 0 改回正数后，需要重启服务重新注册任务；仍可使用手动更新/同步按钮。

> ⚠️ **重要提示**：退出程序时，请使用正常的关闭方式（如 `Ctrl+C` 或发送 `SIGTERM` 信号），以确保内存中的统计数据能正确写入数据库。**请勿使用 `kill -9` 等强制终止方式**，否则可能导致统计数据丢失。

**备份与导入：**

- 设置页可导出 JSON，包含渠道、凭据、授权、分组、API Key、模型价格、运行设置及已经落库的统计。
- 导出不包含管理员账户、`data/config.json`、内存日志和路由冷却/亲和状态；尚未落库的统计也不在导出中。
- 导入采用增量写入，不先清空现有数据库；部分表会更新冲突行，因此导入可能覆盖已有数据。当前导出格式版本为 5，带有其他非零版本号的文件会被拒绝。
- 备份包含上游凭据和客户端 Key 明文，应按密钥文件保管。


## 🔌 客户端接入

先创建渠道、凭据和授权，再建立分组并选定手动成员或启用故障转移，最后在设置页创建 API Key。下文的 Key 均为占位值，`model` 必须填写已建立且该 Key 有权访问的分组名称。

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/v1/models` | 返回当前 Key 可访问的分组名称 |
| POST | `/v1/chat/completions` | OpenAI Chat Completions |
| POST | `/v1/responses` | OpenAI Responses |
| POST | `/v1/messages` | Anthropic Messages |

### OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://127.0.0.1:8080/v1",
    api_key="sk-octopus-REPLACE_WITH_YOUR_KEY",
)
completion = client.chat.completions.create(
    model="octopus",  # 填写实际创建的分组名称
    messages=[
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### Claude Code

编辑 `~/.claude/settings.json`

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8080",
    "ANTHROPIC_AUTH_TOKEN": "sk-octopus-REPLACE_WITH_YOUR_KEY",
    "ANTHROPIC_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "octopus-sonnet-4-5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "octopus-haiku-4-5"
  }
}
```

上述模型值都指 Octopus 分组；可按实际配置替换。Base URL 使用服务根地址，客户端会请求 `/v1/messages`。认证配置参考 [Claude Code 网关接入文档](https://code.claude.com/docs/en/llm-gateway-connect)。

### Codex

在启动 Codex 的环境中设置 `OCTOPUS_API_KEY`：

```bash
export OCTOPUS_API_KEY="sk-octopus-REPLACE_WITH_YOUR_KEY"
```

PowerShell：

```powershell
$env:OCTOPUS_API_KEY = "sk-octopus-REPLACE_WITH_YOUR_KEY"
```

编辑 `~/.codex/config.toml`，`model` 填写实际创建的分组名称：

```toml
model = "octopus"
model_provider = "octopus"

[model_providers.octopus]
base_url = "http://127.0.0.1:8080/v1"
name = "octopus"
env_key = "OCTOPUS_API_KEY"
supports_websockets = false
wire_api = "responses"
```

此配置从环境变量读取 Octopus Key，无需清空或修改已有的 `auth.json`。Octopus 提供 HTTP/SSE Responses 接口，因此关闭 WebSocket。配置字段参考 [Codex 官方配置文档](https://developers.openai.com/codex/config-reference)。


---

## 🤝 致谢

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - 本项目的 LLM API 适配模块直接源自该仓库的实现
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI 模型数据库，提供模型价格数据
- 🇨🇳 [AtomGit](https://atomgit.com/shengmingboai/octopus) - 国内代码托管
- 💬 [Linux.do](https://linux.do/)
