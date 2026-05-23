package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/piyushmishra318/wingman/internal/exec"
	"github.com/piyushmishra318/wingman/internal/model"
)

var (
	steamExe       = `C:\Program Files (x86)\Steam\steam.exe`
	appIDRe        = regexp.MustCompile(`"appid"\s+"(\d+)"`)
	appNameRe      = regexp.MustCompile(`"name"\s+"([^"]+)"`)
	libPathRe      = regexp.MustCompile(`"path"\s+"([^"]+)"`)
	foundryUpdater = filepath.Join(os.Getenv("LOCALAPPDATA"), "foundryvtt-updater", "installer.exe")
)

func foundryInstallDir() string {
	if p := os.Getenv("WINGMAN_FOUNDRY_DIR"); p != "" {
		return p
	}
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "FoundryVTT")
}

func Steam() []model.Package {
	var pkgs []model.Package
	if _, err := os.Stat(steamExe); err == nil {
		pkgs = append(pkgs, model.Package{
			Name: "Steam — update client & all games", ID: "steam-client",
			Current: "installed", Available: "run Steam updater",
			Source: model.SourceSteam, Status: model.StatusUpgrade, Selected: true,
			Detail: "Launches Steam silent update pass",
		})
	}
	seen := map[string]bool{}
	for _, lib := range steamLibraryDirs() {
		matches, _ := filepath.Glob(filepath.Join(lib, "appmanifest_*.acf"))
		for _, acf := range matches {
			b, err := os.ReadFile(acf)
			if err != nil {
				continue
			}
			text := string(b)
			idM := appIDRe.FindStringSubmatch(text)
			nameM := appNameRe.FindStringSubmatch(text)
			if len(idM) < 2 || len(nameM) < 2 {
				continue
			}
			if seen[idM[1]] {
				continue
			}
			seen[idM[1]] = true
			pkgs = append(pkgs, model.Package{
				Name: nameM[1], ID: idM[1], Current: "appid " + idM[1], Available: "via Steam",
				Source: model.SourceSteam, Status: model.StatusUpgrade, Selected: false, Detail: filepath.Dir(acf),
			})
		}
	}
	return pkgs
}

func steamLibraryDirs() []string {
	defaultDir := filepath.Join(filepath.Dir(steamExe), "steamapps")
	var dirs []string
	if st, err := os.Stat(defaultDir); err == nil && st.IsDir() {
		dirs = append(dirs, defaultDir)
	}
	vdf := filepath.Join(defaultDir, "libraryfolders.vdf")
	b, err := os.ReadFile(vdf)
	if err != nil {
		return dirs
	}
	for _, m := range libPathRe.FindAllStringSubmatch(string(b), -1) {
		p := filepath.Join(strings.ReplaceAll(m[1], `\\`, `\`), "steamapps")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			found := false
			for _, d := range dirs {
				if strings.EqualFold(d, p) {
					found = true
					break
				}
			}
			if !found {
				dirs = append(dirs, p)
			}
		}
	}
	return dirs
}

func Foundry() []model.Package {
	if _, err := os.Stat(foundryUpdater); err != nil {
		return nil
	}
	current := "?"
	installDir := foundryInstallDir()
	if b, err := os.ReadFile(filepath.Join(installDir, "package.json")); err == nil {
		var pj struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(b, &pj) == nil && pj.Version != "" {
			current = pj.Version
		}
	}
	return []model.Package{{
		Name: "Foundry VTT", ID: foundryUpdater, Current: current, Available: "run updater",
		Source: model.SourceFoundry, Status: model.StatusUpgrade, Selected: true, Detail: installDir,
	}}
}

func WindowsUpdates() []model.Package {
	script := `
$ErrorActionPreference = 'Stop'
$s = New-Object -ComObject Microsoft.Update.Session
$r = $s.CreateUpdateSearcher().Search("IsInstalled=0")
$list = @()
foreach ($i in 0..($r.Updates.Count - 1)) {
  $u = $r.Updates.Item($i)
  $kb = ''
  if ($u.KBArticleIDs.Count -gt 0) { $kb = $u.KBArticleIDs.Item(0) }
  $list += [PSCustomObject]@{ Index = $i; Title = $u.Title; KB = $kb }
}
$list | ConvertTo-Json -Compress
`
	out, _, err := exec.PowerShell(5*time.Minute, script)
	if err != nil || strings.TrimSpace(out) == "" || strings.TrimSpace(out) == "null" {
		return nil
	}
	var data []struct {
		Index int    `json:"Index"`
		Title string `json:"Title"`
		KB    string `json:"KB"`
	}
	raw := []byte(out)
	if raw[0] == '{' {
		var one struct {
			Index int    `json:"Index"`
			Title string `json:"Title"`
			KB    string `json:"KB"`
		}
		if json.Unmarshal(raw, &one) == nil {
			data = []struct {
				Index int    `json:"Index"`
				Title string `json:"Title"`
				KB    string `json:"KB"`
			}{{one.Index, one.Title, one.KB}}
		}
	} else {
		_ = json.Unmarshal(raw, &data)
	}
	var pkgs []model.Package
	for _, u := range data {
		title := u.Title
		if len(title) > 80 {
			title = title[:80]
		}
		avail := u.KB
		if avail == "" {
			avail = "install"
		}
		pkgs = append(pkgs, model.Package{
			Name: title, ID: "wu-" + strconv.Itoa(u.Index), Current: "pending", Available: avail,
			Source: model.SourceWinUpdate, Status: model.StatusUpgrade, Selected: false,
		})
	}
	if len(pkgs) > 0 {
		all := model.Package{
			Name: "Windows Update — install all (" + strconv.Itoa(len(pkgs)) + " pending)",
			ID: "wu-all", Current: strconv.Itoa(len(pkgs)), Available: "install",
			Source: model.SourceWinUpdate, Status: model.StatusUpgrade, Selected: false,
			Detail: "May require admin; can reboot",
		}
		pkgs = append([]model.Package{all}, pkgs...)
	}
	return pkgs
}
