# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Vue 3 + TypeScript（前端，开发计划已确认）；后端为 Go，负责扫描、连接保存与协议适配，前端先行，后端接口未就绪期间前端以协议一致的真实数据结构做模拟数据。

## Users

单一管理员（自用场景），在局域网内通过浏览器管理 1–20 台装有 SmsForwarder 的安卓手机。主要在桌面浏览器操作，兼顾手机浏览器查看与基本操作。

## Product Purpose

为局域网内的 SmsForwarder 手机提供网页控制台：扫描/手动连接设备，然后直接查询与操作手机上的短信、通话、联系人、电量、定位等能力，并可对手机做 WOL 唤醒与配置克隆/导出/恢复。业务数据不落库，控制台只保存连接参数与必要凭据。

## Positioning

复用手机上已有的 SmsForwarder 远控 HTTP 接口，不改 APK；管理端是独立的网页控制台，设备列表 + 当前设备操作区 + 连接设置三块，不建统一收件箱、不做屏幕查看/远程桌面。

## Operating Context

- 局域网环境（后端所在或可达的网段），单管理员使用。
- 后端扫描流程：并发快扫（128 并发、350ms TCP 超时）→ 发现即调只读 `/config/query` 识别 → 后台复查（64 并发、1000ms）。
- 快扫 + 配置识别目标 2 秒内（与开发计划 §3.3 一致）。
- 真实手机样例：小米入门机型，App 版本 3.3.2（地址已脱敏）。
- 参考项目：工作区内 `SmsForwarder/` 克隆（只读，不改）。

## Capabilities and Constraints

九项远控能力（按 `/config/query` 返回的功能开关启用入口）：短信查询 `POST /sms/query`、短信发送（选 SIM）`POST /sms/send`、通话记录 `POST /call/query`、联系人查询 `POST /contact/query`、添加联系人 `POST /contact/add`、电量 `POST /battery/query`、定位缓存 `POST /location/query`、WOL `POST /wol/send`、配置克隆/导出/恢复 `POST /clone/pull|push`。连接时 `POST /config/query` 读取设备信息、SIM 信息、版本与功能开关。

协议语义必须保留（详见 docs/development-plan.md §6）：HTTP 200 还要查业务 code；发送成功 ≠ 对方收到，超时显示"结果未知"不自动重发；SIM 索引需转换、无法确认 SIM 时阻止提交；定位只读缓存并显示定位时间；WOL 成功 ≠ 目标已开机；空列表不解释为"无记录"；联系人分页行为由适配层核对；配置恢复会替换设置并清理部分数据，操作前明确展示影响范围；安全模式兼容无安全/签名/RSA/SM4，不夸大签名能力。

明确不做：屏幕查看、点击/远程桌面、统一收件箱、Webhook、短信业务数据库、网页已读状态、分组权限、批量任务、额外管理看板。

连接管理：保存连接、切换设备、编辑地址与凭据、删除连接；连接条目用控制台内部标识，不凭备注自动合并设备。错误需区分连接失败、鉴权失败、功能关闭、业务错误。

## Brand Commitments

产品名 SmsFM。界面语言简体中文。视觉方向：用户明确选定「品类常规」——干净浅色的管理后台，按品类最佳执行，常规被完整拥抱、不加反讽或私货。工艺标杆（用户委托选定）：现代 SaaS 产品（Vercel/Linear 一档）的克制与精致，同时保持运维工具的信息密度。除此之外暂无其他视觉资产或品牌约束。

## Evidence on Hand

- 开发计划：`docs/development-plan.md`（已确认，含真实网络探测数据）。
- 上游接口源码：`SmsForwarder/app/src/main/kotlin/cn/ppps/forwarder/server/`（controller/model，已核对请求与响应结构：BaseResponse{code,msg,data,timestamp,sign}；ConfigData 含 9 个 enable_api_* 开关、extra_device_mark、sim_info_list、version_name；SmsInfo 含 name/number/content/date/type/sim_id；发送参数 sim_slot 1=SIM1、2=SIM2）。
- 尚无真实截图或设计资产；演示数据须为合成并标注。

## Product Principles

1. 设备即焦点：一切操作先明确"当前是哪台手机、哪张 SIM"，界面永远显示当前设备与 SIM。
2. 不撒谎的状态表达：协议的每一层不确定（结果未知、空列表存疑、开关≠可用）都要在界面上如实呈现。
3. 局域网工具的直接性：连接到操作路径要短，扫描反馈要即时可见、可取消。
4. 复用上游语义，不重新发明：错误分类、SIM 转换、恢复影响范围都按上游协议表达。

## Accessibility & Inclusion

桌面优先但需在手机浏览器可用（响应式）；中文界面注意中文字体渲染与行高。无其他既定要求。
