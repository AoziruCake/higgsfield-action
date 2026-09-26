# Releasing

## Prerequisites

- CI is green on `main`
- Repository secret `HIGGSFIELD_API_KEY` is configured (for manual integration runs)
- [Integration workflow](.github/workflows/integration.yml) succeeded at least once on `main`

## Publish `v0.1.0` and `v1`

```bash
git checkout main
git pull
git tag -a v0.1.0 -m "First public MVP: Soul v2 image generation"
git tag -a v1 -m "v1 tracks the latest 1.x release"
git push origin v0.1.0 v1
```

Pushing version tags runs [.github/workflows/release.yml](.github/workflows/release.yml) (tests + Docker build).

## After the tag

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

## Pre-release checklist

- [ ] `go test ./...` passes locally
- [ ] `docker build .` succeeds
- [ ] Integration workflow (`workflow_dispatch`) completed with a real API key
- [ ] README usage example matches `action.yml` inputs/outputs
- [ ] No secrets or credentials in the repository
