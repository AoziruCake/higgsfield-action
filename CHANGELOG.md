# Changelog

Notable changes to this Action. The floating tag `v1` points at the latest 1.x release.

## [Unreleased]

## [1.2.0] - 2026-11-01

### Added

- `prompt-file` input. Pass a workspace-relative UTF-8 text file (16 KiB maximum) and the Action uses the trimmed text as the prompt. Set `prompt` or `prompt-file`, not both. Workflows that pass only `prompt` behave as before.
- Example workflows for a push, a daily schedule, and saving the file plus a Discord notification. Copy them from `examples/` into the calling repository. They do not run in this repository.

## [1.1.1] - 2026-10-16

### Fixed

- Output paths stay inside the workspace, and the API key is sent only to the Higgsfield host.
- An unknown video model fails before the duration check.
- Video integration runs upload the artifact as `higgsfield-video`.

## [1.1.0] - 2026-10-12

### Added

- Seedance 2.5 text-to-video (`bytedance/seedance-2.5/text-to-video`), including `duration` and the `video-url` output.

## [0.1.0] - 2026-09-27

### Added

- Soul v2 Standard text-to-image from GitHub Actions: submit, poll, download, and write the file.
