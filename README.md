# clips

Monitors Twitch chat for viral moments, creates clips, and edits them into vertical short-form video. Extensible to other sources in the future.

## Pipeline

```
Chat (IRC) → Detection Engine → Clip API → yt-dlp → ffmpeg edit
```

1. **Chat monitor** — connects to Twitch IRC, streams messages to the detector
2. **Detection engine** — per-channel rolling window, scores activity (velocity, unique users, emote density), emits a trigger when score >= 1.0
3. **Clip creation** — calls Twitch Helix API to create a clip of the streamer
4. **Download** — fetches the clip video via yt-dlp
5. **Edit** — processes into 1080x1920 vertical format with blurred background and optional title overlay

## Setup

```bash
make setup         # install tools + githooks
# fill in config.yaml with your Twitch client_id and channel list
go run ./cmd/clips
```

First run triggers the Twitch device-code OAuth flow — open the URL shown and enter the code. The token is cached in `data/tokens/token.json` and auto-refreshes.

## Requirements

- Go 1.24+
- [ffmpeg](https://ffmpeg.org/) (tested with version 6+)
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)

## Configuration

`config.yaml`:

| Field | Description |
|-------|-------------|
| `client_id` | Twitch application client ID |
| `channels` | List of channels to monitor |
| `thresholds.messages_per_second` | Velocity threshold for scoring |
| `thresholds.unique_users` | Unique user threshold for scoring |
| `thresholds.emotes_per_window` | Emote count threshold for scoring |
| `thresholds.cooldown_seconds` | Minimum seconds between triggers per channel |
| `thresholds.evaluation_window` | Rolling window in seconds |

## Development

```bash
make check    # fmt → vet → lint → build → test
```

Pre-commit hook runs `make check` on staged `.go` files. Skip with `SKIP=1`.
