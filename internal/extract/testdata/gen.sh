#!/usr/bin/env bash
# 生成集成测试 fixture。需要 ffmpeg。
set -euo pipefail

cd "$(dirname "$0")"

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "ffmpeg not installed; aborting" >&2
  exit 1
fi

ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" -b:a 192k audio.mp3 2>/dev/null
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" audio.flac 2>/dev/null
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" audio.wav 2>/dev/null
ffmpeg -y -f lavfi -i "testsrc=duration=1:size=320x240:rate=30" -pix_fmt yuv420p video.mp4 2>/dev/null

echo "Generated:"
ls -lh audio.mp3 audio.flac audio.wav video.mp4
