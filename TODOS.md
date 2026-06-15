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

## verify C2PA / AI 检测的延后项（v2）

**背景**：`verify --c2pa` v1 已做 detection-only 的 AI 生成检测（C2PA manifest + EXIF/PNG keyword + SD 签名）。以下是有意延后的增强。

- [ ] **C2PA 密码学签名验证**（`--c2pa-verify`）：v1 只检测 manifest 存在 + 提取 generator，不验证 COSE 签名 / X.509 cert chain / TSP timestamp。完整验证让 verify 能用于法证。已引入 `fxamacker/cbor`，验证栈可在此基础上加；或 subprocess 调 c2patool（但破坏纯 Go 静态 binary）。
- [ ] **HEIC / MP4 / WebP 的 C2PA**：v1 只解 JPEG App11 + PNG chunk。HEIC（iPhone 默认）、MP4、WebP 用 ISOBMFF / RIFF container 嵌 C2PA，需各自 box/chunk parser。HEIC 优先级最高（iPhone 用户基数大）。
- [ ] **PNG 压缩 iTXt chunk**：v1 只解未压缩 iTXt（flag=0）。压缩 iTXt 需 zlib 解压。ComfyUI 默认不压缩所以不急，但某些工具会压缩。
- [ ] **Photoshop 生成式填充检测**：Photoshop generative fill 写 `Software="Adobe Photoshop"`，AI 生成信息只在 C2PA manifest 的 action（`c2pa.created` + AI ingredient）里。v1 靠 C2PA present 能抓到，但若 C2PA 被剥则漏判为 `edited`。v2 可解析 C2PA actions 区分「AI 生成」vs「AI 编辑」。
- [ ] **cache nullable-field migration**：目前每次扩展 ExifInfo 字段都 bump schema 触发全量重建（已 bump 3 次）。v2 改成 nullable 字段共存 + 增量迁移，避免用户每次升级都重扫整库。

## Demo 脚本里追加 verify 演示

`asciinema` demo（见上面的发布计划）录制时，加一段 verify 演示：
```bash
imfd verify --c2pa ai-image.jpg   # → ai-generated, generator: DALL·E 3
imfd list --ai ~/Downloads        # 找出下载目录里的 AI 图
```
「能检测 AI 生成图的 CLI」是 r/commandline / HN 的流量点。

---

<!-- 未来新增的 TODO 项添加到上面。已完成的项移到下方归档。 -->

## Archive

（暂无）
