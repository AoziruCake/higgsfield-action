# higgsfield-action

Generate images and videos with the [Higgsfield API](https://docs.higgsfield.ai/) from GitHub Actions.

Docker Container Action (Go) for **Soul v2 Standard** text-to-image and **Seedance 2.5** text-to-video: submit, poll, download, and write outputs.

Maintained by **[Sugirep](https://sugirep.com)** — [AoziruCake/higgsfield-action](https://github.com/AoziruCake/higgsfield-action) on GitHub.

This Action calls a **generative AI** model (Higgsfield Soul v2, or Seedance 2.5 for video). Outputs are not guaranteed to be accurate, appropriate, or fit for any purpose. Generation uses **the caller's** Higgsfield account: their [terms](https://docs.higgsfield.ai/), credits, and content moderation apply. This Action does not warrant prompts or generated media.

Support is via [GitHub Issues](https://github.com/AoziruCake/higgsfield-action/issues). The software is licensed under [MIT](LICENSE).

## Usage

```yaml
- name: Generate image
  uses: AoziruCake/higgsfield-action@v1
  with:
    api-key: ${{ secrets.HIGGSFIELD_API_KEY }}
    prompt: "A cat programming in Go"
    output: output.png
```

Text-to-video uses [Seedance 2.5](https://docs.higgsfield.ai/docs/models/seedance-2-5/text-to-video). Only that video endpoint and Soul v2 Standard are accepted.

```yaml
- name: Generate video
  uses: AoziruCake/higgsfield-action@v1
  with:
    api-key: ${{ secrets.HIGGSFIELD_API_KEY }}
    prompt: "A cat programming in Go, cinematic camera move"
    model: bytedance/seedance-2.5/text-to-video
    output: output.mp4
    timeout: 20m
```

`aspect-ratio` defaults to `16:9`, `resolution` to `720p`, and `duration` to `5` when `model` is Seedance 2.5 text-to-video. `duration` is rejected for image models.

`@v1` follows the latest 1.x release. What that release contains is listed in [CHANGELOG.md](CHANGELOG.md). Pin a full version from that list, such as `@v1.2.0`, when a workflow should stay on one release.

Set `prompt` or `prompt-file`, not both. `prompt-file` is a path inside your repository, relative to the workspace, to a UTF-8 text file of at most 16 KiB. The Action trims that file and uses the text as the prompt. A workflow that passes only `prompt` stays valid.

### Recipes

The Action only submits, waits, and downloads. Push, pull request, and cron triggers, artifact upload, and chat notifications belong in the calling workflow. Copy an example into that repository's `.github/workflows/` directory. These files do not run in this repository. Every generation spends the caller's Higgsfield credits. A cron schedule spends credits on every tick, even when nothing changed.

- [Generate when a prompt file changes](examples/generate-on-push.yml) — push or pull request that touches `prompts/**`, using `prompt-file`.
- [Daily generation](examples/daily-cron.yml) — `schedule` plus `workflow_dispatch`. Run it manually before leaving the schedule on.
- [Save the file and notify](examples/save-and-notify.yml) — `actions/upload-artifact`, then a Discord webhook. Slack and a pull request comment are noted in the file.

### Secrets

Create a repository secret `HIGGSFIELD_API_KEY` with your credentials in **`KEY_ID:KEY_SECRET`** format ([Authentication](https://docs.higgsfield.ai/docs/authentication.md)).

### Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `api-key` | yes | — | `KEY_ID:KEY_SECRET` |
| `prompt` | no | — | Text prompt. Set this or `prompt-file`, not both |
| `prompt-file` | no | — | Workspace-relative UTF-8 text file used as the prompt (16 KiB max). Set this or `prompt`, not both |
| `output` | yes | — | Output file path (workspace-relative) |
| `model` | no | `higgsfield-ai/soul/v2/standard` | `higgsfield-ai/soul/v2/standard` or `bytedance/seedance-2.5/text-to-video` |
| `aspect-ratio` | no | `4:3` for images, `16:9` for Seedance 2.5 | Soul v2 accepts its usual ratios. Seedance 2.5 accepts `16:9`, `4:3`, `1:1`, `3:4`, `9:16`, `21:9` |
| `resolution` | no | `720p` | Images: `720p` or `1080p`. Seedance 2.5 also accepts `480p` |
| `duration` | no | `5` for Seedance 2.5 | Seconds, `4`–`30`. Video only |
| `timeout` | no | `10m` | Max wait time (Go duration). Use `20m` or more for video |

### Outputs

| Output | Description |
|--------|-------------|
| `request-id` | Higgsfield request ID |
| `image-url` | Generated image CDN URL (image runs) |
| `video-url` | Generated video CDN URL (Seedance 2.5 runs) |
| `output-path` | Workspace-relative path of the saved file |

## Troubleshooting

| Error | What to do |
|-------|------------|
| `api-key is required` | Create the repository secret `HIGGSFIELD_API_KEY` and pass it as `api-key`. |
| `prompt or prompt-file is required` / `set prompt or prompt-file, not both` | Pass one of the two inputs. |
| `not enough credits` / HTTP 403 | The API key is valid, and the Higgsfield account has no remaining credits. Add credits in [Higgsfield Cloud](https://console.higgsfield.ai) and rerun. |
| `create output directory: ... permission denied` | Use `@v1` or a 1.x release from [CHANGELOG.md](CHANGELOG.md). Older builds cannot write the workspace. |
| Later step cannot find `output-path` | `output-path` is relative to the workspace. Use `$GITHUB_WORKSPACE/${{ steps.higgsfield.outputs.output-path }}` or the same path you passed as `output`. |

## Changes

See [CHANGELOG.md](CHANGELOG.md).

## Support

For bugs, questions, and feature requests about **this Action**, please open a [GitHub Issue](https://github.com/AoziruCake/higgsfield-action/issues) and pick a template (**Bug report**, **Question**, or **Feature request**). That is the primary support channel.

Links (not a substitute for Issues on this repo):

- Website: [sugirep.com](https://sugirep.com)
- X: [@sugirep](https://x.com/sugirep)

## License

[MIT](LICENSE)
