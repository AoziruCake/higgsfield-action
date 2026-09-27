# higgsfield-action

Generate images with the [Higgsfield API](https://docs.higgsfield.ai/) from GitHub Actions.

Docker Container Action (Go) for **Soul v2 Standard** text-to-image: submit, poll, download, and write outputs.

Maintained by **[Sugirep](https://sugirep.com)** — [AoziruCake/higgsfield-action](https://github.com/AoziruCake/higgsfield-action) on GitHub.

## Usage

```yaml
- name: Generate image
  uses: AoziruCake/higgsfield-action@v1
  with:
    api-key: ${{ secrets.HIGGSFIELD_API_KEY }}
    prompt: "A cat programming in Go"
    output: output.png
```

Pin a release with `@v0.1.0` if you need a fixed revision. `@v1` is the floating major tag (see [RELEASING.md](RELEASING.md)).

### Secrets

Create a repository secret `HIGGSFIELD_API_KEY` with your credentials in **`KEY_ID:KEY_SECRET`** format ([Authentication](https://docs.higgsfield.ai/docs/authentication.md)).

### Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `api-key` | yes | — | `KEY_ID:KEY_SECRET` |
| `prompt` | yes | — | Text prompt |
| `output` | yes | — | Output file path (workspace-relative) |
| `model` | no | `higgsfield-ai/soul/v2/standard` | Model endpoint path |
| `aspect-ratio` | no | `4:3` | Soul v2 aspect ratio |
| `resolution` | no | `720p` | `720p` or `1080p` |
| `timeout` | no | `10m` | Max wait time (Go duration) |

### Outputs

| Output | Description |
|--------|-------------|
| `request-id` | Higgsfield request ID |
| `image-url` | Generated image CDN URL |
| `output-path` | Absolute path where the image was saved |

## Development

```bash
go test ./...
go build -o higgsfield-action ./cmd/higgsfield-action
docker build -t higgsfield-action .
```

### Integration test (GitHub)

1. Add secret `HIGGSFIELD_API_KEY` to the repository.
2. Run **Actions → Integration → Run workflow** ([integration.yml](.github/workflows/integration.yml)).
3. Download the `higgsfield-image` artifact from the completed run.

### Troubleshooting

| Error | What to do |
|-------|------------|
| `api-key is required` | The Action now reads Docker's `INPUT_API-KEY`. Update to a commit that includes that fix and rerun. |
| `not enough credits` / HTTP 403 | The API key is valid, but the Higgsfield account has no remaining credits. Add credits in [Higgsfield Cloud](https://console.higgsfield.ai) and rerun. |

## Releasing

See [RELEASING.md](RELEASING.md) for tagging `v1` / `v0.1.0` and Marketplace notes.

## Support

For bugs, questions, and feature requests about **this Action**, please open a [GitHub Issue](https://github.com/AoziruCake/higgsfield-action/issues) and pick a template (**Bug report**, **Question**, or **Feature request**). That is the primary support channel.

Links (not a substitute for Issues on this repo):

- Website: [sugirep.com](https://sugirep.com)
- X: [@sugirep](https://x.com/sugirep)

## License

[MIT](LICENSE)
