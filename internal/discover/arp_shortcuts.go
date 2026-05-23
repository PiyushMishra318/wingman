package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/piyushmishra318/wingman/internal/exec"
	"github.com/piyushmishra318/wingman/internal/model"
	"golang.org/x/sys/windows/registry"
)

var skipARP = []string{"update for", "redistributable", "runtime", "proof", "language pack"}

func ARP(known map[string]bool) []model.Package {
	knownNorm := make(map[string]bool)
	for n := range known {
		knownNorm[normName(n)] = true
	}
	roots := []struct {
		hive registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
	}

	seen := map[string]bool{}
	var pkgs []model.Package

	for _, root := range roots {
		key, err := registry.OpenKey(root.hive, root.path, registry.READ)
		if err != nil {
			continue
		}
		subs, _ := key.ReadSubKeyNames(0)
		key.Close()
		for _, sub := range subs {
			sk, err := registry.OpenKey(root.hive, root.path+`\`+sub, registry.READ)
			if err != nil {
				continue
			}
			display, _, err := sk.GetStringValue("DisplayName")
			if err != nil || display == "" || seen[display] {
				sk.Close()
				continue
			}
			version, _, _ := sk.GetStringValue("DisplayVersion")
			publisher, _, _ := sk.GetStringValue("Publisher")
			sk.Close()

			low := strings.ToLower(display)
			if knownNorm[normName(display)] {
				continue
			}
			skip := false
			for _, bit := range skipARP {
				if strings.Contains(low, bit) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			seen[display] = true
			if len(display) > 70 {
				display = display[:70]
			}
			if len(version) > 20 {
				version = version[:20]
			}
			pub := publisher
			if len(pub) > 40 {
				pub = pub[:40]
			}
			pkgs = append(pkgs, model.Package{
				Name: display, ID: sub, Current: version, Available: "manual",
				Source: model.SourceARP, Status: model.StatusManual, Selected: false, Detail: pub,
			})
		}
	}
	return pkgs
}

type shortcutItem struct {
	Name   string `json:"Name"`
	Path   string `json:"Path"`
	Target string `json:"Target"`
	Args   string `json:"Args"`
	Folder string `json:"Folder"`
}

func StartMenuShortcuts() []model.Package {
	script := `
$dirs = @(
  "$env:ProgramData\Microsoft\Windows\Start Menu\Programs",
  "$env:APPDATA\Microsoft\Windows\Start Menu\Programs"
)
$sh = New-Object -ComObject WScript.Shell
$seen = @{}
$list = @()
foreach ($d in $dirs) {
  if (-not (Test-Path $d)) { continue }
  Get-ChildItem -Path $d -Filter *.lnk -Recurse -ErrorAction SilentlyContinue | ForEach-Object {
    $lnk = $sh.CreateShortcut($_.FullName)
    $key = ($lnk.TargetPath + '|' + $lnk.Arguments).ToLower()
    if ($seen[$key]) { return }
    $seen[$key] = $true
    if (-not $lnk.TargetPath) { return }
    $list += [PSCustomObject]@{
      Name = $_.BaseName; Path = $_.FullName; Target = $lnk.TargetPath
      Args = $lnk.Arguments; Folder = $_.Directory.Name
    }
  }
}
$list | ConvertTo-Json -Compress
`
	out, _, err := exec.PowerShell(2*time.Minute, script)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	var items []shortcutItem
	if err := unmarshalJSONArray(out, &items); err != nil || len(items) == 0 {
		var one shortcutItem
		if json.Unmarshal([]byte(out), &one) == nil && one.Path != "" {
			items = []shortcutItem{one}
		}
	}
	var pkgs []model.Package
	for _, it := range items {
		target := it.Target
		if it.Args != "" {
			target += " " + it.Args
		}
		pkgs = append(pkgs, model.Package{
			Name: it.Name, ID: it.Path, Current: filepath.Base(it.Target), Available: "—",
			Source: model.SourceShortcut, Status: model.StatusManual, Target: target,
			Selected: false, Detail: it.Folder,
		})
	}
	return pkgs
}

func unmarshalJSONArray[T any](out string, dest *[]T) error {
	return json.Unmarshal([]byte(out), dest)
}

func startMenuDirs() []string {
	var dirs []string
	if p := os.Getenv("ProgramData"); p != "" {
		dirs = append(dirs, filepath.Join(p, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if p := os.Getenv("APPDATA"); p != "" {
		dirs = append(dirs, filepath.Join(p, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	return dirs
}
