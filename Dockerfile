# 第一阶段：构建可执行文件
FROM golang:1.24-alpine AS builder

# 安装必要的系统依赖（CGO 针对 SQLite 是必需的）
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 先复制依赖文件并下载，利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制项目源代码
COPY . .

# 编译项目 (启用 CGO 以支持 go-sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server/main.go

# 第二阶段：运行阶段
FROM alpine:latest

# 安装轻量的运行环境依赖
RUN apk add --no-cache tzdata ca-certificates libc6-compat

WORKDIR /app

# 从构建环节复制编译出的可执行文件
COPY --from=builder /app/server .

# 复制用于前端显示的 web 静态文件夹（因为主程序需要解析 ./web/index.html）
COPY --from=builder /app/web ./web

# 创建数据存储目录供 SQLite 和文件上传使用（如果你在 Render 使用磁盘挂载，可以挂载至此目录）
RUN mkdir -p /app/data /app/data/uploads

# 声明使用的端口（Render 默认通过环境变量 PORT 提供）
EXPOSE $PORT

# 启动命令
CMD ["./server"]
