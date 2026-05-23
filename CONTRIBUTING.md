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

When sharing logs or screenshots, redact machine-specific paths and usernames (for example `C:\Users\…`). Do not commit local paths or secrets in docs or tests.
