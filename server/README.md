# SmsFM 控制台后端

局域网设备扫描、连接保存与 SmsForwarder（3.3.x）协议适配，前端已嵌入，**单文件运行**。

## 运行

```bash
# 已含前端构建产物（web/ 内嵌）
./smsfm-server.exe                # 默认 http://localhost:8080，数据存 ./data

# 自定义
./smsfm-server.exe -addr :9090 -data D:/smsfm-data
./smsfm-server.exe -web ../web/dist   # 指向前端目录（开发时免重新嵌入）
```

从源码构建：先 `cd web && npm run build`，再 `cd server && go build`（构建脚本见下）。

## 一键重建（前端改动后）

```bash
cd web && npm run build && rm -rf ../server/web && cp -r dist ../server/web && cd ../server && go build -o smsfm-server.exe .
```

## API 概览

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /api/health | 健康检查（前端据此切换 real/mock 模式） |
| GET | /api/networks | 本机 IPv4 /24 网段（默认扫描范围） |
| GET/POST | /api/connections | 连接列表 / 手动连接（先测试后保存） |
| PUT/DELETE | /api/connections/{id} | 编辑（地址/凭据/备注）/ 删除 |
| POST | /api/connections/{id}/probe | 重连测试 + 读配置 |
| POST | /api/scan/start · GET /api/scan/status · POST /api/scan/cancel | 扫描（128 并发 350ms 快扫 → 识别 → 64 并发 1000ms 复查） |
| POST | /api/connections/{id}/{phone-path} | 业务代理：sms/query、sms/send、call/query、contact/query、contact/add、battery/query、location/query、wol/send、clone/pull、clone/push |

错误统一返回 `{error, kind, kindLabel}`，kind ∈ connection / auth / feature-off / business / protocol（HTTP 200 + 业务 code 由后端代查）。

## 协议适配（对齐上游 3.3.x 源码）

- 请求信封 `{data, timestamp, sign?}`；签名 = `HMAC-SHA256("ts\n"+key, key)` → Base64 → URL 编码（有 Node 参照值单测）
- 模式 2 RSA：Base64(信封) → 公钥分段加密（245B 块）；响应私钥 bt1 块 → 公钥解密（Go 手工实现 bt1 unpad）
- 模式 3 SM4：SM4/CBC/PKCS7、固定 IV `03050609060905090305060906090509`、hex 大写（tjfoc/gmsm + 手写 CBC）
- HTTP 200 不代表成功：必须检查业务 `code`；签名/时间戳类错误归「鉴权失败」，非 JSON 归「协议不符」（未验证候选）

## 测试

```bash
go test ./...          # 协议单元/端到端（四模式假手机）+ 扫描器
go run ./testdata/fakephone -addr :15000 -mode 1 -sign-key KEY   # 独立假手机
```

验证边界（开发计划 §7）：以上为模拟测试，验证协议格式与程序行为；RSA/SM4 对真机的互通仍需阶段 4 真机验收。
