#!/usr/bin/env python3
"""Wingman — Textual TUI for updating software across Windows."""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

from textual import work
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, Vertical
from textual.message import Message
from textual.widgets import DataTable, Footer, Header, Label, ProgressBar, RichLog, Static

from wingman_sources import (
    Package,
    PkgSource,
    PkgStatus,
    discover_all,
    upgrade_package,
    upgrade_sort_key,
)

# Re-export for shortcut helper placement
__all__ = ["AppUpdater", "main"]


class ScanComplete(Message):
    def __init__(self, packages: list[Package], shortcuts: list[Package]) -> None:
        self.packages = packages
        self.shortcuts = shortcuts
        super().__init__()


class UpdateComplete(Message):
    def __init__(self, ok: int, fail: int) -> None:
        self.ok = ok
        self.fail = fail
        super().__init__()


class AppUpdater(App):
    TITLE = "Wingman"
    SUB_TITLE = "winget · Store · choco · npm · pip · Steam · Windows · more"

    CSS = """
    Screen { background: #0d1117; }
    #hero {
        height: auto;
        padding: 1 2;
        background: #161b22;
        border: tall #30363d;
        margin: 1 2 0 2;
    }
    #stats {
        dock: top;
        height: 1;
        padding: 0 2;
        color: #8b949e;
        background: #010409;
    }
    #main { height: 1fr; margin: 0 2; }
    #table-pane { width: 2fr; border: round #30363d; min-height: 12; }
    #log-pane { width: 1fr; border: round #30363d; min-height: 12; }
    DataTable { height: 1fr; }
    RichLog { height: 1fr; background: #010409; }
    #progress-wrap { height: auto; padding: 0 2 1 2; }
    ProgressBar { margin-top: 1; }
    """

    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("r", "refresh", "Refresh"),
        Binding("u", "update_selected", "Update selected"),
        Binding("a", "update_all", "Update all auto"),
        Binding("space", "toggle_row", "Toggle row"),
        Binding("s", "toggle_shortcuts", "Shortcuts"),
        Binding("i", "toggle_inventory", "Inventory (ARP)"),
    ]

    def __init__(self, auto_all: bool = False, include_shortcuts: bool = True) -> None:
        super().__init__()
        self.auto_all = auto_all
        self.include_shortcuts = include_shortcuts
        self.packages: list[Package] = []
        self.shortcuts: list[Package] = []
        self.view_mode = "updates"  # updates | shortcuts | arp
        self._updating = False

    def compose(self) -> ComposeResult:
        yield Header()
        yield Static(
            "[bold]Wingman[/]  —  updates everything we can reach on this PC.\n"
            "[dim]winget · Microsoft Store · Chocolatey · npm · pip · Steam · "
            "Foundry · Windows Update · + ARP inventory[/]",
            id="hero",
        )
        yield Static("Scanning…", id="stats")
        with Horizontal(id="main"):
            with Vertical(id="table-pane"):
                yield DataTable(id="pkg-table", zebra_stripes=True, cursor_type="row")
            with Vertical(id="log-pane"):
                yield Label("Activity log", id="log-label")
                yield RichLog(id="log", highlight=True, markup=True)
        with Vertical(id="progress-wrap"):
            yield ProgressBar(id="progress", show_eta=False)
        yield Footer()

    def on_mount(self) -> None:
        table = self.query_one("#pkg-table", DataTable)
        table.add_columns("✓", "Name", "Current", "Available", "Source", "Status")
        self.query_one("#progress", ProgressBar).display = False
        self.action_refresh()

    def _set_stats(self, text: str) -> None:
        self.query_one("#stats", Static).update(text)

    def _log(self, msg: str) -> None:
        self.query_one("#log", RichLog).write(msg)

    def _visible_rows(self) -> list[Package]:
        if self.view_mode == "shortcuts":
            return self.shortcuts
        if self.view_mode == "arp":
            return [p for p in self.packages if p.source == PkgSource.ARP]
        return [p for p in self.packages if p.source != PkgSource.ARP]

    @work(thread=True)
    def _scan_worker(self) -> None:
        managed, shortcuts = discover_all(self.include_shortcuts)
        self.post_message(ScanComplete(managed, shortcuts))

    def on_scan_complete(self, event: ScanComplete) -> None:
        self.packages = event.packages
        self.shortcuts = event.shortcuts
        self._fill_table()
        up = sum(1 for p in self.packages if p.can_auto_update)
        by_source: dict[str, int] = {}
        for p in self.packages:
            by_source[p.source.value] = by_source.get(p.source.value, 0) + 1
        parts = ", ".join(f"{k}:{v}" for k, v in sorted(by_source.items()))
        arp = sum(1 for p in self.packages if p.source == PkgSource.ARP)
        self._set_stats(
            f"  {len(self.packages)} items · [bold]{up}[/] auto-upgradable · "
            f"ARP inventory {arp} · [dim]{parts}[/]"
        )
        self._log(
            f"[green]Scan complete.[/] {up} auto-upgrade(s). "
            f"[dim]s[/]=shortcuts · [dim]i[/]=ARP inventory"
        )
        if self.auto_all and up:
            self.action_update_all()

    def _fill_table(self) -> None:
        table = self.query_one("#pkg-table", DataTable)
        table.clear()
        for pkg in self._visible_rows():
            mark = "☑" if pkg.selected else "☐"
            if not pkg.can_auto_update:
                mark = "·"
            status = pkg.status.value
            if pkg.source == PkgSource.SHORTCUT:
                status = "launcher"
            elif pkg.source == PkgSource.ARP:
                status = "manual"
            table.add_row(
                mark,
                pkg.name[:40],
                pkg.current[:14],
                pkg.available[:14],
                pkg.source.value,
                status,
                key=pkg.package_id,
            )

    def action_refresh(self) -> None:
        if self._updating:
            return
        self._set_stats("  Scanning all sources (this can take a minute)…")
        self._log("[dim]winget, choco, npm, pip, Steam, Windows Update, registry…[/]")
        self._scan_worker()

    def action_toggle_row(self) -> None:
        table = self.query_one("#pkg-table", DataTable)
        if table.cursor_row is None:
            return
        rows = self._visible_rows()
        if table.cursor_row >= len(rows):
            return
        pkg = rows[table.cursor_row]
        if not pkg.can_auto_update:
            self._log(f"[dim]{pkg.name}[/] — manual / use {pkg.source.value}")
            return
        pkg.selected = not pkg.selected
        row = table.cursor_row
        self._fill_table()
        table.move_cursor(row=row)

    def action_toggle_shortcuts(self) -> None:
        self.view_mode = "shortcuts" if self.view_mode != "shortcuts" else "updates"
        self._fill_table()
        self._log(f"[dim]View: {self.view_mode}[/]")

    def action_toggle_inventory(self) -> None:
        self.view_mode = "arp" if self.view_mode != "arp" else "updates"
        self._fill_table()
        n = sum(1 for p in self.packages if p.source == PkgSource.ARP)
        self._log(f"[dim]ARP inventory ({n} programs, manual update)[/]")

    def _selected_upgradable(self) -> list[Package]:
        return [p for p in self.packages if p.selected and p.can_auto_update]

    @work(thread=True)
    def _update_worker(self, packages: list[Package]) -> None:
        ok = fail = 0
        packages = sorted(packages, key=upgrade_sort_key)
        total = len(packages)
        for i, pkg in enumerate(packages, start=1):
            pkg.status = PkgStatus.WORKING
            self.call_from_thread(self._progress, i, total, pkg.name)

            def log(msg: str) -> None:
                self.call_from_thread(self._log, msg)

            success = upgrade_package(pkg, log)
            pkg.status = PkgStatus.OK if success else PkgStatus.FAIL
            if success:
                ok += 1
            else:
                fail += 1
        self.post_message(UpdateComplete(ok, fail))

    def _progress(self, current: int, total: int, name: str) -> None:
        bar = self.query_one("#progress", ProgressBar)
        bar.display = True
        bar.update(total=total, progress=current - 1)
        self._set_stats(f"  ({current}/{total}) {name[:50]}")

    def on_update_complete(self, event: UpdateComplete) -> None:
        self._updating = False
        bar = self.query_one("#progress", ProgressBar)
        bar.update(total=1, progress=1)
        bar.display = False
        self._log(
            f"\n[bold]Done.[/] [green]{event.ok} ok[/], [red]{event.fail} failed[/]. "
            "Press [bold]r[/] to rescan."
        )
        self.action_refresh()

    def _start_updates(self, packages: list[Package]) -> None:
        if self._updating:
            self._log("[yellow]Already updating.[/]")
            return
        if not packages:
            self._log("[yellow]Nothing selected.[/]")
            return
        self._updating = True
        self._log(f"[bold]Upgrading {len(packages)} item(s)…[/]")
        self._update_worker(packages)

    def action_update_selected(self) -> None:
        self._start_updates(self._selected_upgradable())

    def action_update_all(self) -> None:
        all_pkgs = [p for p in self.packages if p.can_auto_update]
        for p in all_pkgs:
            p.selected = True
        self._start_updates(all_pkgs)


