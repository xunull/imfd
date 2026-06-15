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

# 5) image_ai_c2pa.jpg —— 真实 C2PA 签名 JPEG（AI 生成检测验证）
# 需要 c2patool（brew install c2patool）。这是检测真实 JUMBF/CBOR 结构的关键 fixture——
# 合成 fixture 是 self-fulfilling oracle，真签名文件才能验证 byte-scan 在野外有效。
# 仓库已 checked-in 一份；c2patool 在则重新生成（保持新鲜）。
if command -v c2patool >/dev/null 2>&1; then
  ffmpeg -y -f lavfi -i "color=green:size=320x240" -frames:v 1 _c2pa_base.jpg 2>/dev/null
  cat > _c2pa_manifest.json <<'JSON'
{
  "claim_generator_info": [{ "name": "imfd-test-generator", "version": "1.0" }],
  "assertions": [
    { "label": "c2pa.actions", "data": { "actions": [{ "action": "c2pa.created" }] } }
  ]
}
JSON
  c2patool _c2pa_base.jpg -m _c2pa_manifest.json -o image_ai_c2pa.jpg --force >/dev/null 2>&1 \
    && echo "regenerated image_ai_c2pa.jpg via c2patool" \
    || echo "WARN: c2patool sign failed, keeping checked-in image_ai_c2pa.jpg"
  rm -f _c2pa_base.jpg _c2pa_manifest.json
else
  echo "NOTE: c2patool not installed — keeping checked-in image_ai_c2pa.jpg"
  echo "      (install via 'brew install c2patool' to regenerate)"
fi

echo "Generated:"
ls -lh audio.mp3 audio.flac audio.wav video.mp4 \
       image_original_sony.jpg image_edited_lightroom.jpg \
       image_camera_rendered_sony.jpg image_no_exif.png image_ai_c2pa.jpg
