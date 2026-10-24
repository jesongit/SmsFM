---
name: SmsFM 控制台 (SmsFM Console)
description: 品类常规 light admin console for LAN SmsForwarder phone fleets — white cards on mist, one blue action color, machine data in mono.
colors:
  page-bg: "#f8fafc"
  card: "#ffffff"
  ink: "#111827"
  body-ink: "#374151"
  text-dim: "#6b7280"
  text-faint: "#9ca3af"
  line: "#e5e7eb"
  line-soft: "#f1f3f5"
  hover-fill: "#f3f4f6"
  table-head: "#fafbfc"
  dot-offline: "#d1d5db"
  tab-off: "#c3c8d2"
  blue: "#2563eb"
  blue-soft: "#eff6ff"
  green: "#16a34a"
  green-soft: "#dcfce7"
  green-border: "#bbf7d0"
  amber: "#d97706"
  amber-soft: "#fffbeb"
  amber-border: "#fde68a"
  amber-ink: "#92400e"
  red: "#dc2626"
  red-soft: "#fef2f2"
  red-border: "#fecaca"
  red-ink: "#7f1d1d"
  mask: "rgba(15, 23, 42, 0.32)"
typography:
  stat:
    fontFamily: "\"Segoe UI\", \"Microsoft YaHei\", \"PingFang SC\", sans-serif"
    fontSize: "40px"
    fontWeight: 800
    lineHeight: 1.1
  headline:
    fontFamily: "\"Segoe UI\", \"Microsoft YaHei\", \"PingFang SC\", sans-serif"
    fontSize: "17px"
    fontWeight: 700
    lineHeight: 1.5
  title:
    fontFamily: "\"Segoe UI\", \"Microsoft YaHei\", \"PingFang SC\", sans-serif"
    fontSize: "15px"
    fontWeight: 700
    lineHeight: 1.5
  body:
    fontFamily: "\"Segoe UI\", \"Microsoft YaHei\", \"PingFang SC\", sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.5
  control:
    fontFamily: "\"Segoe UI\", \"Microsoft YaHei\", \"PingFang SC\", sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "\"Segoe UI\", \"Microsoft YaHei\", \"PingFang SC\", sans-serif"
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "2px"
  data:
    fontFamily: "Consolas, \"SF Mono\", \"Courier New\", monospace"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  tag: "4px"
  sm: "6px"
  control: "8px"
  card: "10px"
  pill: "12px"
  overlay: "14px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  gutter-desktop: "26px"
  gutter-mobile: "14px"
components:
  button-primary:
    backgroundColor: "{colors.blue}"
    textColor: "{colors.card}"
    typography: "600 13px/1.5"
    rounded: "{rounded.control}"
    padding: "8px 16px"
  button-secondary:
    backgroundColor: "{colors.card}"
    textColor: "{colors.ink}"
    typography: "400 13px/1.5"
    rounded: "{rounded.control}"
    padding: "8px 16px"
  input-text:
    backgroundColor: "{colors.card}"
    textColor: "{colors.ink}"
    typography: "400 13px/1.5"
    rounded: "{rounded.control}"
    padding: "8px 12px"
  status-badge-online:
    backgroundColor: "{colors.green-soft}"
    textColor: "{colors.green}"
    typography: "600 11.5px/1.5"
    rounded: "{rounded.pill}"
    padding: "3px 10px"
  status-badge-pending:
    backgroundColor: "{colors.amber-soft}"
    textColor: "{colors.amber}"
    typography: "600 11.5px/1.5"
    rounded: "{rounded.pill}"
    padding: "3px 10px"
  chip-sim:
    backgroundColor: "{colors.blue-soft}"
    textColor: "{colors.blue}"
    typography: "400 10.5px/1.5"
    rounded: "{rounded.tag}"
    padding: "2px 8px"
  sidebar-nav-item:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "600 13.5px/1.5"
    rounded: "{rounded.control}"
    padding: "9px 12px"
  table-card:
    backgroundColor: "{colors.card}"
    rounded: "{rounded.card}"
    padding: "0"
  stat-card:
    backgroundColor: "{colors.card}"
    rounded: "{rounded.card}"
    padding: "16px 20px"
  error-banner:
    backgroundColor: "{colors.red-soft}"
    textColor: "{colors.red-ink}"
    typography: "400 13px/1.5"
    rounded: "{rounded.control}"
    padding: "10px 14px"
