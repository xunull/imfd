#!/usr/bin/env bash
# 生成集成测试 fixture。
# 依赖：ffmpeg（音视频）+ exiftool（图像 EXIF 字段写入）
set -euo pipefail

cd "$(dirname "$0")"

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "ffmpeg not installed; aborting" >&2
  exit 1
fi

# ─── 音视频 fixture ─────────────────────────────────────────
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" -b:a 192k audio.mp3 2>/dev/null
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" audio.flac 2>/dev/null
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" audio.wav 2>/dev/null
ffmpeg -y -f lavfi -i "testsrc=duration=1:size=320x240:rate=30" -pix_fmt yuv420p video.mp4 2>/dev/null

# ─── 图像 fixture（verify edit detection 用） ────────────────
# 需要 exiftool。如果没装，跳过图像 fixture 生成（音视频 fixture 不受影响）。
if ! command -v exiftool >/dev/null 2>&1; then
  echo "WARN: exiftool not installed, skipping image fixtures" >&2
  echo "      install via 'brew install exiftool' or 'apt install libimage-exiftool-perl'" >&2
  echo "Generated:"
  ls -lh audio.mp3 audio.flac audio.wav video.mp4
  exit 0
fi

# 用 ffmpeg 生成一张纯色 jpg 作为底片，然后 exiftool 写不同元数据
ffmpeg -y -f lavfi -i "color=red:size=320x240" -frames:v 1 _base.jpg 2>/dev/null

# 1) image_original_sony.jpg —— 模拟 Sony 直出（Software 空，仅 DateTimeOriginal）
cp _base.jpg image_original_sony.jpg
exiftool -overwrite_original \
  -Make='SONY' \
  -Model='ILCE-7RM5' \
  -LensModel='FE 24-70mm F2.8 GM II' \
  -DateTimeOriginal='2024:03:15 14:23:01' \
  -CreateDate='2024:03:15 14:23:01' \
  image_original_sony.jpg >/dev/null

# 2) image_edited_lightroom.jpg —— Lightroom export（Software 命中 + ModifyDate 5d 后）
cp _base.jpg image_edited_lightroom.jpg
exiftool -overwrite_original \
  -Make='SONY' \
  -Model='ILCE-7RM5' \
  -Software='Adobe Lightroom Classic 13.0 (Macintosh)' \
  -DateTimeOriginal='2024:03:15 14:23:01' \
  -CreateDate='2024:03:15 14:23:01' \
  -ModifyDate='2024:03:20 09:15:42' \
  image_edited_lightroom.jpg >/dev/null

# 3) image_camera_rendered_sony.jpg —— Sony 内置 Imaging Edge（camera-rendered，不算 edited）
cp _base.jpg image_camera_rendered_sony.jpg
exiftool -overwrite_original \
  -Make='SONY' \
  -Model='ILCE-7RM5' \
  -Software='Imaging Edge Desktop 1.2.00.13110' \
  -DateTimeOriginal='2024:03:15 14:23:01' \
  -CreateDate='2024:03:15 14:23:01' \
  -ModifyDate='2024:03:15 14:23:15' \
  image_camera_rendered_sony.jpg >/dev/null

# 4) image_no_exif.png —— PNG screenshot（无 EXIF）
ffmpeg -y -f lavfi -i "color=blue:size=320x240" -frames:v 1 image_no_exif.png 2>/dev/null

rm -f _base.jpg

echo "Generated:"
ls -lh audio.mp3 audio.flac audio.wav video.mp4 \
       image_original_sony.jpg image_edited_lightroom.jpg \
       image_camera_rendered_sony.jpg image_no_exif.png
