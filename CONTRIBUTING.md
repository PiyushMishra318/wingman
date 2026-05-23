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

Wingman ships as **alpha pre-releases** only. Use semver pre-release tags: `v0.1.0-alpha.1`, `v0.1.0-alpha.2`, etc. (must match `v*-alpha*` for CI).

Maintainers publish with git only:

```bash
git tag v0.1.0-alpha.1
git push origin v0.1.0-alpha.1
```

CI (`.github/workflows/release.yml`) creates a GitHub **pre-release** and uploads `wingman.exe`. To rebuild assets for a tag that already exists, use **Actions → Release → Run workflow** and enter the tag, or delete and re-push the tag on the remote.

When sharing logs or screenshots, redact machine-specific paths and usernames (for example `C:\Users\…`). Do not commit local paths or secrets in docs or tests.
