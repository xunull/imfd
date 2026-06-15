# imfd - Image & Media File Detective

媒体文件统计工具，扫描目录中的图像、视频和音频文件，提取元数据与 EXIF/编解码信息，进行多维统计分析。

## 功能特性

- 递归扫描目录中的图像、视频、音频文件
- 提取图像 EXIF 信息（光圈、快门、焦距、ISO、白平衡、曝光参数、相机/镜头型号、GPS 等）
- 提取视频元数据（编码器、分辨率、时长、比特率等，依赖 ffprobe）
- 提取音频元数据（编解码器、采样率、比特率、声道布局、时长、录制时间，依赖 ffprobe）
- 离线 GPS 反查到省/市
- 多维度统计聚合：
  - 媒体类型（图像/视频/音频）
  - 相机型号、镜头型号
  - 拍摄时间段（凌晨/上午/中午/下午/晚上/半夜）
  - 拍摄地省/市
  - ISO、光圈、快门、焦距、曝光模式/程序、白平衡、测光模式、闪光灯等
  - 音频编解码器、比特率桶、采样率、声道布局、时长桶
- 可扩展的维度统计框架，新增维度只需一个 `KeyExtractor` 函数（或用 `stats.NewFieldDimension` 工厂）
- 使用 ants 协程池实现目录遍历与媒体提取的并行处理
- 支持终端表格和 JSON 两种输出格式
- 子命令矩阵：`scan`（聚合统计）/ `info`（单文件详情）/ `list`（按条件筛路径，pipe 友好）/ `view`（按条件在 Finder 或指定 app 中弹虚拟视图）/ **`verify`**（侦探：AI 生成检测 + 编辑检测，含 C2PA Content Credentials）/ `cache`（管理元数据 cache）

## 安装

### 预编译二进制（推荐）

