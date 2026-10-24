# SmsFM

局域网 SmsForwarder 手机远控控制台：一个二进制（或一个容器）管住局域网里所有装了 [SmsForwarder](https://github.com/pppscn/SmsForwarder) 的安卓手机——扫描发现、连接管理、短信收发、通话与联系人查询、电量/定位/WOL/配置克隆，全部在浏览器完成。**无需修改手机端 APK**。

![Version](https://img.shields.io/badge/version-v1.0.1-blue) ![Go](https://img.shields.io/badge/Go-1.26-00ADD8) ![Vue](https://img.shields.io/badge/Vue-3-42b883) ![License](https://img.shields.io/badge/license-internal-grey)

## 功能

- **设备发现**：并发快扫（128 并发 / 350ms 超时）→ 端口开放即并行调 `/config/query` 识别 → 后台复查超时地址（64 并发 / 1000ms），支持取消；不把 Ping 作为前提
- **连接管理**：手动连接（先测试后保存）、编辑地址/凭据/备注、删除；条目使用内部标识，IP 变化不合并设备
- **协议适配**：上游四种安全模式全支持——无安全措施 / 时间戳签名（HMAC-SHA256）/ RSA（2048 公钥分段加密）/ SM4（CBC·PKCS7·固定 IV）；错误分级：连接失败 / 鉴权失败 / 功能未开启 / 业务错误 / 协议不符
- **九项能力**：短信查询（收/发、关键词、SIM 筛选、翻页、15s 新短信自动检测）、短信发送（选卡、分条计费、超时显示「结果未知」不自动重发）、通话记录（呼入/呼出/未接）、联系人查询与新增、电量、定位缓存（显示定位时间）、WOL 唤醒、配置克隆/导出/恢复（恢复前强制确认影响范围）
- **如实表达协议语义**：HTTP 200 还查业务 code；发送成功 ≠ 对方收到；空列表不解释为无记录；定位只读缓存；WOL 成功 ≠ 已开机
- **单管理员场景**：1–20 台手机，业务数据不落库，只保存连接参数与凭据

## 快速开始

### Docker

容器网络直接影响扫描能力，按环境选择：

**Linux（推荐 host 网络——容器直接使用宿主网络栈，扫描、直连全部原生可用）：**

```bash
docker run -d --name smsfm --network host -v smsfm-data:/app/data posase/smsfm:v1.0.1
# host 模式下无需 -p 映射，服务直接监听宿主 8080
```

**bridge 网络 / Docker Desktop（Windows·macOS）：** 容器内网卡是 docker 内网（172.x），自动枚举拿不到宿主局域网。两种办法任选：

```bash
# 办法一：环境变量显式指定宿主局域网网段（默认扫描范围即为你家/公司网段）
docker run -d --name smsfm -p 8080:8080 -e SMSFM_NETWORK_CIDRS=192.168.1.0/24   -v smsfm-data:/app/data posase/smsfm:v1.0.1

# 办法二：不改启动命令，扫描弹窗里手动把网段改成 192.168.1.0/24（TCP 出站经 NAT 可达局域网，扫描与业务调用均正常）
```

> bridge 模式下手机的 TCP 连接（扫描/业务）都能经 NAT 正常工作；WOL 由手机端发送，不受容器网络影响。

### 二进制

```bash
./smsfm-server -addr :8080 -data ./data   # 默认 8080，连接配置存 ./data
```

启动后点「扫描局域网设备」自动发现（默认端口 5000），或「连接设置 → 手动连接」添加。后端不可用时前端自动进入演示模式（界面标注「演示数据为合成」）。

## 构建

```bash
# 一键（Docker 多阶段：前端构建 → Go 嵌入 → Alpine 运行镜像）
docker build -t smsfm:v1.0.1 .

# 本地
cd web && npm ci && npm run build && rm -rf ../server/web && cp -r dist ../server/web
cd ../server && go build -o smsfm-server .
```

## 架构

```
web/      Vue 3 + TypeScript 前端（Vite；生产构建嵌入 Go 二进制）
server/   Go 后端
  internal/protocol   上游协议适配（信封/签名/RSA/SM4/错误分级）
  internal/scan       局域网扫描（快扫→识别→复查，可取消）
  internal/store      连接持久化（JSON 文件）
  internal/api        REST API + 前端静态托管
  testdata/fakephone  测试用假手机（协议语义对齐上游，验证四种安全模式）
SmsForwarder/         上游参考克隆（只读，不参与构建）
```

前端探测 `/api/health` 自动切换真实/演示模式；业务调用由后端按连接的安全模式与密钥代调手机，前端零协议知识。

## API 概要

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` `/api/networks` | 健康检查 / 本机网段 |
| GET·POST | `/api/connections` | 连接列表 / 手动连接（同地址幂等） |
| PUT·DELETE | `/api/connections/{id}` | 编辑 / 删除 |
| POST | `/api/scan/start`·`/scan/status`·`/scan/cancel` | 扫描控制 |
| POST | `/api/connections/{id}/{sms/query, sms/send, call/query, contact/query, contact/add, battery/query, location/query, wol/send, clone/pull, clone/push}` | 业务代理 |

错误统一 `{error, kind, kindLabel}`。详见 [server/README.md](server/README.md)。

## 测试

```bash
cd server && go test ./...
# 协议层：签名参照值（Node 锚定）、RSA 双向、SM4 往返、四种安全模式端到端（假手机）
# 扫描层：发现/取消/网段展开
```

## 已验证与边界

- 真机验证（小米入门机 · SmsForwarder 3.3.2）：扫描识别、四种数据接口、双卡短信收发（自发自收）、功能开关禁用语义
- RSA / SM4 已按上游源码逐行适配并通过模拟端到端；不同 ROM/版本的行为差异以真机实测为准（`docs/development-plan.md` §7）
- 明确不做：屏幕查看、远程桌面、统一收件箱、Webhook、短信数据库、分组权限、批量任务
