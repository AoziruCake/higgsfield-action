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

Text-to-video uses [Seedance 2.5](https://docs.higgsfield.ai/docs/models/seedance-2-5/text-to-video). Other video endpoints are rejected before submission.

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

Pin a release with `@v0.1.0` if you need a fixed revision. `@v1` is the floating major tag (see [RELEASING.md](RELEASING.md)).

### Secrets

Create a repository secret `HIGGSFIELD_API_KEY` with your credentials in **`KEY_ID:KEY_SECRET`** format ([Authentication](https://docs.higgsfield.ai/docs/authentication.md)).

### Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `api-key` | yes | — | `KEY_ID:KEY_SECRET` |
| `prompt` | yes | — | Text prompt |
| `output` | yes | — | Output file path (workspace-relative) |
| `model` | no | `higgsfield-ai/soul/v2/standard` | Model endpoint path. Video: `bytedance/seedance-2.5/text-to-video` |
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

## Development

```bash
go test ./...
go build -o higgsfield-action ./cmd/higgsfield-action
docker build -t higgsfield-action .
```

### Integration test (GitHub)

1. Add secret `HIGGSFIELD_API_KEY` to the repository.
2. Run **Actions → Integration → Run workflow** ([integration.yml](.github/workflows/integration.yml)).
3. Download the artifact from the completed run. Image runs upload `higgsfield-image`. Video runs upload `higgsfield-video`.

The workflow dispatch form accepts `model` and `duration`. Leave them at the defaults for an image. For a video run, set `model` to `bytedance/seedance-2.5/text-to-video` and `output` to a path such as `output/integration.mp4`. `duration` can stay empty (the Action uses 5 seconds).

### Troubleshooting

| Error | What to do |
|-------|------------|
| `api-key is required` | The Action now reads Docker's `INPUT_API-KEY`. Update to a commit that includes that fix and rerun. |
| `not enough credits` / HTTP 403 | The API key is valid, but the Higgsfield account has no remaining credits. Add credits in [Higgsfield Cloud](https://console.higgsfield.ai) and rerun. |
| `create output directory: ... permission denied` | The Action container must run as root so it can write to `GITHUB_WORKSPACE`. Update to a commit that drops Distroless `:nonroot` and rerun. |
| Later step cannot find `output-path` | `output-path` is workspace-relative (not `/github/workspace/...`). Use `$GITHUB_WORKSPACE/${{ steps.higgsfield.outputs.output-path }}` or the `output` input. |

## Releasing

See [RELEASING.md](RELEASING.md) for tagging `v1` / `v0.1.0` and Marketplace notes.

## Support

For bugs, questions, and feature requests about **this Action**, please open a [GitHub Issue](https://github.com/AoziruCake/higgsfield-action/issues) and pick a template (**Bug report**, **Question**, or **Feature request**). That is the primary support channel.

Links (not a substitute for Issues on this repo):

- Website: [sugirep.com](https://sugirep.com)
- X: [@sugirep](https://x.com/sugirep)

## License

[MIT](LICENSE)
