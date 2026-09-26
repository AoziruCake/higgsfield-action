# Releasing

## Prerequisites

- CI is green on `main`
- Repository secret `HIGGSFIELD_API_KEY` is configured (for manual integration runs)
- [Integration workflow](.github/workflows/integration.yml) succeeded at least once on `main`

## Publish `v0.1.0`

```bash
git checkout main
git pull
git tag -a v0.1.0 -m "First public MVP: Soul v2 image generation"
git push origin v0.1.0
```

Pushing the tag runs [.github/workflows/release.yml](.github/workflows/release.yml) (tests + Docker build).

## After the tag

1. Open **GitHub → Releases → Draft a new release** for `v0.1.0`.
2. Summarize: Soul v2 image generation, Docker Action, inputs/outputs documented in [README.md](README.md).
3. For Marketplace listing, point consumers at:

   ```yaml
   uses: AoziruCake/higgsfield-action@v0.1.0
   ```

4. When you are ready for a floating major tag, move `v1` to the latest `v1.x.x` commit ([GitHub Actions versioning](https://docs.github.com/en/actions/sharing-automations/creating-actions/about-custom-actions#using-tags-for-release-management)).

## Pre-release checklist

- [ ] `go test ./...` passes locally
- [ ] `docker build .` succeeds
- [ ] Integration workflow (`workflow_dispatch`) completed with a real API key
- [ ] README usage example matches `action.yml` inputs/outputs
- [ ] No secrets or credentials in the repository
