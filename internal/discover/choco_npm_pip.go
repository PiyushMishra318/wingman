package discover

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/piyushmishra318/wingman/internal/exec"
	"github.com/piyushmishra318/wingman/internal/model"
)

func ChocoOutdated() []model.Package {
	if exec.Which("choco") == "" {
		return nil
	}
	out, _, _ := exec.RunTimeout(5*time.Minute, "choco", "outdated", "--limit-output", "--timeout", "300")
	var pkgs []model.Package
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 || parts[1] == parts[2] {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name: parts[0], ID: parts[0], Current: parts[1], Available: parts[2],
			Source: model.SourceChoco, Status: model.StatusUpgrade, Selected: true,
		})
	}
	return pkgs
}

func NPMGlobal() []model.Package {
	npm := exec.Which("npm")
	if npm == "" {
		return nil
	}
	out, _, err := exec.RunTimeout(2*time.Minute, npm, "outdated", "-g", "--json")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	var data map[string]json.RawMessage
	if json.Unmarshal([]byte(out), &data) != nil {
		return nil
	}
	var pkgs []model.Package
	for name, raw := range data {
		var info struct {
			Current string `json:"current"`
			Latest  string `json:"latest"`
			Wanted  string `json:"wanted"`
		}
		if json.Unmarshal(raw, &info) != nil {
			continue
		}
		avail := info.Latest
		if avail == "" {
			avail = info.Wanted
		}
		pkgs = append(pkgs, model.Package{
			Name: name, ID: name, Current: info.Current, Available: avail,
			Source: model.SourceNPM, Status: model.StatusUpgrade, Selected: true,
		})
	}
	return pkgs
}

func PIPOutdated() []model.Package {
	pip := exec.Which("pip")
	if pip == "" {
		return nil
	}
	out, _, err := exec.RunTimeout(3*time.Minute, pip, "list", "--outdated", "--format=json")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	var items []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Latest  string `json:"latest_version"`
	}
	if json.Unmarshal([]byte(out), &items) != nil {
		return nil
	}
	var pkgs []model.Package
	for _, it := range items {
		pkgs = append(pkgs, model.Package{
			Name: it.Name, ID: it.Name, Current: it.Version, Available: it.Latest,
			Source: model.SourcePIP, Status: model.StatusUpgrade, Selected: false, Detail: "bulk",
		})
	}
	if len(pkgs) > 0 {
		bulk := model.Package{
			Name:      "pip — all outdated (" + strconv.Itoa(len(pkgs)) + " packages)",
			ID:        "__pip_all__",
			Current:   strconv.Itoa(len(pkgs)),
			Available: "upgrade all",
			Source:    model.SourcePIP, Status: model.StatusUpgrade, Selected: false, Detail: "bulk",
		}
		pkgs = append([]model.Package{bulk}, pkgs...)
	}
	return pkgs
}
