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