def create_start_menu_shortcut(
    bat_path: Path,
    shortcut_name: str = "Wingman",
    icon_path: Path | None = None,
) -> Path:
    import win32com.client  # type: ignore

    if icon_path is None:
        icon_path = bat_path.parent / "assets" / "wingman.ico"
    icon_loc = (
        f"{icon_path.resolve()},0"
        if icon_path.is_file()
        else f"{sys.executable},0"
    )

    candidates = [
        Path(os.environ.get("ProgramData", r"C:\ProgramData"))
        / "Microsoft"
        / "Windows"
        / "Start Menu"
        / "Programs",
        Path(os.environ.get("APPDATA", ""))
        / "Microsoft"
        / "Windows"
        / "Start Menu"
        / "Programs",
    ]

    shell = win32com.client.Dispatch("WScript.Shell")
    last_error: Exception | None = None

    for programs in candidates:
        if not str(programs):
            continue
        try:
            programs.mkdir(parents=True, exist_ok=True)
            lnk_path = programs / f"{shortcut_name}.lnk"
            shortcut = shell.CreateShortcut(str(lnk_path))
            shortcut.TargetPath = str(bat_path.resolve())
            shortcut.WorkingDirectory = str(bat_path.parent.resolve())
            shortcut.IconLocation = icon_loc
            shortcut.Description = "Wingman — upgrade software across this PC"
            shortcut.save()
            return lnk_path
        except Exception as exc:  # noqa: BLE001
            last_error = exc
            continue

    raise RuntimeError("Could not create Start Menu shortcut") from last_error


