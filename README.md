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

## Releases (alpha)

Wingman is in **alpha**. Releases are pre-releases only, tagged with [semver pre-release](https://semver.org/#spec-item-9) IDs:

`v0.1.0-alpha.1`, `v0.1.0-alpha.2`, …

Every push to `master` (or `main`) that includes new commits since the latest alpha tag triggers **Release** in GitHub Actions. The workflow picks the next tag from [conventional commits](https://www.conventionalcommits.org/) since the last release:

| Commits since last tag | Next tag example |
| --- | --- |
| `fix:`, `docs:`, `chore:`, etc. | `v0.1.0-alpha.1` → `v0.1.0-alpha.2` (alpha serial only) |
| `feat:` | `v0.1.0-alpha.2` → `v0.2.0-alpha.1` (minor base bump, alpha reset) |
| `BREAKING CHANGE` / `!` | same as feat while on `0.x` |

Add `[skip release]` anywhere in the commit message to skip one push. No local `git tag` or `gh` CLI is required.

GitHub Actions builds `wingman.exe` on Windows and attaches it to a **pre-release** on GitHub.

To rebuild assets for an **existing** alpha tag, use **Actions → Release → Run workflow** and enter the tag name.

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
