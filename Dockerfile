# 使用官方的 Go 基础镜像
FROM golang:1.22 as builder

# 启用go module
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.io,direct

# 设置工作目录
WORKDIR /app

# 将源代码复制到容器中
COPY . .

# 构建应用程序
RUN cd ./cmd/report && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

# 使用 scratch 作为基础镜像
FROM scratch

# 从构建器中复制构建的应用程序
COPY --from=builder /app/cmd/report/app /app

# 设置运行时命令
ENTRYPOINT ["/app"]
