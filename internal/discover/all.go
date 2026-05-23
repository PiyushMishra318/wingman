package discover

import (
	"sort"
	"strings"
	"sync"

	"github.com/piyushmishra318/wingman/internal/model"
)

type Result struct {
	Packages  []model.Package
	Shortcuts []model.Package
}

func All(includeShortcuts bool) Result {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		winget   []model.Package
		choco    []model.Package
		npm      []model.Package
		pip      []model.Package
		steam    []model.Package
		winup    []model.Package
		shortcuts []model.Package
	)

	run := func(fn func() []model.Package, slot *[]model.Package) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pkgs := fn()
			mu.Lock()
			*slot = pkgs
			mu.Unlock()
		}()
	}

	run(WingetUpgrades, &winget)
	run(ChocoOutdated, &choco)
	run(NPMGlobal, &npm)
	run(PIPOutdated, &pip)
	run(Steam, &steam)
	run(WindowsUpdates, &winup)

	if includeShortcuts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc := StartMenuShortcuts()
			mu.Lock()
			shortcuts = sc
			mu.Unlock()
		}()
	}

	wg.Wait()

	knownCap := len(winget) + len(choco) + len(npm) + len(steam)
	known := make(map[string]bool, knownCap)
	for _, batch := range [][]model.Package{winget, choco, npm, steam} {
		for _, p := range batch {
			known[p.Name] = true
		}
	}
	arp := ARP(known)

	merged := Merge(winget, choco, npm, pip, steam, winup, arp)
	return Result{Packages: merged, Shortcuts: shortcuts}
}

func Merge(groups ...[]model.Package) []model.Package {
	total := 0
	for _, g := range groups {
		total += len(g)
	}
	byKey := make(map[string]model.Package, total)
	nameSeen := make(map[string]struct{}, total)

	for _, group := range groups {
		for _, pkg := range group {
			if pkg.Source == model.SourceARP {
				byKey["arp:"+pkg.ID] = pkg
				continue
			}
			nk := normName(pkg.Name)
			if _, dup := nameSeen[nk]; dup {
				continue
			}
			key := string(pkg.Source) + ":" + strings.ToLower(pkg.ID)
			byKey[key] = pkg
			nameSeen[nk] = struct{}{}
		}
	}

	out := make([]model.Package, 0, len(byKey))
	for _, p := range byKey {
		out = append(out, p)
	}
	sortPackages(out)
	return out
}

func sortPackages(out []model.Package) {
	nameKeys := make([]string, len(out))
	for i := range out {
		nameKeys[i] = normName(out[i].Name)
	}
	sort.Slice(out, func(i, j int) bool {
		oi, oj := model.SourceOrder(out[i].Source), model.SourceOrder(out[j].Source)
		if oi != oj {
			return oi < oj
		}
		ci, cj := out[i].CanAutoUpdate(), out[j].CanAutoUpdate()
		if ci != cj {
			return ci
		}
		return nameKeys[i] < nameKeys[j]
	})
}