def main() -> int:
    parser = argparse.ArgumentParser(description="Wingman — update software on this PC")
    parser.add_argument("-y", "--yes-all", action="store_true", help="Upgrade all and exit")
    parser.add_argument(
        "--install-shortcut",
        nargs="?",
        const="Wingman",
        metavar="NAME",
        help='Create Start Menu shortcut (default: "Wingman")',
    )
    args = parser.parse_args()

    script_dir = Path(__file__).resolve().parent
    bat_path = script_dir / "wingman.bat"

    if args.install_shortcut is not None:
        name = args.install_shortcut or "Wingman"
        path = create_start_menu_shortcut(bat_path, shortcut_name=name)
        print(f"Created shortcut: {path}")
        return 0

    if args.yes_all:
        packages, _ = discover_all(include_shortcuts=False)
        targets = sorted(
            [p for p in packages if p.can_auto_update],
            key=upgrade_sort_key,
        )
        ok = fail = 0
        for pkg in targets:
            print(f"Upgrading [{pkg.source.value}] {pkg.name}…")
            if upgrade_package(pkg, print):
                ok += 1
            else:
                fail += 1
        print(f"Done: {ok} ok, {fail} failed.")
        return 0 if fail == 0 else 1

    AppUpdater().run()
    return 0


if __name__ == "__main__":
    sys.exit(main())
