# Releasing

## Prerequisites

- CI is green on `main`
- Repository secret `HIGGSFIELD_API_KEY` is configured (for manual integration runs)
- [Integration workflow](.github/workflows/integration.yml) succeeded at least once on `main`

## Publish `v1.2.0`

`prompt-file` is verified. [CHANGELOG.md](CHANGELOG.md) already lists it under 1.2.0. From `main`:

```bash
git checkout main
git pull
git tag -a v1.2.0 -m "Add a prompt-file input"
git push origin v1.2.0
git tag -f v1
git push -f origin v1
```

Pushing `v1.2.0` runs [.github/workflows/release.yml](.github/workflows/release.yml) (tests and Docker build). Then draft the GitHub Release from the 1.2.0 notes. Consumers keep using `@v1`. Pin `@v1.2.0` only when a workflow must not pick up a later 1.x tag.

The manual prompt-file trial workflow is not part of this release.

## Publish `v0.1.0` and `v1`

Already published. Kept here as the original release record.

```bash
git checkout main
git pull
git tag -a v0.1.0 -m "First public MVP: Soul v2 image generation"
git tag -a v1 -m "v1 tracks the latest 1.x release"
git push origin v0.1.0 v1
```

Pushing version tags runs [.github/workflows/release.yml](.github/workflows/release.yml) (tests + Docker build).

## After the first tag

These steps were for `v0.1.0`. Later releases draft the GitHub Release from [CHANGELOG.md](CHANGELOG.md) and move `v1` as in the 1.2.0 steps above.

1. Open **GitHub → Releases → Draft a new release** for `v0.1.0`.
2. Summarize: Soul v2 image generation, Docker Action, inputs/outputs documented in [README.md](README.md).
3. Consumers should use the floating major tag:

   ```yaml
   uses: AoziruCake/higgsfield-action@v1
   ```

   Pin `@v0.1.0` only when a workflow must not pick up later `v1.x` tags.

4. After each later `v1.x.x` release, move `v1` to that commit ([GitHub Actions versioning](https://docs.github.com/en/actions/sharing-automations/creating-actions/about-custom-actions#using-tags-for-release-management)):

   ```bash
   git tag -f v1
   git push -f origin v1
   ```

## Check the Action before tagging

```bash
go test ./...
go build -o higgsfield-action ./cmd/higgsfield-action
docker build -t higgsfield-action .
```

Integration on this repository:

1. Add secret `HIGGSFIELD_API_KEY`.
2. Run **Actions → Integration → Run workflow** ([integration.yml](.github/workflows/integration.yml)).
3. Download the artifact. Image runs upload `higgsfield-image`. Video runs upload `higgsfield-video`.

The form sends `prompt`. Leave `model` and `duration` at the defaults for an image. For a video run, set `model` to `bytedance/seedance-2.5/text-to-video` and `output` to a path such as `output/integration.mp4`. `duration` can stay empty.

`prompt-file` is covered by the caller examples in [README.md](README.md). The integration form does not send it.

## Pre-release checklist

- [ ] `go test ./...` passes locally
- [ ] `docker build .` succeeds
- [ ] Integration workflow (`workflow_dispatch`) completed with a real API key
- [ ] README usage example matches `action.yml` inputs/outputs
- [ ] [CHANGELOG.md](CHANGELOG.md) moves the Unreleased notes under the new version
- [ ] No secrets or credentials in the repository