从 [GitHub Releases](https://github.com/xunull/imfd/releases/latest) 下载对应平台的 tarball：

```bash
# macOS Apple Silicon
tar -xzf imfd_*_darwin_arm64.tar.gz
sudo mv imfd /usr/local/bin/

# macOS 首次运行需绕过 Gatekeeper（未做 Apple 公证）
xattr -d com.apple.quarantine /usr/local/bin/imfd

# Linux
tar -xzf imfd_*_linux_amd64.tar.gz   # 或 linux_arm64
sudo mv imfd /usr/local/bin/
```

支持平台：`darwin-amd64` / `darwin-arm64` / `linux-amd64` / `linux-arm64`。

### 通过 go install

```bash
go install github.com/xunull/imfd@latest
```

### 从源码编译

```bash
git clone https://github.com/xunull/imfd.git
cd imfd
go build -o imfd .
```

## 前置依赖

- **ffprobe**（可选）：用于提取视频和音频元数据，属于 FFmpeg 工具集。未安装时仍会按扩展名计数和扫描，只是音视频元数据不可用，会在 `Attributes["video_error"]` / `Attributes["audio_error"]` 中记录原因。
  ```bash
  # macOS
  brew install ffmpeg
  # Ubuntu/Debian
  sudo apt install ffmpeg
  ```

## 使用方法

```bash
# 扫描当前目录（默认全部媒体类型）
imfd scan

# 扫描指定目录
imfd scan /path/to/photos

# 仅扫描指定类型
imfd scan audio /path/to/music     # 仅音频
imfd scan image /path/to/photos    # 仅图像
imfd scan video /path/to/videos    # 仅视频
imfd scan all /path/to/dir         # 等同于 imfd scan /path/to/dir

# 指定输出格式
imfd scan -f json /path/to/photos
imfd scan -f both /path/to/photos   # 同时输出表格和 JSON

# 调整并发参数
imfd scan -w 16 -e 32 /path/to/photos
imfd scan -w 16 audio /path/to/music   # flag 自动继承到子命令
```

### 子命令

| 命令 | 行为 |
|---|---|
| `imfd scan [dir]` | 扫全部媒体类型（向后兼容） |
| `imfd scan all [dir]` | 显式全扫，等同上一行 |
| `imfd scan audio [dir]` | 仅扫音频文件（walker 层过滤，不浪费 ffprobe 调用） |
| `imfd scan image [dir]` | 仅扫图像文件 |
| `imfd scan video [dir]` | 仅扫视频文件 |

按指定类型扫描时，与该类型无关的统计维度（如 scan image 时的音频维度）会自动跳过注册，输出更聚焦。

### 输出示例（dashboard）

```
imfd · scan audio · ./music
─────────────────────────────────────────────────────────────────
scanned 24 files · 0.12s · 0 errors

OVERVIEW
  图像       0  ░░░░░░░░░░░░░░░░░░░░   0%
  视频       0  ░░░░░░░░░░░░░░░░░░░░   0%
  音频      24  ████████████████████ 100%
  总计      24

AUDIO
  音频编解码器   mp3              18  ████████████████████  75%
                 flac              4  ████░░░░░░░░░░░░░░░░  17%
                 wav               2  ██░░░░░░░░░░░░░░░░░░   8%

  音频比特率     320kbps          12  █████████████░░░░░░░  50%
                 192kbps           8  █████████░░░░░░░░░░░  33%
                 128kbps           4  █████░░░░░░░░░░░░░░░  17%

  音频采样率     44.1kHz          20  ████████████████████  83%
                 48kHz             4  ████░░░░░░░░░░░░░░░░  17%

  音频声道       stereo           22  ████████████████████  92%
                 mono              2  ██░░░░░░░░░░░░░░░░░░   8%

  音频时长       1-5 分钟         18  ████████████████████  75%
                 <1 分钟           4  ████░░░░░░░░░░░░░░░░  17%
                 5-30 分钟         2  ██░░░░░░░░░░░░░░░░░░   8%
```

混合扫描时 (scan all) 与当前 scan 类型无关的「全 Unknown」维度默认折叠为单行提示，`-v` 展开。终端默认 256 色（image=蓝/video=紫/audio=绿）；`NO_COLOR=1` 或 pipe 到文件时自动禁色。`IMFD_ASCII=1` 用 ASCII fallback (#/.) 替代 Unicode block。

### 进度反馈

扫描期间在 stderr 显示 spinner + 实时计数：

```
⠋ scanned 1245 files · 412 extracted...
```

不在 TTY、NO_COLOR、IMFD_NO_SPINNER=1 时自动关闭。

### 关键 flag

| flag | 默认 | 说明 |
|---|---|---|
| `-v`, `--verbose` | false | 展开「全 Unknown」维度 |
| `--legacy-table` | false | 回退到旧 go-pretty 表格输出 |
| `-f`, `--format` | table | `table` / `json` / `both`；table 默认走 dashboard |

## 单文件查询 (info)

`imfd info <file>` 看单个文件的完整元数据。和 `scan`（聚合统计）相对——`info` 是逐文件展开。

```bash
imfd info photo.jpg               # 单文件
imfd info *.jpg                   # shell glob 多文件
find . -name '*.heic' | xargs imfd info  # xargs 流水线
imfd info song.mp3 -f json        # JSON 输出（snake_case 字段）
```

输出按 section 分组（FILE / EXIF / GPS / AUDIO / VIDEO / ERRORS），缺失字段或全空 section 自动隐藏：

```
FILE
  路径         /Users/quincy/Pictures/IMG_8784.JPG
  大小         2.29 MB
  修改时间     2024-08-12 15:30:42
  类型         image

EXIF
  相机         Canon EOS 1300D
  镜头         EF-S18-55mm f/3.5-5.6 IS II
  拍摄时间     2024-03-15 14:22:11
  ISO          800
  光圈         f/5
  快门         1/60s
  焦距         42mm
  尺寸         6000x4000

GPS
  纬度         31.230400
  经度         121.473700
  地点         上海市 / 黄浦区
```

**多文件错误模型**：某个文件失败不中断后面文件，错误印到 stderr，末尾退出码 = 1 如果任一失败。
`imfd info ./dir` 传目录时友好报错并提示用 `scan`。

### info 关键 flag

| flag | 默认 | 说明 |
|---|---|---|
| `-f`, `--format` | table | `table`（人读 section）/ `json`（marshal MediaRecord，snake_case）|
| `-g`, `--geo-provider` | offline | GPS 反查方式：offline（离线中国城市表）/ nominatim（OSM 在线）|

## 按条件查询 (list)

`imfd list` 按 EXIF / GPS / 设备类型等条件挑出文件路径，pipe 友好。把 imfd 从「统计 / 查看」工具升格为「**可组合的 unix 媒体查询工具**」。

```bash
# 找出云南手机拍的照片，复制到目录做相册
imfd list --type image --province 云南 --device phone ~/Pictures \
  | xargs -d '\n' cp -t ~/yunnan-phone/

# 找出 Sony 拍的 ISO > 1000 的图像
imfd list --type image --camera-make Sony --iso ">1000" ~/Pictures

# 找出 starry_sky 启发式匹配的星空照
imfd list --type image --scene starry_sky ~/Pictures

# 高阶组合查询用 expr-lang DSL
imfd list --filter "province contains '云南' and device_type == 'phone' and capture_year >= 2024" ~/Pictures

# NUL 分隔（filenames 含 \n 也安全）
imfd list --camera-make Sony -0 ~/Pictures | xargs -0 wc -c

# 统计某品牌照片数量
imfd list --camera-make Nikon ~/Pictures | wc -l

# pipe 到 imfd info 看详情
imfd list --camera-make Sony ~/Pictures | head -3 | xargs -d '\n' imfd info
```

### list 字段

DSL 表达式可用的字段（扁平 snake_case）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `file_path` / `file_name` / `file_size` / `type` | str/str/int/str | 基础 |
| `camera_make` / `camera_model` / `lens_make` / `lens_model` | str | EXIF |
| `iso` / `aperture_value` / `shutter_seconds` / `focal_length_mm` | int / float | EXIF 数值（已 parse） |
| `image_width` / `image_height` | int | 图像尺寸 |
| `province` / `city` / `country` | str | GPS 反查后地点（中文） |
| `capture_year` / `capture_hour` | int | 从 EXIF DateTimeOriginal 推导 |
| `audio_codec` / `audio_bitrate` / `audio_sample_rate` | str / int / int | 音频 |
| `video_codec` / `video_width` / `video_height` | str / int / int | 视频 |
| `device_type` | str | `phone` / `camera` / `unknown`（基于 camera_make 内置映射） |
| `scene_starry_sky` | bool | 启发式：iso>1600 AND shutter>10s AND 拍摄小时 ∈[22,4] |

### list 常用 flag

| flag | 说明 |
|---|---|
| `--type {image,video,audio,all}` | 默认 all；走 walker 层早期过滤减少 ffprobe 调用 |
| `--camera-make`, `--camera-model` | 字符串 substring 匹配（case-insensitive）；可重复 → OR |
| `--lens` | 镜头型号 substring；可重复 → OR |
| `--device {phone,camera}` | 设备类别精确匹配 |
| `--codec` | 编解码器 substring，**同时匹配音频和视频**；可重复 → OR |
| `--audio-codec`, `--video-codec` | 单独指定音/视频 codec；可重复 → OR |
| `--province`, `--city` | 字符串 substring；可重复 → OR |
| `--scene starry_sky` | v1 唯一 scene 启发式 |
| `--iso N`, `--iso ">N"`, `--iso "<N"`, `--iso "N-M"` | 数值 range 4 种语法 |
| `--year N` / `--year ">=N"` / `--year "N-M"` | 同上 |
| `--filter "expr"` | expr-lang DSL（高阶；和 flag 是 AND） |
| `-0`, `--print0` | NUL 分隔（xargs -0 友好） |

### list exit codes

| code | 含义 |
|---|---|
| 0 | 成功（0 个结果也是 0；xargs 友好） |
| 1 | IO 错误（路径不存在 / 权限拒绝） |
| 2 | filter 语法错误（stderr 印 column） |

### list DSL 语法

`--filter` 是 [expr-lang/expr](https://expr-lang.org/) 语法。常用：

- 比较：`==`、`!=`、`<`、`>`、`<=`、`>=`
- 布尔：`and`、`or`、`not`
- 字符串 substring：`field contains "X"`（**不是 `"X" in field`**）
- 字符串 lowercase：`lower(field)` 
- 集合 membership（精确匹配字段）：`type in ["image", "video"]`
- 字符串字面量：**必须有引号**——`audio_codec == 'flac'` 或 `"flac"`。裸字 `flac` 会被当成字段名查 env

**Shell 引号嵌套小技巧**：

```bash
# 推荐：外层单引号，shell 不解释里面任何字符
imfd list --filter 'audio_codec == "flac"' ./dir

# 也行：外层双引号，内层单引号
imfd list --filter "audio_codec == 'flac'" ./dir
```

**但单字段精确匹配建议用 flag**（零引号、零 quote 嵌套）：

```bash
# 这个：
imfd list --codec flac ./dir

# 等价于：
imfd list --filter 'audio_codec contains "flac" or video_codec contains "flac"' ./dir
```

DSL 留给**多字段复合查询**（`A and B or not C` 这种）。

**nil-safe**：缺字段的比较自动返回 false（如 audio 文件没 EXIF，`iso > 800` 不命中、不报错）。

### 命令参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--dir` | `-d` | `.` | 要扫描的目录路径 |
| `--workers` | `-w` | `8` | 目录遍历并发数 |
| `--extractors` | `-e` | `16` | 媒体提取并发数 |
| `--format` | `-f` | `table` | 输出格式: table, json, both |
| `--channel-size` | | `1024` | 内部通道缓冲大小 |

## 虚拟视图 (view)

`imfd view` 是 `list` 的图形化对应物：用同样的筛选条件，把命中的文件以**符号链接**形式放进一个临时目录，默认在 Finder 打开。也可以用 `--exec` 把虚拟目录交给任意 app（Lightroom / Photos / 文件管理器）。**原始文件零移动、零修改**——你看到的是「假目录」，里面的链接指回原始位置。

```bash
# 在 Finder 中弹出云南手机拍的照片（虚拟目录，macOS）
imfd view --province 云南 --device phone ~/Pictures

# 在 Lightroom 中打开虚拟视图（虚拟目录直接进 Lightroom）
imfd view --province 云南 ~/Pictures --exec "open -a 'Adobe Lightroom Classic'"

# 重命名 + Lightroom（导入时直接看到日期+地点+相机的文件名）
imfd view --province 云南 ~/Pictures \
    --rename "{date}_{city}_{camera_make}.{ext}" \
    --exec "open -a 'Adobe Lightroom Classic'"

# Linux 用户：用自己的文件管理器
imfd view --province 云南 ~/Pictures --exec nautilus
imfd view --province 云南 ~/Pictures --exec thunar

# 不打开任何 app，只输出虚拟目录路径（脚本里用）
imfd view --province 云南 ~/Pictures --no-open

# 同一查询重复运行 → 同一虚拟目录（Finder 不会开新窗口，仅刷新内容）
imfd view --province 云南 ~/Pictures
imfd view --province 云南 ~/Pictures  # 命中已有 Finder 窗口
```

### view 重命名模板字段

`--rename` 模板里可用的占位符：

| 占位符 | 值 | 缺失时 |
|---|---|---|
| `{year}` / `{month}` / `{day}` | 拍摄日期分段（补零） | 回退到文件 mtime |
| `{date}` | `2024-01-15` 快捷写法 | 同上 |
| `{camera_make}` / `{camera_model}` | EXIF 相机品牌/型号 | `Unknown` |
| `{city}` / `{province}` | GPS 反查地点 | `Unknown` |
| `{type}` | `image` / `video` / `audio` | `unknown` |
| `{iso}` | EXIF ISO | `0` |
| `{ext}` | 原扩展名（小写） | 保持原样 |
| `{filename}` | 原文件名去扩展名 | 保持原样 |

文件名中的 `/`、`:`、`\0` 自动替换为 `_`，避免 macOS 文件系统报错。

### view 常用 flag

| flag | 说明 |
|---|---|
| `--rename "{tmpl}"` | symlink 重命名模板（默认保留原文件名） |
| `--exec "<cmd>"` | 执行命令，虚拟目录作为最后一个参数追加（隐含 `--no-open`）。例: `--exec "open -a 'Adobe Lightroom'"` 或 `--exec nautilus` |
| `--no-open` | 不打开 Finder，只把虚拟目录路径输出到 stdout |
| `--no-cache` | 跳过元数据 cache，强制重新提取 |
| 过滤 flag（`--type` / `--camera-make` / `--province` / `--filter` 等） | 与 `imfd list` **完全相同**，复用同一筛选引擎 |

**`--exec` 行为细节**：

- 实现等价 `sh -c "<cmd> <viewDir-shell-quoted>"`，所以可以用 shell 引号嵌套、环境变量、`&&` 等
- 虚拟目录追加在命令末尾（类似 `find -exec ... {} +` / `xargs`）
- 用户命令的退出码会传播：`imfd view --exec false; echo $?` 输出 `1`
- 0 个文件匹配时不执行（无意义）

### view 工作原理

```
imfd view --province 云南 ~/Pictures
        │
        ├─ 计算 FNV-32 hash(filter_expr + sorted(abs_paths))
        │
        ├─ /tmp/imfd-view-a3f7b2/  ← 同查询同目录
        │     ├─ DSC_0001.JPG → /Users/q/Pictures/2024/DSC_0001.JPG
        │     ├─ DSC_0042.JPG → /Users/q/Pictures/2024/raw/DSC_0042.JPG
        │     └─ ...
        │
        └─ open /tmp/imfd-view-a3f7b2/   ← Finder 在此聚焦
```

- **生命周期**：临时目录在 `$TMPDIR`（macOS 默认 `/var/folders/...`），系统重启后自动清理；同一查询重复运行只刷新 symlink，不会泄漏旧目录。
- **碰撞处理**：跨子目录有同名文件时自动加 `_1`、`_2` 后缀。
- **symlink target 是绝对路径**：用 Finder 直接打开链接也能找到原始文件（无论你之后在哪个目录运行 imfd）。
- **`cleanOldSymlinks` 只清理 symlink**：你手动放进虚拟目录的普通文件（笔记、对比图）不会被删。

### view 平台支持

| 平台 | 默认（开 Finder） | `--exec <cmd>` | `--no-open` |
|---|---|---|---|
| macOS | ✅ | ✅ | ✅ |
| Linux | ❌ exit 2 | ✅（用 nautilus/thunar/krusader 等） | ✅ |
| Windows | ❌ exit 2 | ✅（理论上；未在 Windows 上测） | ✅ |

只有「默认动作 = 打开 Finder」依赖 macOS 的 `open` 命令。指定 `--exec` 或 `--no-open` 时，view 在任何平台上都能工作。

## 侦探 (verify) — AI 生成检测 + 编辑检测

`imfd verify` 是 imfd 的「侦探」命令，对一张图像给出**两个独立维度**的判定：

- **AI 生成**：`ai-generated` / `not-ai` / `unknown` —— 这张图是 AI 画的吗？
- **后期编辑**：`original` / `camera-rendered` / `edited` / `unknown` —— 这张图被 Lightroom/PS 处理过吗？

> ⚠ **detection-only**：verify 只读元数据声明，**不验证密码学签名**。EXIF Software
> 字段和 C2PA manifest 都可以被伪造。**不要把 verify 结果当作法律证据或内容审核的
> 唯一依据**。它回答的是「这张图_声称_自己是什么」，不是「我能_证明_它是什么」。

```bash
# 单文件人类可读报告（AI + 编辑双判定）
imfd verify ~/photo.jpg

# 展开 C2PA Content Credentials manifest 详情
imfd verify ~/photo.jpg --c2pa

# JSON 输出（脚本友好）
imfd verify ~/photo.jpg -f json

# 批量审计
imfd list --ai ~/Photos              # 只看 AI 生成图
imfd list --not-ai ~/Photos          # 排除 AI 生成图
imfd list --edited ~/Photos          # 只看编辑过的
imfd list --ooc ~/Photos             # out-of-camera 直出

# 组合：在 Lightroom 打开「所有非 AI、Sony 拍的直出」
imfd view --ooc --not-ai --camera-make Sony ~/Photos --exec "open -a 'Adobe Lightroom Classic'"

# DSL：找出 AI 生成 + 已编辑的图
imfd list --filter "is_ai_generated == true and is_edited == true" ~/Photos | wc -l
```

### AI 生成检测

3 级判定，多信号「强/弱」合并：

| AI Verdict | 含义 |
|---|---|
| `ai-generated` | 检测到 AI 生成信号（强信号一个即判，或弱信号 ≥2） |
| `not-ai` | 有可分析元数据但无 AI 信号 |
| `unknown` | 元数据不足（无 EXIF / C2PA / PNG 文本） |

**强信号（任一命中 → ai-generated）：**
- C2PA Content Credentials manifest（DALL·E 3 / Adobe Firefly / ChatGPT 默认嵌入，JPEG App11 + PNG chunk 都解析）
- EXIF `Software` 字段含 AI 工具关键字
- PNG `Software` 文本字段含 AI 工具关键字
- PNG `parameters` 含 Stable Diffusion 生成签名（`Steps:` + `Sampler:`/`CFG scale:`）

**弱信号（需 ≥2 不同 key → ai-generated）：** PNG 文本里出现 `prompt` / `workflow` / `comfyui` / `parameters`（ComfyUI 写 prompt+workflow 两个 → 命中）

**识别的 AI 工具关键字：** DALL·E, Midjourney, Stable Diffusion, SDXL, Automatic1111/A1111, ComfyUI, InvokeAI, Fooocus, Adobe Firefly, Bing Image Creator, Imagen, FLUX, NovelAI, Leonardo.ai, Ideogram, Recraft, DreamStudio, GPT-image

**防假阳**：AI 关键字**只在 Software / C2PA generator / 指定 PNG key 上匹配**，绝不在任意文本里匹配——所以 `Description="midjourney inspired"` 的普通照片**不会**被误判。

### 编辑检测

| Verdict | 含义 | 触发条件 |
|---|---|---|
| `original` | 相机直出（OOC） | Software 空且 ModifyDate 与 DateTimeOriginal 差 ≤60s（或 ModifyDate 缺失） |
| `camera-rendered` | 相机内置软件渲染 | Software 含相机厂商关键字：Imaging Edge / DPP / RAW Converter / HDR+ 等 |
| `edited` | 经过后期编辑工具处理 | Software 含编辑器关键字 或 ModifyDate 比 DateTimeOriginal 晚 >60s |
| `unknown` | 信号不足无法判定 | 无 EXIF 数据（PNG screenshot），或 Software 字段未归类 |

**编辑器关键字**（→ `edited`）：lightroom, photoshop, capture one, luminar, affinity, pixelmator, preview, photos (macOS/iOS), darktable, rawtherapee, on1, dxo, snapseed, vsco, gimp

**相机内置软件**（→ `camera-rendered`，**不算** edited）：imaging edge (Sony), digital photo professional (Canon), raw file converter (Fujifilm), hdr+ (Pixel), firmware

> AI 工具名（如 "Adobe Firefly"）优先归 AI，不归编辑器——AI 生成 ≠ 人工编辑，两个维度独立。

### flag

| 命令 | flag | 说明 |
|---|---|---|
| `verify` | `-f` / `--format` | `table`（默认）/ `json` |
| `verify` | `--c2pa` | 展开 C2PA MANIFEST section（生成器、信任级别） |
| `list` / `view` | `--ai` / `--not-ai` | 过滤 AI 生成 / 非 AI（互斥） |
| `list` / `view` | `--edited` / `--ooc` | 过滤已编辑 / 直出（互斥） |

互斥 flag 同时给会 exit 2。可与所有现有 filter 组合（`--camera-make` / `--province` / `--filter`）。

### 支持格式

| 格式 | EXIF | C2PA manifest | PNG 文本信号 |
|---|---|---|---|
| JPEG | ✅ | ✅ App11 JUMBF（多 marker 自动重组） | — |
| PNG | — | ✅ 内嵌 JUMBF | ✅ tEXt + iTXt（未压缩） |
| HEIC / MP4 / WebP | — | ❌ v2 | — |

### Edge cases

| 场景 | 处理 |
|---|---|
| 非图像文件（mp4 / mp3） | verify SKIP exit 0；多文件继续 |
| 缺 EXIF 整段（PNG / 截图） | edit/AI verdict 视信号而定，无信号 → `unknown` |
| 相机 RAW→JPEG 转换（ModifyDate 晚几十秒） | 60s 容忍窗口保护，不算 edited |
| C2PA manifest > 64 KB | 只读文件头 64 KB；超出可能仅 Present 无 generator，或退回 keyword 信号 |
| 单个 PNG `parameters` 无 SD 签名 | 仅 1 弱信号 → 不判 AI（防假阳） |

### 已知局限（v1 不修，见 TODOS.md）

- **不验证 C2PA 密码学签名** —— detection-only，manifest 可伪造
- Photoshop 生成式填充（generative fill）写 `Software="Adobe Photoshop"`，AI 生成信息只在 C2PA 里 —— 无 C2PA 时会被判 `edited` 而非 ai-generated
- PNG **压缩** iTXt chunk 不解（仅未压缩）；ComfyUI 默认不压缩，不受影响
- HEIC（iPhone 默认）/ MP4 / WebP 的 C2PA 暂不解
- 不深度解析 XMP edit history

## 元数据 Cache

imfd 内置 SQLite cache，首次扫描后将 EXIF/音视频元数据写入本地数据库，后续对同一目录的 `scan`/`list` 调用无需重新启动 ExifTool/FFprobe，速度提升 10-100x。

**Cache 透明工作**，无需配置：文件 mtime 未变 → 命中缓存；文件修改/新增 → 自动失效并重新提取。

> **升级提示**：当 imfd 扩展提取的元数据字段时（如新增 verify 的 C2PA 检测），cache schema 版本会自增，**首次运行旧 cache 会自动重建一次**（无需手动操作，全量重提取，几分钟到 30 分钟取决于库大小）。重建后恢复秒级。

### 查看 cache 状态

```bash
imfd cache stats
# Cache DB:  /Users/q/.cache/imfd/cache.db
# Entries:   12,847
# Size:      42.3 MB
# Oldest:    2025-12-01 (192 days ago)
```

### 清理 cache

```bash
# 删除 90 天未访问的旧条目（推荐定期执行）
imfd cache clean --older-than 90d

# 清空全部
imfd cache clear
```

### 跳过 cache

```bash
# 强制重新提取（调试 / 元数据更新后验证）
imfd scan --no-cache ~/Pictures
imfd list --no-cache --province 云南 ~/Pictures
```

### Cache 位置

| 环境 | 路径 |
|------|------|
| 默认 | `~/.cache/imfd/cache.db` |
| 自定义 | 设置 `XDG_CACHE_HOME` 环境变量 |

---

## 架构设计

### 并发流水线

```
目录并行遍历 (ants pool) → 文件通道 → 并行媒体提取 (ants pool / cache hit) → 记录通道 → 单点聚合 → 报告输出
```

### 可扩展统计维度

新增统计维度只需要：

1. 编写一个 `KeyExtractor` 函数
2. 用 `NewGroupCounter` 包装
3. 注册到 `Registry`

```go
func NewMyDimension() stats.DimensionCounter {
    return stats.NewGroupCounter(
        "维度名称",
        func(record *media.MediaRecord) []string {
            // 从 record 中提取分组键
            return []string{record.Exif.SomeField}
        },
        stats.DimensionMeta{SortBy: "count", SortOrder: "desc"},
    )
}
```

## 支持的文件格式

### 图像
JPG, JPEG, PNG, GIF, BMP, TIFF, WebP, HEIC, HEIF, RAW, CR2, CR3, NEF, ARW, DNG, ORF, RW2, PEF, SR2, RAF

### 视频
MP4, MOV, AVI, MKV, WMV, FLV, M4V, MPG, MPEG, 3GP, WebM, MTS, M2TS, TS

### 音频
MP3, FLAC, AAC, M4A, OGG, OGA, Opus, WAV, WMA, APE, WV, ALAC, DSD, DSF, DFF, AIFF, AIF, AMR

## JSON 输出契约

- `totals.total_count` = `image_count` + `video_count` + `audio_count`
- 未来新增媒体类型会继续加 `*_count` 字段；下游消费者请用 `total_count`，不要用各分项相加推算

## License

See [LICENSE](LICENSE) file.