---

# Design System: SmsFM 控制台

## Overview

**Creative North Star: "The Lit-Up Machine Room"（把真实过程亮出来）**

SmsFM 控制台 is a canon light admin console — the category convention fully embraced, no costume. White cards sit on a faint blue-tinted mist (`#f8fafc`), separated by 1px hairlines instead of shadows, with one saturated blue (`#2563eb`) reserved for action and current selection. The craft bar is modern SaaS restraint (Vercel/Linear grade) carrying ops-tool information density: every panel leads with the current device and SIM, and every query result lands in a dense, mono-annotated data table.

The system's signature is honesty made visible. State is never implied by decoration alone — it is named: a green pill says 已连接, a grayed tab renames itself 「定位（未开启）」, an empty table says the empty list may be a permission problem rather than "no records", and overlays narrate their own process (快扫中 · 128 并发 · 350ms 连接超时). Reliability comes from lighting up the real process, not from gloss.

**Key Characteristics:**
- Flat white cards bounded by 1px `#e5e7eb` rules; shadows exist only on overlays (drawer, modal).
- One blue for action/selection; green/amber/red report device reality and never decorate.
- Machine-produced values (IPs, phone numbers, versions, coordinates) set in Consolas mono.
- System Chinese sans stack only — zero serif, zero webfonts.
- Radius ladder 4/6/8/10/12/14px with 8px as the control default; dashed 1px borders mark explainer blocks.
- Disabled capabilities stay visible, grayed, and renamed with the reason.

**Documented deviation from the approved comp** (`.impeccable/mocks/comp-canon.png`): the device header carries a 「演示数据为合成」 dashed tag that the comp does not show. PRODUCT.md requires the synthetic-data label ("演示数据须为合成并标注"); the comp predates it. This is intentional and stays.

## Colors

A cool near-neutral gray family on a blue-tinted page, one action blue, and three semantic state colors — nothing else.

