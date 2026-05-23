package discover

import (
	"regexp"
	"strings"
	"time"

	"github.com/piyushmishra318/wingman/internal/exec"
	"github.com/piyushmishra318/wingman/internal/model"
)

var dashLine = regexp.MustCompile(`^-{3,}`)

func columnSlices(header string) []int {
	labels := []string{"Name", "Id", "Version", "Available", "Source"}
	pos := []int{}
	for _, l := range labels {
		if i := strings.Index(header, l); i >= 0 {
			pos = append(pos, i)
		}
	}
	if len(pos) < 3 {
		return []int{0, 36, 72, 92, 108, 512}
	}
	pos = append(pos, len(header)+80)
	return pos
}

func sliceCol(line string, pos []int, n int) string {
	if n >= len(pos)-1 {
		return ""
	}
	start, end := pos[n], pos[n+1]
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

func parseWingetTable(out string, requireAvailable bool) []model.Package {
	lines := strings.Split(out, "\n")
	headerIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "Name") && strings.Contains(line, "Id") && strings.Contains(line, "Version") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil
	}
	sepIdx := headerIdx + 1
	for sepIdx < len(lines) && !dashLine.MatchString(lines[sepIdx]) {
		sepIdx++
	}
	if sepIdx >= len(lines) {
		return nil
	}
	pos := columnSlices(lines[headerIdx])
	var pkgs []model.Package
	for _, line := range lines[sepIdx+1:] {
		line = strings.TrimRight(line, " \t\r")
		if line == "" || dashLine.MatchString(line) {
			continue
		}
		name := sliceCol(line, pos, 0)
		id := sliceCol(line, pos, 1)
		current := sliceCol(line, pos, 2)
		avail := sliceCol(line, pos, 3)
		srcRaw := strings.ToLower(sliceCol(line, pos, 4))
		if name == "" || id == "" {
			continue
		}
		if requireAvailable && (avail == "" || avail == current) {
			continue
		}
		if strings.HasPrefix(id, "ARP\\") || strings.Contains(id, "Steam App") {
			continue
		}
		src := model.SourceWinget
		if srcRaw == "msstore" {
			src = model.SourceMSStore
		}
		pkgs = append(pkgs, model.Package{
			Name:      name,
			ID:        id,
			Current:   orDefault(current, "?"),
			Available: orDefault(avail, "?"),
			Source:    src,
			Status:    model.StatusUpgrade,
			Selected:  true,
		})
	}
	return pkgs
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func WingetUpgrades() []model.Package {
	out, errOut, _ := exec.RunTimeout(5*time.Minute, "winget", "upgrade", "--include-unknown",
		"--accept-source-agreements", "--disable-interactivity")
	return parseWingetTable(out+errOut, true)
}
