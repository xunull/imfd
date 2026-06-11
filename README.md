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

## 安装

```bash
go install github.com/xunull/imfd@latest
```

或从源码编译：

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

## 架构设计

### 并发流水线

```
目录并行遍历 (ants pool) → 文件通道 → 并行媒体提取 (ants pool) → 记录通道 → 单点聚合 → 报告输出
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
