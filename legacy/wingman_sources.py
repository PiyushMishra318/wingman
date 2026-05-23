"""Discovery and upgrade logic for all Wingman sources."""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import winreg
from dataclasses import dataclass
from enum import Enum
from pathlib import Path
from typing import Callable

WINGET = os.environ.get("WINGET", "winget")
CHOCO = os.environ.get("CHOCO", "choco")
STEAM_EXE = Path(
    os.environ.get("STEAM_EXE", r"C:\Program Files (x86)\Steam\steam.exe")
)
FOUNDRY_UPDATER = Path(os.environ.get("LOCALAPPDATA", "")) / "foundryvtt-updater" / "installer.exe"
FOUNDRY_VTT = Path(os.environ.get("FOUNDRY_VTT", r"D:\foundry-pi\Foundry\foundryvtt"))

START_MENU_DIRS = [
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

AUTO_SOURCES = frozenset(
    {
        "winget",
        "msstore",
        "choco",
        "npm",
        "pip",
        "steam",
        "foundry",
        "winupdate",
        "scoop",
    }
)


class PkgStatus(str, Enum):
    UPGRADABLE = "upgrade"
    CURRENT = "current"
    MANUAL = "manual"
    SKIPPED = "skipped"
    WORKING = "working"
    OK = "ok"
    FAIL = "fail"


class PkgSource(str, Enum):
    WINGET = "winget"
    MSSTORE = "msstore"
    CHOCO = "choco"
    NPM = "npm"
    PIP = "pip"
    STEAM = "steam"
    FOUNDRY = "foundry"
    WINUPDATE = "winupdate"
    SCOOP = "scoop"
    ARP = "arp"
    SHORTCUT = "shortcut"


@dataclass
class Package:
    name: str
    package_id: str
    current: str
    available: str
    source: PkgSource
    status: PkgStatus = PkgStatus.UPGRADABLE
    target: str = ""
    selected: bool = True
    detail: str = ""

    @property
    def can_auto_update(self) -> bool:
        if self.source == PkgSource.ARP:
            return False
        if self.source == PkgSource.SHORTCUT:
            return False
        if self.source == PkgSource.PIP and self.detail == "bulk":
            return True
        return self.source.value in AUTO_SOURCES and self.status in (
            PkgStatus.UPGRADABLE,
            PkgStatus.FAIL,
        )


def _run(cmd: list[str], timeout: int = 600) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=timeout,
        creationflags=subprocess.CREATE_NO_WINDOW if sys.platform == "win32" else 0,
    )


def _run_ps(script: str, timeout: int = 1800) -> subprocess.CompletedProcess[str]:
    return _run(
        [
            "powershell",
            "-NoProfile",
            "-ExecutionPolicy",
            "Bypass",
            "-Command",
            script,
        ],
        timeout=timeout,
    )


def _column_slices(header: str) -> list[slice]:
    labels = ["Name", "Id", "Version", "Available", "Source"]
    positions: list[int] = []
    for label in labels:
        idx = header.find(label)
        if idx >= 0:
            positions.append(idx)
    if len(positions) < 3:
        return [slice(0, 36), slice(36, 72), slice(72, 92), slice(92, 108), slice(108, None)]
    positions.append(len(header) + 40)
    return [slice(positions[i], positions[i + 1]) for i in range(len(positions) - 1)]


def _parse_winget_table(stdout: str, require_available: bool) -> list[Package]:
    lines = stdout.splitlines()
    header_idx = None
    for i, line in enumerate(lines):
        if "Name" in line and "Id" in line and "Version" in line:
            header_idx = i
            break
    if header_idx is None:
        return []

    sep_idx = header_idx + 1
    while sep_idx < len(lines) and not re.match(r"^-{3,}", lines[sep_idx]):
        sep_idx += 1
    if sep_idx >= len(lines):
        return []

    cols = _column_slices(lines[header_idx])
    packages: list[Package] = []

    for line in lines[sep_idx + 1 :]:
        if not line.strip() or line.startswith("-"):
            continue
        if len(line) < 10:
            continue

        def col(n: int) -> str:
            if n >= len(cols):
                return ""
            return line[cols[n]].strip()

        name = col(0)
        pkg_id = col(1)
        current = col(2)
        available = col(3)
        source_raw = (col(4) if len(cols) > 4 else "winget").lower()

        if not name or not pkg_id:
            continue
        if require_available and (not available or available == current):
            continue
        if pkg_id.startswith("ARP\\") or "Steam App" in pkg_id:
            continue

        source = PkgSource.MSSTORE if source_raw == "msstore" else PkgSource.WINGET
        packages.append(
            Package(
                name=name,
                package_id=pkg_id,
                current=current or "?",
                available=available or "?",
                source=source,
                status=PkgStatus.UPGRADABLE,
            )
        )
    return packages