### Primary
- **Console Blue** (#2563eb): the only saturated accent. Primary buttons (查询 / 写短信 / 扫描局域网设备 / 发送), active tab text + 2px underline, active device row tint, selected segment fill, progress-bar fill, brand logo tile, text-selection highlight, and the SIM tag. Its soft tint **Mist Blue** (#eff6ff) backgrounds selected/identified states.

### Status Colors
- **Signal Green** (#16a34a, soft #dcfce7, border #bbf7d0): online state dots, 已连接 badge, success banners (已提交手机发送), battery bar, scan results tagged 配置识别成功. Always means "the device/service is actually working".
- **Caution Amber** (#d97706, soft #fffbeb, border #fde68a, deep text #92400e): pending credentials (待凭据), re-check candidates (待复查), blocked-SIM warnings, and the "result unknown" banner. Always means "uncertain or blocked, read the explanation".
- **Alarm Red** (#dc2626, soft #fef2f2, border #fecaca, deep text #7f1d1d): error banners (kind label + message) and the low-battery bar (<20%). Only ever reports a failure.

### Neutral
- **Page Mist** (#f8fafc): app background; also modal/drawer body where content rows are white cards.
- **Card White** (#ffffff): every resting surface — sidebar, header, tabs, cards, table, controls. Also the text color on blue.
- **Ink** (#111827): headings, device names, primary text, primary data values.
- **Body Slate** (#374151): SMS content and long-form table text.
- **Dim** (#6b7280): secondary text — table headers, IPs in the header line, hints, pager, battery %.
- **Faint** (#9ca3af): tertiary text — footnotes, meta lines, field labels, offline badges, close icons.
- **Rule Line** (#e5e7eb): every border — cards, controls, dividers between regions.
- **Soft Rule** (#f1f3f5): row dividers inside tables and progress-bar track.
- Micro-neutrals: **Hover Fill** #f3f4f6 (sidebar row hover), **Table Head** #fafbfc, **Offline Dot** #d1d5db, **Disabled Tab** #c3c8d2, **Overlay Mask** rgba(15, 23, 42, 0.32).

### Named Rules
**The One-Blue Rule.** #2563eb marks primary action and current selection only. State colors report device reality; neither blue nor any state color is used decoratively. If a screen needs a second accent, the screen is wrong.

**The Soft-Tint Pair Rule.** Every signal color ships as a pair — the strong color for text/dot/fill and its soft tint for the background (green on #dcfce7, blue on #eff6ff, amber on #fffbeb, red on #fef2f2). Banners add the mid-tone border (#bbf7d0, #fde68a, #fecaca). Never put a strong signal color on white as a fill.

## Typography

**Body Font:** System Chinese sans — "Segoe UI", "Microsoft YaHei", "PingFang SC", sans-serif (no webfonts, no serif anywhere)
**Label/Mono Font:** Consolas, "SF Mono", "Courier New", monospace

**Character:** A utilitarian system stack tuned for Simplified Chinese UI text at 14px/1.5, with machine output set apart in mono so network-produced values read as data, not prose.

### Hierarchy
- **Stat** (800, 40px, 1.1): the battery hero number only, with an 18px/600 unit suffix.
- **Headline** (700, 17px): brand wordmark (800 in the sidebar) and current device name.
- **Title** (700, 15–16px): drawer and modal titles (15px drawers, 16px modal).
- **Body** (400, 14px, 1.5): base text; table cells 13.5px with content in #374151.
- **Control** (400/600, 13px): buttons, inputs, segments, tab row neighbors (tabs at 14px).
- **Small** (400, 10.5–12.5px): field labels (11–12px), table headers (12px/600), hints (12.5px), badges (11.5px/600), meta and footnotes (10.5–11px). 10.5px is the floor.
- **Data** (mono, 10.5–13.5px): IP:port, phone numbers, app versions, coordinates, device-select options, scan hosts. Weight 600 when the number is the row's identity.

### Named Rules
**The Mono-Is-Machine Rule.** Anything the phone or the network produced — IPs, ports, phone numbers, versions, coordinates, percentages — is set in Consolas mono. Human language stays in the sans stack. If you're typing an IP in sans, it's a bug.

## Layout

Desktop-first master-detail at 1440×900. A fixed 264px white sidebar (`--sidebar-w`, 1px right border) holds brand, the 扫描局域网设备 primary button, the device list, and a footer line; the fluid main column holds a sticky 64px device header (z-5), a sticky tab row pinned at `top: 64px` (z-4), then panel content. The main-area horizontal gutter is 26px (tools `padding: 16px 26px 0`, table card `margin: 16px 26px`).

Density is medium-firm: table header cells `12px 18px`, body cells `13px 18px`, sidebar rows `9px 12px`, stat cards `16px 20px`. Cards wrap in a flex row with `gap: 14px` (`min-width: 150–280px`); form cards cap at `max-width: 560px`; the SMS filter input is `flex: 1; max-width: 340px`. Overlays: right-side drawers at 430px (compose) and 460px (settings), `max-width: 92vw`; centered modal at 640px, `max-width: 94vw; max-height: 86vh`.

Responsive fold at `max-width: 768px` (single breakpoint, desktop-first): the shell stacks vertically and the sidebar folds into a top device-select bar — brand (15px), compact scan button, and a native `<select>` (mono) replacing the device list; the header wraps with the IP line taking full width on its own row; tabs scroll horizontally with 14px gap; all gutters drop 26px → 14px; the SMS table gets `min-width: 560px` inside an `overflow-x: auto` card.

## Elevation & Depth

Flat by default, layered only at the edges. Resting surfaces (cards, table, sidebar, header) carry no shadow at all — depth between regions is drawn with 1px `#e5e7eb` borders plus sticky stacking (z-index 4/5). Shadow appears only when a surface separates from the page as an overlay, above a rgba(15, 23, 42, 0.32) mask.

### Shadow Vocabulary
- **Drawer Edge** (`box-shadow: -12px 0 28px rgba(15, 23, 42, 0.08)`): right-side drawers only; a soft leftward cast anchoring the panel to the screen edge.
- **Modal Float** (`box-shadow: 0 24px 60px rgba(15, 23, 42, 0.2)`): the centered scan modal; the one large-drop shadow in the system.

### Named Rules
**The Border-First Rule.** Surfaces are flat: white cards are drawn with 1px #e5e7eb borders and no shadow at rest. Shadow is an overlay event, never a card style.

## Shapes

The radius ladder climbs with surface size: 4px for tags and the focus ring, 6px for mini buttons and close buttons, **8px for every standard control** (`--radius: 8px` — buttons, inputs, segments, sidebar rows, banners), 10px for cards and the table, 12px for the badge pill, 14px for the modal. State dots and the logo tile are the only circles/rounded exceptions (50% / 7px).

Borders are always 1px. Solid `#e5e7eb` means bounded and interactive; **dashed 1px means explainer** — the 「演示数据为合成」 tag, the feature-off blocks (off-block), and the safety-mode note block all wear dashed borders to signal "reading, not acting". Row dividers inside tables soften to `#f1f3f5`.

### Named Rules
**The Dashed-Means-Explainer Rule.** A dashed 1px border marks a passive explanatory block (demo-data tag, feature-off notice, safety note). Interactive containers and controls always take solid borders. Don't use dashed for decoration.

## Components

### Buttons
- **Shape:** gently rounded 8px (`--radius`), 1px border.
- **Primary:** Console Blue fill and border (#2563eb), white text, `padding: 8px 16px`, 13px/600. The sidebar scan button is the full-width variant (`padding: 10px`, 13.5px). Disabled: `opacity: 0.55`, `cursor: not-allowed` (also while busy: 「发送中…」).
- **Secondary / Default:** white card fill, 1px #e5e7eb border, ink text, `8px 16px`, 13px. Mini variant inside modals: 6px radius, `5px 12px`, 12px.
- **Hover / Focus:** no custom hover treatment is defined — buttons rely on cursor + disabled states; interactive rows and tabs transition color/background at `0.14s ease`. Keyboard focus takes the global ring: `:focus-visible { outline: 2px solid #2563eb; outline-offset: 2px }`.

### Chips & Tags
- **SIM Tag:** #eff6ff background, #2563eb text, 4px radius, `2px 8px`, 10.5px — marks which SIM a row belongs to.
- **Result Tags** (scan modal): 4px radius, `2px 9px`, 11px — ok: #dcfce7/#16a34a; amber: #fffbeb/#d97706.
- **Status Badge (pill):** 12px radius, `3px 10px`, 11.5px/600 — 已连接 #dcfce7/#16a34a; 待凭据 #fffbeb/#d97706; 离线/连接中 #f2f4f7/#9ca3af.
- **Demo Tag:** 「演示数据为合成」 — 10.5px #9ca3af, 1px **dashed** #e5e7eb, 10px radius, `2px 8px`. Documented deviation from the comp; PRODUCT.md mandates the synthetic-data label.

### Cards / Containers
- **Corner Style:** 10px on all cards; overflow hidden on the table card to clip corners.
- **Background:** #ffffff on every resting card; overlay bodies may sit on #f8fafc with white rows.
- **Shadow Strategy:** none at rest — see The Border-First Rule.
- **Border:** 1px solid #e5e7eb; dashed variant for explainer blocks.
- **Internal Padding:** `16px 20px` stat cards, `18px 20px` form cards, `12px 18px` table cells; table footer pager `10px 18px`.
- **Table anatomy:** header row #fafbfc, 12px/600 #6b7280 labels; rows divided by #f1f3f5; numbers/hosts mono 600; pager with 21×21px page cells (5px radius, active cell blue with white text).

### Inputs / Fields
- **Style:** 1px #e5e7eb border, 8px radius (7px inside drawer/modal form grids), `padding: 8px 12px`, 13px, white fill. Machine values (host, port, MAC, phone) get the mono class.
- **Focus:** border shifts to #2563eb with a `2px solid rgba(37, 99, 235, 0.25)` outline (drawer forms). Textarea: same, `resize: vertical`. Select: native control dressed in the same border/radius.
- **Labels:** 11–12px #6b7280/#9ca3af above or beside the field; field groups gap 4–10px.
- **Error / Disabled:** errors surface as banners, not field styles; disabled submit `opacity: 0.55`.

### Navigation
- **Sidebar (desktop):** 264px white column, 1px right border, `padding: 18px 14px`. Brand: 26px blue logo tile (7px radius, white "S", 13px) + 17px/800 wordmark. Section label 11px #6b7280 with `letter-spacing: 2px`. Device rows: 8px status dot (green online / amber pending / #d1d5db offline), 13.5px/600 name, 11px mono IP, right-aligned battery %; hover #f3f4f6; active #eff6ff; transition `0.14s ease`. Footer 11px #6b7280.
- **Tabs:** white row, 1px bottom border, `gap: 22px`, `padding: 0 26px`. Tab: 14px #6b7280, `padding: 13px 2px`, transparent 2px bottom border; active: #2563eb text + blue underline + 600; disabled: #c3c8d2, `cursor: not-allowed`, renamed with its reason (「定位（未开启）」).
- **Mobile:** sidebar folds into a top bar (brand + compact scan button + native mono select); tabs scroll horizontally.

### Signature: Honest-State Surfaces
The recurring custom pattern that defines the system — every uncertainty gets a named, styled surface:
- **Loading hint:** 「查询中…」 12.5px #9ca3af, plain text, no spinner theater.
- **Empty hint:** 12.5px #6b7280 and worded to preserve doubt: 「未查询到短信——空列表可能与手机端权限或查询异常有关，不一定是没有记录」.
- **Error Banner** (`role="alert"`): #fef2f2 / 1px #fecaca, kind label in #dc2626 600 + message in #7f1d1d, 8px radius, `10px 14px`, 13px.
- **Result banners:** success #dcfce7 / #bbf7d0 / #16a34a; "result unknown" #fffbeb / #fde68a / #92400e — same geometry (8px radius, 12.5px/1.6).
- **Feature-off block:** dashed-border explainer with bold name + fix instruction (定位/WOL panels, blocked-SIM notice).
- **Progress:** 6px #f1f3f5 track (3px radius) with blue fill, live meta line (并发/超时/percent), cancel button — process narrated in place.

## Do's and Don'ts

### Do:
- **Do** set machine-produced values (IP:port, phone numbers, versions, coordinates) in Consolas mono (`The Mono-Is-Machine Rule`).
- **Do** use the soft-tint pair for every status: strong color on a soft tint (#16a34a on #dcfce7, #2563eb on #eff6ff, #d97706 on #fffbeb, #dc2626 on #fef2f2).
- **Do** keep controls at 8px radius, 13px text, `8px 16px` padding; body text at 14px/1.5.
- **Do** keep honest empty/loading/error language with the hint styles (12.5px, #9ca3af/#6b7280) — name the uncertainty instead of saying "no records".
- **Do** keep the global focus ring on keyboard navigation: `:focus-visible { outline: 2px solid #2563eb; outline-offset: 2px }`.
- **Do** keep the 「演示数据为合成」 dashed tag in the device header — it is a documented, mandated deviation from the comp.

### Don't:
- **Don't** introduce serif faces or webfonts — the system stack (Segoe UI / Microsoft YaHei / PingFang SC) and Consolas are the entire cast.
- **Don't** shadow a resting card; borders carry depth (`The Border-First Rule`).
- **Don't** add new accent hues — blue + green/amber/red is the complete chromatic set (`The One-Blue Rule`).
- **Don't** hide a disabled capability — gray it (#c3c8d2), keep it in place, and rename it with the reason (「定位（未开启）」) or use a dashed explainer block.
- **Don't** set UI text below 10.5px, and don't fake depth with heavy offset shadows — this is not a neobrutalist world.
