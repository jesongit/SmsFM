# ── 前端构建 ──
FROM dkr.pid.im/library/node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── 后端构建（前端嵌入二进制）──
FROM dkr.pid.im/library/golang:1.26-alpine AS server
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web /src/web/dist ./web
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/smsfm-server .

# ── 运行 ──
FROM alpine:3.23
WORKDIR /app
COPY --from=server /out/smsfm-server /app/smsfm-server
# 连接配置持久化目录
VOLUME /app/data
EXPOSE 8080
ENTRYPOINT ["/app/smsfm-server", "-addr", ":8080", "-data", "/app/data"]