def discover_winget_upgrades() -> list[Package]:
    result = _run(
        [
            WINGET,
            "upgrade",
            "--include-unknown",
            "--accept-source-agreements",
            "--disable-interactivity",
        ],
        timeout=300,
    )
    return _parse_winget_table(result.stdout + result.stderr, require_available=True)


def discover_choco_outdated() -> list[Package]:
    result = _run([CHOCO, "outdated", "--limit-output", "--timeout", "300"], timeout=300)
    packages: list[Package] = []
    for line in result.stdout.splitlines():
        line = line.strip()
        if not line or "|" not in line:
            continue
        parts = line.split("|")
        if len(parts) < 3:
            continue
        name, current, available = parts[0], parts[1], parts[2]
        if current == available:
            continue
        packages.append(
            Package(
                name=name,
                package_id=name,
                current=current,
                available=available,
                source=PkgSource.CHOCO,
            )
        )
    return packages


def _tool_cmd(name: str, env_key: str) -> list[str] | None:
    exe = shutil.which(os.environ.get(env_key, name))
    if not exe:
        return None
    return [exe]


def discover_npm_global() -> list[Package]:
    cmd = _tool_cmd("npm", "NPM")
    if not cmd:
        return []
    result = _run([*cmd, "outdated", "-g", "--json"], timeout=120)
    if not result.stdout.strip():
        return []
    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        return []
    packages: list[Package] = []
    for name, info in data.items():
        if not isinstance(info, dict):
            continue
        packages.append(
            Package(
                name=name,
                package_id=name,
                current=str(info.get("current", "?")),
                available=str(info.get("latest", info.get("wanted", "?"))),
                source=PkgSource.NPM,
            )
        )
    return packages


def discover_pip_outdated() -> list[Package]:
    cmd = _tool_cmd("pip", "PIP")
    if not cmd:
        return []
    result = _run([*cmd, "list", "--outdated", "--format=json"], timeout=180)
    if not result.stdout.strip():
        return []
    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError:
        return []
    packages: list[Package] = []
    for item in data:
        name = item.get("name")
        if not name:
            continue
        packages.append(
            Package(
                name=name,
                package_id=name,
                current=str(item.get("version", "?")),
                available=str(item.get("latest_version", "?")),
                source=PkgSource.PIP,
                selected=False,
                detail="bulk",
            )
        )
    if packages:
        packages.insert(
            0,
            Package(
                name=f"pip — all outdated ({len(packages)} packages)",
                package_id="__pip_all__",
                current=str(len(packages)),
                available="upgrade all",
                source=PkgSource.PIP,
                detail="bulk",
            ),
        )
    return packages


def discover_scoop_outdated() -> list[Package]:
    scoop = os.environ.get("SCOOP", "scoop")
    if not shutil.which(scoop):
        return []
    result = _run([scoop, "status"], timeout=120)
    packages: list[Package] = []
    for line in result.stdout.splitlines():
        line = line.strip()
        if not line or line.startswith("Scoop") or ":" not in line:
            continue
        # "app: Update available (1.0 -> 2.0)"
        m = re.match(r"^(\S+):\s+Update available\s+\((.+)\s+->\s+(.+)\)", line)
        if m:
            name, current, available = m.group(1), m.group(2), m.group(3)
            packages.append(
                Package(
                    name=name,
                    package_id=name,
                    current=current,
                    available=available,
                    source=PkgSource.SCOOP,
                )
            )
    return packages


