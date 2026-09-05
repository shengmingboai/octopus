<div align="center">

<img src="web/public/logo.svg" alt="Octopus Logo" width="120" height="120">

### Octopus

**A Simple, Beautiful, and Elegant LLM API Aggregation Service for Individuals**

 English | [简体中文](README_zh.md)

</div>


## ✨ Features

- 🔀 **Multiple Channels and Credentials** - Manage multiple credentials per channel, with protocols assigned to each model–credential grant
- 🔄 **Protocol Conversion** - OpenAI Chat / OpenAI Responses / Anthropic Messages, with same-protocol passthrough preferred
- 💰 **Model Pricing** - Sync reference prices, set custom prices, and rebuild prices for channel models
- 🔃 **Model Sync and Group Assignment** - Sync models per credential, preserve missing grants, and add new models to existing matching groups
- 🛡️ **Manual Routing and Failover** - Priority-based selection with global retries, exponential cooldown, and fallback affinity
- 🔍 **Live Requests and Retry History** - Inspect each round's channel, credential, model, and error; interrupt upstream calls waiting for a response
- 🚧 **Retries Before Response Delivery** - Keep client requests open while retrying upstream failures; channel switching stops once streaming output begins
- 📊 **Statistics and Sharing** - Track requests, tokens, costs, first-response latency, and throughput; export dashboard and channel statistics as images
- 🔑 **API Key Management** - Custom keys, expiration, spending limits, group restrictions, and a dedicated usage dashboard
- 🎨 **Web Management Panel** - Light/dark themes and Simplified Chinese, Traditional Chinese, and English
- 📦 **Lightweight Single-Binary Deployment** - Run as a single binary with no external runtime dependencies
- 🗄️ **Multi-Database Support** - Support for SQLite, MySQL, PostgreSQL


## 🚀 Quick Start

### 🐳 Docker

Run directly:

```bash
docker run -d --name octopus -v /path/to/data:/app/data -p 8080:8080 shengmingboai/octopus
```

Or use docker compose:

```bash
wget https://raw.githubusercontent.com/shengmingboai/octopus/refs/heads/master/docker-compose.yml
docker compose up -d
```


### 📦 Download from Release

