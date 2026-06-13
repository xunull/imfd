# TODOS

未来要做、但当前不优先的事项。新增请用相同格式：标题、动机、待办清单、预估、参考链接。

---

## Demo + 公开发布（等核心功能更完善后再做）

**目标**：让 GitHub / Reddit 上的访客 30 秒内看到 imfd 能干什么，从而决定要不要装。

**前置条件**（等做完再启动这一项）：
- 至少把 `dedup` 或 `verify` 一个再做掉，让工具的「detective」身份更立得住
- 核心功能稳定，CI 全绿
- 已经 tag 过一个 stable 版本（目前 v0.0.2）

**待办**：

- [ ] 录 asciinema cast（脚本见下）
  - `brew install asciinema`
  - `asciinema rec -i 2 --title "imfd vX.Y.Z" imfd-demo.cast`
  - 60 秒脚本（4 个场景）：
    1. `imfd scan ~/Pictures/2024` — dashboard 一晃而过（5s）
    2. `imfd list --province 云南 --device phone ~/Pictures | head -5` — 管道组合（15s）
    3. `imfd view --province 云南 ~/Pictures --rename "{date}_{city}.{ext}"` — Finder 弹虚拟相册（15s）
    4. `imfd view --province 云南 ~/Pictures --exec "open -a 'Adobe Lightroom Classic'"` — Lightroom 直接打开（20s）
    5. `imfd cache stats` — 展示 cache 加速效果（5s）

- [ ] 上传：`asciinema upload imfd-demo.cast` → 拿到 URL
- [ ] README 顶部加 badge：`[![asciicast](https://asciinema.org/a/XXXXXX.svg)](https://asciinema.org/a/XXXXXX)`
- [ ] 同时用 `agg` 转 GIF 备份到 `docs/imfd-demo.gif`，README 用 `<picture>` 提供两种 fallback
- [ ] 发 r/commandline
  - 标题：`[OC] imfd — query and organize photos/videos by EXIF metadata`
  - 正文模板：1 句话痛点 + 3-5 个 hero use-case 代码块 + asciinema 链接 + repo URL
  - 发帖时机：周二-周四 UTC 14:00-18:00（美东早上 / 欧洲下午）
- [ ] 间隔几天后发其它相关板块（避免被判 spam）：
  - `r/macapps`（view 的 Finder 魔法对 mac 用户冲击最大）
  - `r/golang`
  - `r/photography`
  - `r/selfhosted`
  - Hacker News（Show HN）— 一次性大流量但短暂

**预估**：30-60 分钟（录制 + 上传 + 发帖）

**为什么暂不做**：工具够好之前发出去 ROI 反而低——访客一看「就这？」很难回来第二次。先让 dedup / verify 落地，让「media file detective」名字真正立得住。

---

<!-- 未来新增的 TODO 项添加到上面。已完成的项移到下方归档。 -->

## Archive

（暂无）
