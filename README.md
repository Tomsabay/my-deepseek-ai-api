# AI API 项目

基于 Go + Gin 框架的 AI 聊天 API 服务，支持多轮对话和流式响应。

## 功能特性

- ✅ **单轮对话** - 简单的一问一答
- ✅ **多轮对话** - 支持上下文记忆的连续对话
- ✅ **流式响应 (SSE)** - 像 ChatGPT 一样逐字显示
- ✅ **OpenAI 兼容** - 支持所有兼容 OpenAI 协议的 AI 服务
- ✅ **图片上传** - 支持上传图片和文档附件
- ✅ **多模态 (Vision)** - 支持图片理解（需要使用支持 Vision 的模型）
- ✅ **拖拽上传** - 支持拖拽文件到页面直接上传
- ✅ **用户认证** - JWT Token 认证系统
- ✅ **对话持久化** - SQLite 保存对话历史

## 快速开始

### 1. 配置环境变量

```bash
# 加载配置
source .env

# 或者直接设置
export AI_API_URL="https://api.deepseek.com/v1"
export AI_API_KEY="your-api-key"
export AI_MODEL="deepseek-chat"
```

### 2. 启动服务器

```bash
go run ./cmd/server/main.go
```

### 3. 测试 API

#### 单轮对话
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "你好，请介绍一下你自己"}'
```

#### 多轮对话
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "你好，我叫小明"},
      {"role": "assistant", "content": "你好小明！很高兴认识你！"},
      {"role": "user", "content": "你还记得我叫什么吗？"}
    ]
  }'
```

#### 流式响应（方式1：使用 stream 参数）
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "讲一个小故事", "stream": true}'
```

#### 流式响应（方式2：使用专用接口）
```bash
curl -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "讲一个小故事"}'
```

## API 文档

### POST /api/v1/chat

聊天接口，支持单轮对话、多轮对话和流式响应。

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| message | string | 否* | 用户消息（单轮对话） |
| messages | array | 否* | 对话历史（多轮对话） |
| stream | boolean | 否 | 是否使用流式响应，默认 false |
| model | string | 否 | 指定模型（可选） |

*`message` 和 `messages` 至少提供一个

**响应格式（非流式）：**
```json
{
  "reply": "AI 的回复内容"
}
```

**响应格式（流式，SSE）：**
```
data: {"content":"你"}
data: {"content":"好"}
data: {"content":"！"}
data: [DONE]
```

### POST /api/v1/chat/stream

专用流式聊天接口，总是返回 SSE 流式响应。

## 配置说明

| 环境变量 | 说明 | 默认值 |
|---------|------|-------|
| AI_API_URL | AI API 基础地址 | https://api.openai.com/v1 |
| AI_API_KEY | API 密钥 | (必填) |
| AI_MODEL | 模型名称 | gpt-3.5-turbo |
| AI_SYSTEM_PROMPT | 系统提示词 | (内置默认值) |
| PORT | 服务器端口 | 8080 |

## 支持的 AI 提供商

本项目使用 OpenAI 兼容格式，支持以下提供商：

| 提供商 | Base URL | 模型 |
|--------|----------|------|
| DeepSeek | `https://api.deepseek.com/v1` | `deepseek-chat`, `deepseek-reasoner` |
| OpenAI | `https://api.openai.com/v1` | `gpt-3.5-turbo`, `gpt-4` |
| Moonshot (Kimi) | `https://api.moonshot.cn/v1` | `moonshot-v1-8k` |
| 通义千问 | 需要使用兼容层 | - |

## 项目结构

```
ai-api/
├── cmd/server/main.go       # 入口文件
├── internal/
│   ├── api/handler.go       # HTTP 处理器（路由、请求处理）
│   ├── model/chat.go        # 数据模型（请求/响应结构）
│   └── service/ai_service.go # AI 服务层（业务逻辑）
├── .env                     # 配置文件（不要提交）
├── .env.example             # 配置文件示例
├── .gitignore               # Git 忽略规则
└── README.md                # 项目文档
```

## 前端集成示例

### 使用 Fetch API（流式）

```javascript
async function streamChat(message) {
  const response = await fetch('/api/v1/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message, stream: true })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    const text = decoder.decode(value);
    const lines = text.split('\n');

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6);
        if (data === '[DONE]') return;
        
        const json = JSON.parse(data);
        console.log(json.content); // 输出每个文本片段
      }
    }
  }
}
```

### 使用 EventSource（SSE）

```javascript
function streamChatSSE(message) {
  // 注意：EventSource 只支持 GET，需要用其他方式发送 POST
  // 这里仅作为 SSE 概念演示
  const eventSource = new EventSource('/api/v1/chat/stream?message=' + encodeURIComponent(message));
  
  eventSource.onmessage = (event) => {
    if (event.data === '[DONE]') {
      eventSource.close();
      return;
    }
    const data = JSON.parse(event.data);
    console.log(data.content);
  };

  eventSource.onerror = (error) => {
    console.error('SSE Error:', error);
    eventSource.close();
  };
}
```

## 技术架构

```
┌─────────────────────────────────────────────────────────┐
│                      客户端                              │
│              (浏览器 / curl / 移动端)                     │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP/SSE
┌─────────────────────▼───────────────────────────────────┐
│                   Gin 路由器                             │
│              (Logger, Recovery 中间件)                   │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                  API Handler                             │
│           (请求解析、响应构建、SSE 处理)                  │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                  AI Service                              │
│     (Chat / ChatWithHistory / ChatStream)                │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP/SSE
┌─────────────────────▼───────────────────────────────────┐
│              AI API (DeepSeek/OpenAI)                    │
└─────────────────────────────────────────────────────────┘
```
