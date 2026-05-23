# Contributing

Thanks for helping improve Wingman.

## Setup

- Windows 10/11 for full manual testing (discovery and upgrades call platform tools).
- [Go 1.23+](https://go.dev/dl/)

## Checks

```bash
go test ./...
go build -o wingman.exe ./cmd/wingman
```

Use the issue and pull request templates when opening work on GitHub.

## Releases (alpha)

Wingman ships as **alpha pre-releases** only (`v0.1.0-alpha.1`, `v0.1.0-alpha.2`, …). Merging or pushing to `master`/`main` runs `.github/workflows/release.yml`, which tags the next alpha version from commits since the last tag (`fix:` → alpha serial bump, `feat:` / breaking → minor base bump on `0.x`). Use `[skip release]` in a commit message to skip one release.

CI creates a GitHub **pre-release** and uploads `wingman.exe`. To rebuild assets for an existing tag, use **Actions → Release → Run workflow** and enter the tag.

When sharing logs or screenshots, redact machine-specific paths and usernames (for example `C:\Users\…`). Do not commit local paths or secrets in docs or tests.
