# AI 创意助手

基于豆包大模型的 AI Agent 创意助手，支持图片生成、视频生成和创意文案写作。

## 功能特性

- **AI 文生图**：根据文字描述生成高质量图片，支持自定义尺寸和风格
- **AI 文生视频**：支持文本生成视频和图片生成视频（图生视频）
- **创意文案写作**：自动撰写自媒体风格文章、脚本、短文案
- **联网搜索**（可选）：集成博查搜索 API，获取实时信息
- **智能 Prompt 优化**：自动将简短描述扩充为高质量提示词
- **多轮对话**：支持上下文记忆的连续对话
- **会话管理**：前端保留 200 条历史消息，会话重置时自动标记

## 技术架构

- **后端**：Go 语言实现的 AI Agent 框架
  - 支持 Function Calling 的 LLM 编排
  - 工具注册与动态调用机制
  - SSE 流式响应
  - 内存会话管理
- **前端**：原生 HTML/CSS/JavaScript
  - 实时消息流展示
  - 图片/视频自动渲染
  - Markdown 格式支持
  - localStorage 持久化存储

## 快速开始

### 本地开发

#### 1. 环境准备

- Go 1.21+
- 豆包 API Key（[获取地址](https://console.volcengine.com/ark)）
- （可选）博查搜索 API Key（用于联网搜索功能）

#### 2. 配置环境变量

复制 `.env.example` 为 `.env` 并配置：

```bash
# 豆包 API 基础 URL
ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3

# Chat 模型（需要支持 function calling）
ARK_CHAT_MODEL=doubao-seed-2-0-pro-260215

# 图片生成模型
ARK_IMAGE_MODEL=doubao-seedream-5-0-260128

# 视频生成模型
ARK_VIDEO_MODEL=doubao-seedance-1-5-pro-251215

# 服务监听配置
HOST=              # 留空监听所有网络接口（0.0.0.0），设置为 127.0.0.1 仅本地访问
PORT=58712         # 服务端口，默认 58712
```

#### 3. 启动服务

```bash
# 加载环境变量并启动
go run main.go
```

服务将在 `http://0.0.0.0:58712` 启动（可通过局域网访问）。

#### 4. 使用

1. 打开浏览器访问 `http://localhost:58712`
2. **在前端页面输入豆包 API Key**（必填）
3. （可选）输入博查搜索 API Key 以启用联网搜索
4. 开始对话，尝试：
   - "画一个赛博朋克风格的未来城市"
   - "生成一个关于春天的短视频"
   - "写一篇关于 AI 技术的自媒体文章"

### 线上部署

#### 方式一：单一可执行文件（推荐）

Go 会自动将 `web/` 目录嵌入到二进制文件中，部署时只需一个文件。

```bash
# 1. 编译
go build -o ai-agent-demo

# 2. 配置环境变量
export ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3
export ARK_CHAT_MODEL=doubao-seed-2-0-pro-260215
export ARK_IMAGE_MODEL=doubao-seedream-5-0-260128
export ARK_VIDEO_MODEL=doubao-seedance-1-5-pro-251215
export HOST=0.0.0.0
export PORT=58712

# 3. 启动
./ai-agent-demo
```

#### 方式二：Docker 部署

创建 `Dockerfile`：

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ai-agent-demo

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ai-agent-demo .
EXPOSE 58712
CMD ["./ai-agent-demo"]
```

构建和运行：

```bash
docker build -t ai-agent-demo .
docker run -d -p 58712:58712 \
  -e ARK_BASE_URL=https://ark.cn-beijing.volces.com/api/v3 \
  -e ARK_CHAT_MODEL=doubao-seed-2-0-pro-260215 \
  -e ARK_IMAGE_MODEL=doubao-seedream-5-0-260128 \
  -e ARK_VIDEO_MODEL=doubao-seedance-1-5-pro-251215 \
  --name ai-agent-demo \
  ai-agent-demo
```

#### 前后端分离部署（可选）

如果需要前后端分离部署，可以使用 Nginx 托管 `web/` 目录，并代理 `/api/` 到后端服务：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    # 前端静态资源
    root /path/to/web;
    index index.html;
    
    # 代理后端 API
    location /api/ {
        proxy_pass http://127.0.0.1:58712;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_buffering off;  # SSE 支持
    }
}
```

## 为什么要在前端输入 API Key？

当前版本采用**前端输入 API Key** 的设计，主要用于：

1. **多人测试场景**：不同用户使用自己的 API Key，独立计费，避免共享额度
2. **安全性**：避免将敏感 API Key 硬编码在服务端或提交到代码仓库
3. **灵活性**：用户可以随时切换不同的 API Key，无需重启服务

> **注意**：`.env.example` 中的 `ARK_API_KEY` 配置项在当前版本中**不起作用**，API Key 必须通过前端页面输入。

**如果你需要提供 SaaS 服务**，建议重新设计架构：
- 实现用户认证系统（JWT/Session）
- 在服务端统一管理 API Key（加密存储）
- 添加使用量统计和限流机制
- 实现计费和配额管理

## 当前版本说明

### 模型使用情况

- **聊天对话**：使用 `ARK_CHAT_MODEL` 配置的模型（默认 `doubao-seed-2-0-pro-260215`）
- **文章写作**：与聊天对话使用**同一个模型**
- **图片生成**：使用 `ARK_IMAGE_MODEL` 配置的模型
- **视频生成**：使用 `ARK_VIDEO_MODEL` 配置的模型

### 会话管理机制

- **前端 localStorage**：
  - 存储最近 200 条消息
  - 刷新页面后立即恢复界面显示
  - 用户可以手动清空（点击"清空历史"按钮）

- **后端内存存储**：
  - 存储当前会话的上下文（用于 LLM 多轮对话）
  - 服务重启后会丢失
  - 通过 `session_id` 关联前后端会话

**工作流程**：
1. 用户发送消息时，前端带上 `session_id`
2. 后端根据 `session_id` 查找历史上下文
3. 后端返回新消息后，前端追加到 localStorage（保留最近 200 条）
4. 如果后端找不到 `session_id`（如服务重启），会创建新会话
5. 前端的历史记录仍然保留，用户可以继续查看

**会话重置提示**：

前端通过检测后端返回的 `session_id` 是否与本地存储的不同来判断：
- 如果 `session_id` 发生变化，说明后端创建了新会话
- 前端会在当前消息位置插入一条分隔线，显示"以上为历史对话"
- 同时弹出提示："会话已重置，上下文已清空"
- 历史消息仍然可见，但后端已经丢失了上下文

### 已知限制

- 聊天和文章写作共用同一个 LLM 模型，无法针对不同场景选择最优模型
- 工具配置中未提供模型列表和特点说明，LLM 无法自主决策使用哪个模型
- 后端会话存储在内存中，服务重启后会丢失（前端历史仍保留）

## 改进建议

以下是用户可以自行改进的方向：

### 1. 多模型支持与智能选择

**目标**：让 LLM 根据任务类型自主选择最优模型。

**实现思路**：
- 在工具注册时提供模型列表和特点描述
- 修改 `write_article` 工具，支持传入 `model` 参数
- 在 System Prompt 中说明各模型的特点（如速度、质量、成本）

**示例配置**：
```go
// 在 api/tools.go 中定义模型列表
var availableModels = map[string]string{
    "doubao-seed-2-0-pro-260215": "高质量通用模型，适合复杂推理和创意写作",
    "doubao-lite-4k-240515": "轻量快速模型，适合简单对话和快速响应",
    "doubao-pro-32k-240515": "长文本模型，适合长篇文章和深度分析",
}
```

### 2. 分离聊天和写作模型

**目标**：为不同场景配置专用模型。

**实现思路**：
- 新增环境变量 `ARK_WRITER_MODEL`
- 修改 `write_article` 工具，使用独立的 LLM Provider
- 针对写作场景优化 System Prompt

### 3. 持久化会话存储

**目标**：支持服务重启后恢复会话历史。

**实现思路**：
- 使用 SQLite/Redis 存储会话数据
- 实现会话过期清理机制
- 支持导出/导入会话历史

### 4. 服务端 API Key 管理

**目标**：适配生产环境的安全需求。

**实现思路**：
- 实现用户认证系统（JWT/Session）
- 在服务端加密存储用户的 API Key
- 添加 API Key 使用量统计和限流

### 5. 工具能力扩展

可以添加更多工具：
- 图片编辑（裁剪、滤镜、风格迁移）
- 语音合成（TTS）
- 文档解析（PDF/Word）
- 数据可视化（图表生成）

## 项目结构

```
.
├── agent/              # AI Agent 核心框架
│   ├── agent.go       # Agent 编排逻辑
│   ├── config.go      # 配置和 System Prompt
│   ├── llm.go         # LLM Provider 接口
│   ├── llm_openai.go  # OpenAI 兼容实现
│   ├── tool.go        # 工具注册表
│   └── message.go     # 消息结构定义
├── api/               # HTTP API 层
│   ├── handler.go     # 路由和会话管理
│   ├── doubao.go      # 豆包 API 调用
│   └── tools.go       # 工具实现（图片/视频/文章）
├── web/               # 前端页面
│   ├── index.html     # 主页面
│   ├── css/           # 样式文件
│   └── js/            # JavaScript 逻辑
├── main.go            # 程序入口
├── .env.example       # 环境变量示例
└── README.md          # 本文档
```

## 常见问题

### Q: 为什么不直接使用百度搜索，而是集成博查 API？

**原因**：
1. **合规性**：直接爬取百度搜索结果违反服务条款，存在法律风险
2. **稳定性**：百度会检测和封禁爬虫，导致服务不稳定
3. **结构化数据**：博查 API 返回结构化的 JSON 数据，易于解析和处理
4. **速度和质量**：专业搜索 API 响应更快，结果质量更高

**如果你想自己实现搜索功能**，可以考虑：

1. **使用其他搜索 API**：
   - Google Custom Search API（需要海外服务器）
   - Bing Search API（微软 Azure）
   - SerpAPI（聚合多个搜索引擎）
   - DuckDuckGo Instant Answer API（免费但功能有限）

2. **实现思路**（以 Bing Search API 为例）：
```go
// 在 api/tools.go 中添加新工具
registry.Register(agent.NewTool(
    "web_search",
    "联网搜索工具，可以搜索互联网获取实时信息",
    `{"type": "object", "properties": {"query": {"type": "string"}}, "required": ["query"]}`,
    func(ctx context.Context, arguments string) (string, error) {
        var args struct{ Query string `json:"query"` }
        json.Unmarshal([]byte(arguments), &args)
        
        // 调用 Bing Search API
        req, _ := http.NewRequest("GET", "https://api.bing.microsoft.com/v7.0/search", nil)
        req.Header.Set("Ocp-Apim-Subscription-Key", "YOUR_BING_API_KEY")
        q := req.URL.Query()
        q.Add("q", args.Query)
        req.URL.RawQuery = q.Encode()
        
        // 处理响应...
        return searchResults, nil
    },
    false,
))
```

### Q: 前端和后端都保存了消息，会不会有冲突？

**不会冲突，这是有意设计的双层存储架构**：

- **前端 localStorage**：
  - 存储用户的完整聊天历史（最近 200 条）
  - 刷新页面后立即恢复界面显示
  - 用户可以手动清空（点击"清空历史"按钮）

- **后端内存存储**：
  - 存储当前会话的上下文（用于 LLM 多轮对话）
  - 服务重启后会丢失
  - 通过 `session_id` 关联前后端会话

**改进建议**：
如果需要更好的同步机制，可以：
1. 实现后端持久化存储（SQLite/Redis），避免重启丢失会话
2. 添加会话过期机制，自动清理长时间未使用的会话
3. 支持会话导出/导入功能

### Q: 图片生成失败，提示尺寸错误？

A: 确保宽 × 高的像素总数不低于 3686400（约 1920×1920）。推荐尺寸：
- 16:9 → 2560×1440
- 1:1 → 1920×1920
- 9:16 → 1440×2560

### Q: 视频生成一直显示"排队中"？

A: 视频生成是异步任务，通常需要 1-3 分钟。工具会自动轮询最多 3 分钟，请耐心等待。

### Q: 如何启用联网搜索功能？

A: 在前端页面的"博查搜索 Key"输入框中填入你的博查 API Key，即可使用 `web_search` 工具。

### Q: 服务重启后会话历史丢失？

A: 当前版本后端使用内存存储，重启会清空后端上下文。但前端 localStorage 中的历史记录（200 条）仍然保留，用户可以继续查看。会话重置时会显示分隔线提示。可参考"改进建议"章节实现后端持久化存储。

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
