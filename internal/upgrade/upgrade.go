package upgrade

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	xexec "github.com/piyushmishra318/wingman/internal/exec"
	"github.com/piyushmishra318/wingman/internal/model"
)

type Logger func(string)

func Package(pkg model.Package, log Logger) bool {
	switch pkg.Source {
	case model.SourceWinget, model.SourceMSStore:
		return winget(pkg, log)
	case model.SourceChoco:
		return choco(pkg, log)
	case model.SourceNPM:
		return npm(pkg, log)
	case model.SourcePIP:
		return pip(pkg, log)
	case model.SourceSteam:
		return steam(pkg, log)
	case model.SourceFoundry:
		return foundry(pkg, log)
	case model.SourceWinUpdate:
		return windowsUpdate(pkg, log)
	default:
		return false
	}
}

func SortPackages(pkgs []model.Package) {
	sort.Slice(pkgs, func(i, j int) bool {
		oi, oj := model.UpgradeOrder(pkgs[i].Source), model.UpgradeOrder(pkgs[j].Source)
		if oi != oj {
			return oi < oj
		}
		return strings.ToLower(pkgs[i].Name) < strings.ToLower(pkgs[j].Name)
	})
}

func winget(pkg model.Package, log Logger) bool {
	log(fmt.Sprintf("winget: %s", pkg.Name))
	_, _, err := xexec.RunTimeout(15*time.Minute, "winget", "upgrade", "--id", pkg.ID, "-e",
		"--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity", "-h")
	return err == nil
}

func choco(pkg model.Package, log Logger) bool {
	log(fmt.Sprintf("choco: %s", pkg.Name))
	_, _, err := xexec.RunTimeout(15*time.Minute, "choco", "upgrade", pkg.ID, "-y", "--timeout", "900")
	return err == nil
}

func npm(pkg model.Package, log Logger) bool {
	npm := xexec.Which("npm")
	if npm == "" {
		return false
	}
	log(fmt.Sprintf("npm: %s", pkg.Name))
	_, _, err := xexec.RunTimeout(10*time.Minute, npm, "install", "-g", pkg.ID+"@latest")
	return err == nil
}

func pip(pkg model.Package, log Logger) bool {
	pip := xexec.Which("pip")
	if pip == "" {
		return false
	}
	if pkg.ID == "__pip_all__" {
		log("pip: upgrading all outdated…")
		_, _, _ = xexec.RunTimeout(2*time.Minute, pip, "install", "--upgrade", "pip")
		out, _, _ := xexec.RunTimeout(3*time.Minute, pip, "list", "--outdated", "--format=json")
		var items []struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal([]byte(out), &items)
		ok := true
		for _, it := range items {
			if it.Name == "" {
				continue
			}
			_, _, err := xexec.RunTimeout(5*time.Minute, pip, "install", "--upgrade", it.Name)
			if err != nil {
				ok = false
			}
		}
		return ok
	}
	log(fmt.Sprintf("pip: %s", pkg.Name))
	_, _, err := xexec.RunTimeout(5*time.Minute, pip, "install", "--upgrade", pkg.ID)
	return err == nil
}

func steam(pkg model.Package, log Logger) bool {
	const steamExe = `C:\Program Files (x86)\Steam\steam.exe`
	if pkg.ID == "steam-client" {
		log("steam: silent update pass")
		cmd := exec.Command(steamExe, "-silent")
		_ = cmd.Start()
		return true
	}
	log(fmt.Sprintf("steam: %s", pkg.Name))
	return exec.Command("cmd", "/c", "start", "", "steam://update/"+pkg.ID).Start() == nil
}

func foundry(pkg model.Package, log Logger) bool {
	log("foundry: launching updater")
	cmd := exec.Command(pkg.ID)
	return cmd.Start() == nil
}

func windowsUpdate(pkg model.Package, log Logger) bool {
	log("windows: installing updates")
	var script string
	if pkg.ID == "wu-all" {
		script = `
$ErrorActionPreference = 'Stop'
$s = New-Object -ComObject Microsoft.Update.Session
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0")
$coll = New-Object -ComObject Microsoft.Update.UpdateColl
for ($i = 0; $i -lt $r.Updates.Count; $i++) { [void]$coll.Add($r.Updates.Item($i)) }
if ($coll.Count -eq 0) { 'NONE'; exit 0 }
$installer = $s.CreateUpdateInstaller()
$installer.Updates = $coll
$r = $installer.Install()
"RESULT=$($r.ResultCode)"
`
	} else {
		idx := strings.TrimPrefix(pkg.ID, "wu-")
		script = fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$s = New-Object -ComObject Microsoft.Update.Session
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0")
$coll = New-Object -ComObject Microsoft.Update.UpdateColl
[void]$coll.Add($r.Updates.Item(%s))
$installer = $s.CreateUpdateInstaller()
$installer.Updates = $coll
$r = $installer.Install()
"RESULT=$($r.ResultCode)"
`, idx)
	}
	out, _, err := xexec.PowerShell(60*time.Minute, script)
	if err != nil {
		return false
	}
	return strings.Contains(out, "RESULT=2") || strings.Contains(out, "RESULT=3") || strings.Contains(out, "NONE")
}

func InstallShortcut(batPath, name, iconPath string) (string, error) {
	script := fmt.Sprintf(`
$sh = New-Object -ComObject WScript.Shell
$dirs = @("$env:ProgramData\Microsoft\Windows\Start Menu\Programs", "$env:APPDATA\Microsoft\Windows\Start Menu\Programs")
foreach ($d in $dirs) {
  try {
    New-Item -ItemType Directory -Force -Path $d | Out-Null
    $lnk = Join-Path $d '%s.lnk'
    $s = $sh.CreateShortcut($lnk)
    $s.TargetPath = '%s'
    $s.WorkingDirectory = '%s'
    $s.IconLocation = '%s,0'
    $s.Description = 'Wingman — upgrade software on this PC'
    $s.Save()
    Write-Output $lnk
    exit 0
  } catch {}
}
exit 1
`, strings.ReplaceAll(name, "'", "''"),
		strings.ReplaceAll(batPath, "'", "''"),
		strings.ReplaceAll(filepathDir(batPath), "'", "''"),
		strings.ReplaceAll(iconPath, "'", "''"))
	out, _, err := xexec.PowerShell(30*time.Second, script)
	return strings.TrimSpace(out), err
}

func filepathDir(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[:i]
	}
	return p
}