Download the binary for your platform from [Releases](https://github.com/shengmingboai/octopus/releases), then run:

```bash
./octopus start
```

### 🛠️ Build from Source

**Requirements:**

- Go 1.26.4 or later, as specified in [go.mod](go.mod)
- Node.js 24.11.0 or later to satisfy the current frontend dependencies' `engines` requirements
- pnpm 11.x; CI uses 11.10.0

```bash
# Clone the repository
git clone https://github.com/shengmingboai/octopus.git
cd octopus
# Build frontend
cd web
pnpm install --frozen-lockfile
pnpm run build
cd ..
# Start the backend from the repository root
go run . start
```

> 💡 **Tip**: The frontend build writes directly to `static/out`, which is embedded into the Go binary. Build the frontend before serving the management panel through the backend, and rebuild or rerun the backend after changing those assets.

**Development Mode**

Open two terminals at the repository root:

```bash
# Terminal 1: backend
go run . start
```

```bash
# Terminal 2: frontend
cd web
pnpm install --frozen-lockfile
pnpm run dev
```

Visit `http://localhost:5173`. Vite proxies `/api` to `http://127.0.0.1:8080`; LLM clients connect directly to the backend on port `8080`. See the [frontend development guide](web/README.md) for proxy configuration, production assets, and source layout.

### 🔐 Default Credentials

After first launch, visit http://localhost:8080 and log in to the management panel with:

- **Username**: `admin`
- **Password**: `admin`

> ⚠️ **Security Notice**: Please change the default password immediately after first login.

### 📝 Configuration File

The configuration file is located at `data/config.json` by default and is automatically generated on first startup.

Use `./octopus start --config /path/to/config.json` to select another file, or `.\octopus.exe start --config C:/path/to/config.json` on Windows. Relative data paths are resolved from the process's working directory.

**Complete Configuration Example:**

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

**Configuration Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `server.host` | Listen address | `0.0.0.0` |
| `server.port` | Server port | `8080` |
| `database.type` | Database type | `sqlite` |
| `database.path` | Database connection string | `data/data.db` |
| `log.level` | Log level | `info` |

**Database Configuration:**

Three database types are supported:

| Type | `database.type` | `database.path` Format |
|------|-----------------|-----------------------|
| SQLite | `sqlite` | `data/data.db` |
| MySQL | `mysql` | `user:password@tcp(host:port)/dbname` |
| PostgreSQL | `postgres` | `postgresql://user:password@host:port/dbname?sslmode=disable` |

**MySQL Configuration Example:**

```json
{
  "database": {
    "type": "mysql",
    "path": "root:password@tcp(127.0.0.1:3306)/octopus"
  }
}
```

**PostgreSQL Configuration Example:**

```json
{
  "database": {
    "type": "postgres",
    "path": "postgresql://user:password@localhost:5432/octopus?sslmode=disable"
  }
}
```

> 💡 **Tip**: MySQL and PostgreSQL require manual database creation. The application will automatically create the table structure.

### 🌐 Environment Variables

All configuration options can be overridden via environment variables using the format `OCTOPUS_` + configuration path (joined with `_`):

| Environment Variable | Configuration Option |
|---------------------|---------------------|
| `OCTOPUS_SERVER_PORT` | `server.port` |
| `OCTOPUS_SERVER_HOST` | `server.host` |
| `OCTOPUS_DATABASE_TYPE` | `database.type` |
| `OCTOPUS_DATABASE_PATH` | `database.path` |
| `OCTOPUS_LOG_LEVEL` | `log.level` |
| `OCTOPUS_GITHUB_PAT` | For rate limiting when getting the latest version (optional) |

## 📸 Screenshots

### 🖥️ Desktop

<div align="center">
<table>
<tr>
<td align="center"><b>Dashboard</b></td>
<td align="center"><b>Channel Management</b></td>
<td align="center"><b>Group Management</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-home.png" alt="Dashboard" width="400"></td>
<td><img src="web/public/screenshot/desktop-channel.png" alt="Channel" width="400"></td>
<td><img src="web/public/screenshot/desktop-group.png" alt="Group" width="400"></td>
</tr>
<tr>
<td align="center"><b>Price Management</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/desktop-price.png" alt="Price Management" width="400"></td>
<td><img src="web/public/screenshot/desktop-log.png" alt="Logs" width="400"></td>
<td><img src="web/public/screenshot/desktop-setting.png" alt="Settings" width="400"></td>
</tr>
</table>
</div>

### 📱 Mobile

<div align="center">
<table>
<tr>
<td align="center"><b>Home</b></td>
<td align="center"><b>Channel</b></td>
<td align="center"><b>Group</b></td>
<td align="center"><b>Price</b></td>
<td align="center"><b>Logs</b></td>
<td align="center"><b>Settings</b></td>
</tr>
<tr>
<td><img src="web/public/screenshot/mobile-home.png" alt="Mobile Home" width="140"></td>
<td><img src="web/public/screenshot/mobile-channel.png" alt="Mobile Channel" width="140"></td>
<td><img src="web/public/screenshot/mobile-group.png" alt="Mobile Group" width="140"></td>
<td><img src="web/public/screenshot/mobile-price.png" alt="Mobile Price" width="140"></td>
<td><img src="web/public/screenshot/mobile-log.png" alt="Mobile Logs" width="140"></td>
<td><img src="web/public/screenshot/mobile-setting.png" alt="Mobile Settings" width="140"></td>
</tr>
</table>
</div>


## 📖 Documentation

### 📡 Channel Management

A channel stores the upstream address, protocol paths, proxy, headers, and parameter overrides. Each channel can contain multiple credentials and models:

| Concept | Purpose |
|---------|---------|
| Credential | A named upstream key with its own enabled state and usage statistics |
| Model | The actual model name accepted by the upstream service |
| Grant | A model–credential pair with one or more supported protocols selected |

Groups select grants, so the same model with different credentials can be separate routing members. Disabling a channel or credential prevents further calls through that target. Deleting a model or credential also deletes its grants and associated group members.

**Addresses and protocol paths:**

Provider presets prefill addresses and paths, which can be edited for your upstream service. Empty paths fall back to these defaults:

| Protocol | Default Path | Example Base URL | Full Request URL Example |
|----------|--------------|------------------|--------------------------|
| OpenAI Chat Completions | `/v1/chat/completions` | `https://api.openai.com` | `https://api.openai.com/v1/chat/completions` |
| OpenAI Responses | `/v1/responses` | `https://api.openai.com` | `https://api.openai.com/v1/responses` |
| Anthropic Messages | `/v1/messages` | `https://api.anthropic.com` | `https://api.anthropic.com/v1/messages` |

With default paths, use the service root as the Base URL to avoid duplicating `/v1`. For services with custom prefixes, check both the Base URL and protocol paths. Relay currently supports these three protocols; models such as Gemini require an upstream endpoint compatible with one of them.

When a grant supports the client's protocol, passthrough is preferred. Otherwise, the relay selects a supported protocol in this order: Anthropic Messages, OpenAI Responses, OpenAI Chat Completions, and converts the request and response.

When proxying is enabled, a channel-specific proxy takes precedence; leaving it empty uses the global proxy from Settings. Parameter overrides are a JSON object and cannot replace `model` or `stream`. Custom headers cannot replace sensitive authentication headers already set by the relay transformer.

**Model synchronization and automatic group assignment:**

- Model discovery fetches OpenAI and Anthropic model lists per credential and merges them. A regular expression can filter model names.
- OpenAI list results are initially marked as Responses; Anthropic results are marked as Messages. Discovery does not test generation endpoints. Confirm the selected protocols against your upstream's capabilities and manually adjust targets that only support Chat.
- Scheduled sync and the Settings page's manual sync process enabled channels with automatic model synchronization turned on, using their enabled credentials.
- New models and grants are created automatically. Existing grants only have their missing-upstream flag updated; configured protocols, group positions, and statistics are preserved. Reappearing models restore their original grants.
- Failed or empty discovery results leave that credential's existing grants unchanged. A channel sync fails if none of its credentials produces a valid result.
- Automatic group assignment applies only to models newly introduced by that sync, appending members to existing matching groups. Matching is case-insensitive: the full name is checked first, then the suffix after the last `/`. It neither creates groups nor backfills group members for existing models.

---

### 📁 Group Management

**The group name is the client's `model` value.** The relay replaces it with the selected member's actual upstream model name. For example, create an `octopus` group containing several grants, then call it with `model: "octopus"`.

| Mode | Behavior |
|------|----------|
| Manual (default) | Uses only the member you select. Select a member after creating the group. Failures wait and retry; you can switch the member in the UI |
| Failover | Selects members in list order, skipping disabled channels/credentials and members in cooldown. Drag members to change their priority |

The group page displays the current member, cooldown countdowns, and affinity state live. When no target is available, requests already in the relay loop wait for configuration changes or recovery and select again.

**Global failover policy:**

Configure this in Settings → Failover Policy. All failover groups share these values, and subsequent routing and retry decisions read the latest settings.

| Setting | Default | Meaning |
|---------|---------|---------|
| Total attempts | 2 | Attempts per member, including the first call, before switching members |
| Retry interval | 3 seconds | Delay between retries on the same member and while waiting for a target; also used in manual mode |
| Base cooldown | 30 seconds | Initial cooldown after a member exhausts its attempts |
| Maximum cooldown | 600 seconds | Cap for exponential backoff: 30, 60, 120… seconds with the default base |
| Affinity | 300 seconds | Time to keep a fallback member after a successful switch; 0 disables affinity |

After cooldown expires, each group allows at most one recovery probe at a time. A successful probe clears cooldown and resets backoff; during affinity, requests stay on the fallback member. A channel's “No cooldown on failure” option only bypasses cooldown across requests: each request still switches members when attempts are exhausted. Attempts, retry interval, and cooldown values must be at least 1; affinity must be at least 0.

“Total attempts” is not a retry limit for the entire client request. Octopus does not impose an upstream response-wait timeout; client and reverse-proxy timeouts still apply. Interrupt a waiting upstream round from the log page, or cancel the whole request from the client.

Retries and switching are possible only before a response is sent to the client. Errors after streaming output begins end the request; another channel cannot continue that stream. Authentication failures, disallowed groups, and groups missing when the request starts return errors directly.

---

### 💰 Price Management

The pricing page stores the model prices used for cost statistics: input, output, cache reads, and cache writes, in USD per million tokens.

- The application includes reference prices and periodically refreshes them from models.dev.
- Creating or editing channels, or syncing new models, initializes missing price records from matching reference data. Unmatched models start at zero and can be priced manually.
- The page contains saved price records, including models found in the reference data. Relay costs use these prices, looked up by the actual upstream model name without regard to case.
- A regular price sync refreshes reference data only. It does not overwrite saved prices, preserving manual pricing.
- Settings → Model Pricing → Rebuild Prices removes models no longer referenced by any channel and replaces remaining prices with current reference values. Unmatched prices become zero. This overwrites manual pricing; use the update button first if you want fresh reference data.

Costs are statistics calculated from returned usage and configured prices. Missing usage or prices affects those records. Changing prices does not recalculate historical statistics.

---

### 🔍 Logs and Statistics

The log page uses SSE to display request state, client and upstream protocols, target channel/credential/model, and each round's errors live. Completed requests retain an expandable history in call order, alongside the original request and aggregated response bodies.

- **First-response latency**: Time to the first valid event for streaming requests, or to the complete response for non-streaming requests.
- **Total duration**: Includes routing, waiting, retries, and response delivery.
- **Throughput**: Output tokens divided by total duration, in `t/s`, including time spent waiting for retries.
- **Interrupt round**: Cancels only the current upstream call while it is waiting for a response, then selects a target again. It neither cancels the client request nor counts as a channel failure.

Logs are kept only in the current process, with up to 50 finished requests retained in addition to active requests. Restarting clears them. Clearing logs removes finished records only.

The dashboard provides daily activity, trends for today/the last 7 days/the last 30 days, and channel/model rankings sortable by metric. Dashboard and channel statistics can be previewed as PNG images, then copied or downloaded. Overall and API Key statistics count client requests, while channel, model, and credential statistics include upstream rounds, so counts can differ when retries occur.

---

### 🔑 API Keys

Create Octopus API Keys in Settings for client access. These are managed separately from upstream credentials stored in channels:

- Leave the key value empty to generate a `sk-octopus-` key, or supply a custom full value with no required prefix.
- Configure enabled state, expiration, a spending limit, and allowed groups. An empty group selection means unrestricted access.
- API requests accept `Authorization: Bearer <key>` or `x-api-key: <key>`.
- Signing in with an API Key opens a dashboard for that key's usage, costs, and allowed models without granting administrator access.

---

### ⚙️ Settings

The listening address and database connection belong in `data/config.json`. The following runtime settings are managed in the Web panel and stored in the database:

| Setting | Default | Description |
|---------|---------|-------------|
| Statistics save interval | 10 minutes | Batch-save in-memory statistics; restart after changing this interval |
| Model price update interval | 24 hours | Refresh reference prices; also runs at startup when enabled |
| Channel model sync interval | 24 hours | Sync enabled channels with automatic sync turned on; also runs at startup when enabled |
| Global proxy URL | Empty | Used by proxy-enabled channels without a channel-specific proxy |
| CORS allowlist | Empty | Empty blocks cross-origin requests; accepts comma-separated origins or domains, or `*` for all origins |

Setting the price or channel sync interval to 0 disables that scheduled task. After changing it from 0 to a positive value, restart the service to register the task again. Manual update/sync buttons remain available.

> ⚠️ **Important**: When exiting the program, use proper shutdown methods (like `Ctrl+C` or sending `SIGTERM` signal) to ensure in-memory statistics are correctly written to the database. **Do NOT use `kill -9` or other forced termination methods**, as this may result in statistics data loss.

**Backup and import:**

- Settings can export JSON containing channels, credentials, grants, groups, API Keys, model prices, runtime settings, and statistics already saved to the database.
- Exports do not include the administrator account, `data/config.json`, in-memory logs, or routing cooldown/affinity state. Unsaved statistics are also excluded.
- Import writes incrementally without clearing the database first. Some tables update conflicting rows, so existing data can be overwritten. The current export format version is 5; other nonzero version numbers are rejected.
- Backups contain upstream credentials and client keys in plain text and should be stored as secret files.

---

## 🔌 Client Integration

Create a channel, credentials, and grants; create a group and select a manual member or enable failover; then create an API Key in Settings. Keys below are placeholders. Every `model` value must be an existing group that the key is allowed to access.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/v1/models` | List group names accessible to the current key |
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
    model="octopus",  # Use a group you have created
    messages=[
        {"role": "user", "content": "Hello"},
    ],
)
print(completion.choices[0].message.content)
```

### Claude Code

Edit `~/.claude/settings.json`

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

These model values are Octopus group names; replace them to match your configuration. Use the service root as the Base URL; the client calls `/v1/messages`. See the [Claude Code gateway connection guide](https://code.claude.com/docs/en/llm-gateway-connect) for authentication settings.

### Codex

Set `OCTOPUS_API_KEY` in the environment where you launch Codex:

```bash
export OCTOPUS_API_KEY="sk-octopus-REPLACE_WITH_YOUR_KEY"
```

PowerShell:

```powershell
$env:OCTOPUS_API_KEY = "sk-octopus-REPLACE_WITH_YOUR_KEY"
```

Edit `~/.codex/config.toml`, using a group you have created as the `model`:

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

This configuration reads the Octopus key from the environment; there is no need to clear or edit an existing `auth.json`. WebSockets are disabled because Octopus serves Responses over HTTP/SSE. See the [official Codex configuration reference](https://developers.openai.com/codex/config-reference) for these fields.

---

## 🤝 Acknowledgments

- 🙏 [looplj/axonhub](https://github.com/looplj/axonhub) - The LLM API adaptation module in this project is directly derived from this repository
- 📊 [sst/models.dev](https://github.com/sst/models.dev) - AI model database providing model pricing data
- 🇨🇳 [AtomGit](https://atomgit.com/shengmingboai/octopus) - China-based code hosting
- 💬 [Linux.do](https://linux.do/)
