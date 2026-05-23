# Wingman

Terminal UI to discover and upgrade software on Windows — winget, Chocolatey, npm, pip, Steam, Windows Update, and ARP “manual” entries.

## Requirements

- Windows 10/11
- [Go 1.23+](https://go.dev/dl/)
- Optional package managers: `winget`, `choco`, `npm`, `pip`, Steam

## Quick start

```bat
wingman.bat
```

Build a release binary:

```bat
build.bat
```

Non-interactive upgrade of everything that supports auto-update:

```bat
wingman.exe -y
```

Create a Start Menu shortcut:

```bat
wingman.exe -install-shortcut
```

## Development

```bash
go test ./...
go test ./internal/discover -bench=Merge -benchmem
go run ./cmd/scanbench    # parallel scan timing
go run ./cmd/wingman      # TUI
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Releases

Publishing a release is tag-only (no `gh` CLI required locally):

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

GitHub Actions builds `wingman.exe` on Windows and attaches it to the release.

To attach assets to an **existing** tag (for example `v1.0.0` was pushed before CI existed), either:

- In GitHub: **Actions → Release → Run workflow**, enter the tag name, or
- Re-push the tag with git: `git push origin :refs/tags/v1.0.0` then `git push origin v1.0.0`

## Project layout

```
cmd/wingman/       Main TUI and CLI flags
cmd/scanbench/     Discovery scan benchmark
internal/discover/ Parallel source scanners + merge
internal/model/    Package types and ordering
internal/upgrade/  Per-source upgrade runners
internal/tui/      Bubble Tea interface
legacy/            Retired Python prototype (not required to build)
```

## License

MIT — see [LICENSE](LICENSE). Change the copyright line if you fork.
