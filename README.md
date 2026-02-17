# imfd - Image & Media File Detective

媒体文件统计工具，扫描目录中的图像和视频文件，提取元数据与 EXIF 信息，进行多维统计分析。

## 功能特性

- 递归扫描目录中的图像和视频文件
- 提取图像 EXIF 信息（光圈、快门、焦距、ISO、白平衡、曝光参数、相机/镜头型号、GPS 等）
- 提取视频元数据（编码器、分辨率、时长等，依赖 ffprobe）
- 离线 GPS 反查到省/市
- 多维度统计聚合：
  - 媒体类型（图像/视频）
  - 相机型号、镜头型号
  - 拍摄时间段（凌晨/上午/中午/下午/晚上/半夜）
  - 拍摄地省/市
  - ISO、光圈、快门、焦距、曝光模式/程序、白平衡、测光模式、闪光灯等
- 可扩展的维度统计框架，新增维度只需一个 `KeyExtractor` 函数
- 使用 ants 协程池实现目录遍历与媒体提取的并行处理
- 支持终端表格和 JSON 两种输出格式

## 安装

```bash
go install github.com/xunull/imfd/cmd/imfd@latest
```

或从源码编译：

```bash
git clone https://github.com/xunull/imfd.git
cd imfd
go build -o imfd ./cmd/imfd/
```

## 前置依赖

- **ffprobe**（可选）：用于提取视频元数据，属于 FFmpeg 工具集
  ```bash
  # macOS
  brew install ffmpeg
  # Ubuntu/Debian
  sudo apt install ffmpeg
  ```

## 使用方法

```bash
# 扫描当前目录
imfd scan

# 扫描指定目录
imfd scan /path/to/photos

# 指定输出格式
imfd scan -f json /path/to/photos
imfd scan -f both /path/to/photos   # 同时输出表格和 JSON

# 调整并发参数
imfd scan -w 16 -e 32 /path/to/photos
```

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

## License

See [LICENSE](LICENSE) file.