def _steam_library_dirs() -> list[Path]:
    dirs: list[Path] = []
    default = STEAM_EXE.parent / "steamapps"
    if default.is_dir():
        dirs.append(default)
    vdf = default / "libraryfolders.vdf"
    if vdf.is_file():
        text = vdf.read_text(encoding="utf-8", errors="replace")
        for raw in re.findall(r'"path"\s+"([^"]+)"', text):
            path = Path(raw.replace("\\\\", "\\")) / "steamapps"
            if path.is_dir() and path not in dirs:
                dirs.append(path)
    return dirs


def _parse_steam_acf(acf: Path) -> tuple[str, str] | None:
    text = acf.read_text(encoding="utf-8", errors="replace")
    appid_m = re.search(r'"appid"\s+"(\d+)"', text)
    name_m = re.search(r'"name"\s+"([^"]+)"', text)
    if not appid_m or not name_m:
        return None
    return appid_m.group(1), name_m.group(1)


def discover_steam() -> list[Package]:
    packages: list[Package] = []
    if STEAM_EXE.is_file():
        packages.append(
            Package(
                name="Steam — update client & all games",
                package_id="steam-client",
                current="installed",
                available="run Steam updater",
                source=PkgSource.STEAM,
                detail="Launches Steam silent update pass",
            )
        )

    seen: set[str] = set()
    for lib in _steam_library_dirs():
        for acf in lib.glob("appmanifest_*.acf"):
            parsed = _parse_steam_acf(acf)
            if not parsed:
                continue
            appid, name = parsed
            if appid in seen:
                continue
            seen.add(appid)
            packages.append(
                Package(
                    name=name,
                    package_id=appid,
                    current=f"appid {appid}",
                    available="via Steam",
                    source=PkgSource.STEAM,
                    selected=False,
                    detail=str(acf.parent),
                )
            )
    return packages


def discover_foundry() -> list[Package]:
    if not FOUNDRY_UPDATER.is_file():
        return []
    current = "?"
    pkg_json = FOUNDRY_VTT / "package.json"
    if pkg_json.is_file():
        try:
            current = json.loads(pkg_json.read_text(encoding="utf-8")).get("version", "?")
        except (json.JSONDecodeError, OSError):
            pass
    return [
        Package(
            name="Foundry VTT",
            package_id=str(FOUNDRY_UPDATER),
            current=current,
            available="run updater",
            source=PkgSource.FOUNDRY,
            detail=str(FOUNDRY_VTT),
        )
    ]


def discover_windows_updates() -> list[Package]:
    script = r"""
$ErrorActionPreference = 'Stop'
$s = New-Object -ComObject Microsoft.Update.Session
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0")
$list = @()
foreach ($i in 0..($r.Updates.Count - 1)) {
    $u = $r.Updates.Item($i)
    $list += [PSCustomObject]@{
        Index = $i
        Title = $u.Title
        KB = ($u.KBArticleIDs | Select-Object -First 1)
    }
}
$list | ConvertTo-Json -Compress
"""
    result = _run_ps(script, timeout=300)
    out = result.stdout.strip()
    if not out or out == "null":
        return []
    try:
        data = json.loads(out)
    except json.JSONDecodeError:
        return []
    if isinstance(data, dict):
        data = [data]
    packages: list[Package] = []
    for item in data:
        title = str(item.get("Title", "Windows Update"))[:80]
        idx = item.get("Index", 0)
        kb = item.get("KB", "")
        packages.append(
            Package(
                name=title,
                package_id=f"wu-{idx}",
                current="pending",
                available=kb or "install",
                source=PkgSource.WINUPDATE,
                selected=False,
            )
        )
    if packages:
        packages.insert(
            0,
            Package(
                name=f"Windows Update — install all ({len(packages)} pending)",
                package_id="wu-all",
                current=str(len(packages)),
                available="install",
                source=PkgSource.WINUPDATE,
                detail="May require admin; can reboot",
            ),
        )
    return packages


