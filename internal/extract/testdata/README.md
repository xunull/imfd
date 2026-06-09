# Test fixtures

集成测试（`go test -tags=integration ./internal/extract/`）需要这里的样本文件。
**这些文件不进 git**（见 `.gitignore`）——本地或 CI 里现场用 ffmpeg 生成。

## 生成命令

```bash
cd internal/extract/testdata

# audio: 1 秒 440Hz 正弦波，分别压成 mp3/flac/wav
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" -b:a 192k audio.mp3
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" audio.flac
ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" audio.wav

# video: 1 秒 320x240 测试图样
ffmpeg -y -f lavfi -i testsrc=duration=1:size=320x240:rate=30 -pix_fmt yuv420p video.mp4
```

或直接跑：

```bash
./gen.sh
```

文件总大小 ~30KB，生成耗时 <1s。

## CI

如需在 CI 跑集成测试：

```yaml
- run: sudo apt-get install -y ffmpeg
- run: ./internal/extract/testdata/gen.sh
- run: go test -tags=integration ./internal/extract/
```