def discover_arp_programs(known_names: set[str]) -> list[Package]:
    """Installed programs from Add/Remove Programs not covered elsewhere."""
    packages: list[Package] = []
    seen: set[str] = set()
    roots = [
        (winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"),
        (winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"),
        (winreg.HKEY_CURRENT_USER, r"SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"),
    ]

    def norm(s: str) -> str:
        return re.sub(r"[^a-z0-9]+", "", s.lower())

    known_norm = {norm(n) for n in known_names}

    for hive, path in roots:
        try:
            key = winreg.OpenKey(hive, path)
        except OSError:
            continue
        i = 0
        while True:
            try:
                sub = winreg.EnumKey(key, i)
                i += 1
            except OSError:
                break
            try:
                sk = winreg.OpenKey(key, sub)
                name, _ = winreg.QueryValueEx(sk, "DisplayName")
                try:
                    version, _ = winreg.QueryValueEx(sk, "DisplayVersion")
                except OSError:
                    version = "?"
                try:
                    publisher, _ = winreg.QueryValueEx(sk, "Publisher")
                except OSError:
                    publisher = ""
                winreg.CloseKey(sk)
            except OSError:
                continue

            if not name or name in seen:
                continue
            if norm(name) in known_norm:
                continue
            skip_bits = (
                "update for",
                "redistributable",
                "runtime",
                "proof",
                "language pack",
            )
            if any(b in name.lower() for b in skip_bits):
                continue

            seen.add(name)
            packages.append(
                Package(
                    name=str(name)[:70],
                    package_id=sub,
                    current=str(version)[:20],
                    available="manual",
                    source=PkgSource.ARP,
                    status=PkgStatus.MANUAL,
                    selected=False,
                    detail=str(publisher)[:40],
                )
            )

    packages.sort(key=lambda p: p.name.lower())
    return packages


def _read_shortcut(path: Path) -> tuple[str, str]:
    try:
        import win32com.client  # type: ignore

        shell = win32com.client.Dispatch("WScript.Shell")
        shortcut = shell.CreateShortcut(str(path))
        return shortcut.TargetPath or "", shortcut.Arguments or ""
    except Exception:
        return "", ""


def discover_start_menu_shortcuts() -> list[Package]:
    seen: set[str] = set()
    shortcuts: list[Package] = []
    for root in START_MENU_DIRS:
        if not root.is_dir():
            continue
        for lnk in root.rglob("*.lnk"):
            try:
                if lnk.stat().st_size == 0:
                    continue
            except OSError:
                continue
            target, args = _read_shortcut(lnk)
            if not target:
                continue
            key = f"{target}|{args}".lower()
            if key in seen:
                continue
            seen.add(key)
            target_display = f"{target} {args}".strip() if args else target
            shortcuts.append(
                Package(
                    name=lnk.stem,
                    package_id=str(lnk),
                    current=Path(target).name,
                    available="—",
                    source=PkgSource.SHORTCUT,
                    status=PkgStatus.MANUAL,
                    target=target_display,
                    selected=False,
                    detail=str(lnk.parent.name),
                )
            )
    shortcuts.sort(key=lambda p: p.name.lower())
    return shortcuts


def merge_packages(*groups: list[Package]) -> list[Package]:
    by_key: dict[str, Package] = {}
    for group in groups:
        for pkg in group:
            if pkg.source in (PkgSource.ARP, PkgSource.SHORTCUT):
                key = f"{pkg.source.value}:{pkg.package_id}"
            else:
                key = f"{pkg.source.value}:{pkg.package_id.lower()}"
                name_key = pkg.name.lower()
                if any(
                    name_key == existing.name.lower()
                    for existing in by_key.values()
                    if existing.source not in (PkgSource.ARP, PkgSource.SHORTCUT)
                ):
                    continue
            by_key[key] = pkg

    merged = list(by_key.values())
    order = {
        PkgSource.WINUPDATE: 0,
        PkgSource.WINGET: 1,
        PkgSource.MSSTORE: 2,
        PkgSource.CHOCO: 3,
        PkgSource.NPM: 4,
        PkgSource.PIP: 5,
        PkgSource.SCOOP: 6,
        PkgSource.STEAM: 7,
        PkgSource.FOUNDRY: 8,
        PkgSource.ARP: 9,
        PkgSource.SHORTCUT: 10,
    }
    merged.sort(
        key=lambda p: (
            order.get(p.source, 99),
            not p.can_auto_update,
            p.name.lower(),
        )
    )
    return merged


def discover_all(include_shortcuts: bool = True) -> tuple[list[Package], list[Package]]:
    winget = discover_winget_upgrades()
    choco = discover_choco_outdated()
    npm = discover_npm_global()
    pip = discover_pip_outdated()
    scoop = discover_scoop_outdated()
    steam = discover_steam()
    foundry = discover_foundry()
    winupdate = discover_windows_updates()

    known = {p.name for p in winget + choco + npm + steam + foundry}
    arp = discover_arp_programs(known)

    managed = merge_packages(winget, choco, npm, pip, scoop, steam, foundry, winupdate, arp)
    shortcuts = discover_start_menu_shortcuts() if include_shortcuts else []
    return managed, shortcuts


# ---------------------------------------------------------------------------
# Upgrades
# ---------------------------------------------------------------------------


def upgrade_winget(pkg: Package, log: Callable[[str], None]) -> bool:
    log(f"[cyan]winget[/] {pkg.name}…")
    result = _run(
        [
            WINGET,
            "upgrade",
            "--id",
            pkg.package_id,
            "-e",
            "--accept-package-agreements",
            "--accept-source-agreements",
            "--disable-interactivity",
            "-h",
        ],
        timeout=900,
    )
    ok = result.returncode == 0
    log(f"[green]✓[/] {pkg.name}" if ok else f"[red]✗[/] {pkg.name} ({result.returncode})")
    return ok


def upgrade_choco(pkg: Package, log: Callable[[str], None]) -> bool:
    log(f"[yellow]choco[/] {pkg.name}…")
    result = _run([CHOCO, "upgrade", pkg.package_id, "-y", "--timeout", "900"], timeout=900)
    ok = result.returncode == 0
    log(f"[green]✓[/] {pkg.name}" if ok else f"[red]✗[/] {pkg.name} ({result.returncode})")
    return ok


def upgrade_npm(pkg: Package, log: Callable[[str], None]) -> bool:
    cmd = _tool_cmd("npm", "NPM")
    if not cmd:
        return False
    log(f"[#cb3837]npm[/] {pkg.name}…")
    result = _run([*cmd, "install", "-g", f"{pkg.package_id}@latest"], timeout=600)
    ok = result.returncode == 0
    log(f"[green]✓[/] {pkg.name}" if ok else f"[red]✗[/] {pkg.name}")
    return ok


def upgrade_pip(pkg: Package, log: Callable[[str], None]) -> bool:
    cmd = _tool_cmd("pip", "PIP")
    if not cmd:
        return False
    if pkg.package_id == "__pip_all__":
        log("[#3776ab]pip[/] upgrading all outdated packages…")
        result = _run([*cmd, "install", "--upgrade", "pip"], timeout=120)
        if result.returncode != 0:
            log("[yellow]pip self-upgrade skipped[/]")
        result = _run([*cmd, "list", "--outdated", "--format=json"], timeout=180)
        try:
            items = json.loads(result.stdout or "[]")
        except json.JSONDecodeError:
            items = []
        ok = True
        for item in items:
            name = item.get("name")
            if not name:
                continue
            log(f"  [dim]pip[/] {name}")
            r = _run([*cmd, "install", "--upgrade", name], timeout=300)
            if r.returncode != 0:
                ok = False
        log("[green]✓ pip bulk[/]" if ok else "[red]✗ pip bulk (some failed)[/]")
        return ok

    log(f"[#3776ab]pip[/] {pkg.name}…")
    result = _run([*cmd, "install", "--upgrade", pkg.package_id], timeout=300)
    ok = result.returncode == 0
    log(f"[green]✓[/] {pkg.name}" if ok else f"[red]✗[/] {pkg.name}")
    return ok


def upgrade_scoop(pkg: Package, log: Callable[[str], None]) -> bool:
    scoop = os.environ.get("SCOOP", "scoop")
    log(f"[#79b8ff]scoop[/] {pkg.name}…")
    result = _run([scoop, "update", pkg.package_id], timeout=600)
    ok = result.returncode == 0
    log(f"[green]✓[/] {pkg.name}" if ok else f"[red]✗[/] {pkg.name}")
    return ok


def upgrade_steam(pkg: Package, log: Callable[[str], None]) -> bool:
    if not STEAM_EXE.is_file():
        log("[red]Steam not found[/]")
        return False
    if pkg.package_id == "steam-client":
        log("[#1b2838]steam[/] launching silent update pass…")
        subprocess.Popen(
            [str(STEAM_EXE), "-silent"],
            creationflags=subprocess.CREATE_NO_WINDOW if sys.platform == "win32" else 0,
        )
        log("[green]✓[/] Steam started — games update in background")
        return True
    log(f"[#1b2838]steam[/] queue update for {pkg.name}…")
    os.startfile(f"steam://update/{pkg.package_id}")
    log(f"[green]✓[/] opened steam://update/{pkg.package_id}")
    return True


def upgrade_foundry(pkg: Package, log: Callable[[str], None]) -> bool:
    log("[#ff6a00]foundry[/] launching official updater…")
    if not Path(pkg.package_id).is_file():
        log("[red]foundryvtt-updater not found[/]")
        return False
    subprocess.Popen([pkg.package_id])
    log("[green]✓[/] Foundry updater opened — finish in its window")
    return True


def upgrade_windows_update(pkg: Package, log: Callable[[str], None]) -> bool:
    log("[#0078d4]windows[/] installing updates (may need admin)…")
    if pkg.package_id == "wu-all":
        script = r"""
$ErrorActionPreference = 'Stop'
$s = New-Object -ComObject Microsoft.Update.Session
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0")
$coll = New-Object -ComObject Microsoft.Update.UpdateColl
for ($i = 0; $i -lt $r.Updates.Count; $i++) { [void]$coll.Add($r.Updates.Item($i)) }
if ($coll.Count -eq 0) { Write-Output 'NONE'; exit 0 }
$installer = $s.CreateUpdateInstaller()
$installer.Updates = $coll
$result = $installer.Install()
Write-Output ("RESULT=" + $result.ResultCode)
"""
    else:
        idx = pkg.package_id.replace("wu-", "")
        script = f"""
$ErrorActionPreference = 'Stop'
$s = New-Object -ComObject Microsoft.Update.Session
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0")
$coll = New-Object -ComObject Microsoft.Update.UpdateColl
[void]$coll.Add($r.Updates.Item({idx}))
$installer = $s.CreateUpdateInstaller()
$installer.Updates = $coll
$result = $installer.Install()
Write-Output ("RESULT=" + $result.ResultCode)
"""
    result = _run_ps(script, timeout=3600)
    out = (result.stdout + result.stderr).strip()
    ok = "RESULT=2" in out or "RESULT=3" in out or result.returncode == 0
    if "NONE" in out:
        log("[dim]no pending updates[/]")
        return True
    for line in out.splitlines()[-5:]:
        log(f"  {line}")
    log("[green]✓ Windows Update[/]" if ok else "[red]✗ Windows Update (try Run as admin)[/]")
    return ok


def upgrade_package(pkg: Package, log: Callable[[str], None]) -> bool:
    handlers = {
        PkgSource.WINGET: upgrade_winget,
        PkgSource.MSSTORE: upgrade_winget,
        PkgSource.CHOCO: upgrade_choco,
        PkgSource.NPM: upgrade_npm,
        PkgSource.PIP: upgrade_pip,
        PkgSource.SCOOP: upgrade_scoop,
        PkgSource.STEAM: upgrade_steam,
        PkgSource.FOUNDRY: upgrade_foundry,
        PkgSource.WINUPDATE: upgrade_windows_update,
    }
    handler = handlers.get(pkg.source)
    if handler:
        return handler(pkg, log)
    return False


def upgrade_sort_key(pkg: Package) -> tuple[int, str]:
    order = {
        PkgSource.NPM: 0,
        PkgSource.PIP: 1,
        PkgSource.CHOCO: 2,
        PkgSource.WINGET: 3,
        PkgSource.MSSTORE: 4,
        PkgSource.SCOOP: 5,
        PkgSource.FOUNDRY: 6,
        PkgSource.STEAM: 7,
        PkgSource.WINUPDATE: 8,
    }
    return (order.get(pkg.source, 50), pkg.name.lower())
