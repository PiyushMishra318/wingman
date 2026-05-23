package discover

import (
	"testing"

	"github.com/piyushmishra318/wingman/internal/model"
)

func TestMerge_deduplicatesSameNameAcrossSources(t *testing.T) {
	t.Parallel()
	winget := []model.Package{{
		Name: "Visual Studio Code", ID: "Microsoft.VisualStudioCode",
		Source: model.SourceWinget, Status: model.StatusUpgrade,
	}}
	choco := []model.Package{{
		Name: "visual studio code", ID: "vscode",
		Source: model.SourceChoco, Status: model.StatusUpgrade,
	}}
	merged := Merge(winget, choco)
	if len(merged) != 1 {
		t.Fatalf("got %d packages, want 1", len(merged))
	}
	if merged[0].Source != model.SourceWinget {
		t.Fatalf("got source %q, want winget (first source wins)", merged[0].Source)
	}
}

func TestMerge_keepsDistinctARPByID(t *testing.T) {
	t.Parallel()
	arp := []model.Package{
		{Name: "App A", ID: "guid-1", Source: model.SourceARP, Status: model.StatusManual},
		{Name: "App B", ID: "guid-2", Source: model.SourceARP, Status: model.StatusManual},
	}
	merged := Merge(arp)
	if len(merged) != 2 {
		t.Fatalf("got %d ARP entries, want 2", len(merged))
	}
}

func TestMerge_deduplicatesPunctuationVariants(t *testing.T) {
	t.Parallel()
	a := []model.Package{{Name: "7-Zip 24.08", ID: "a", Source: model.SourceWinget, Status: model.StatusUpgrade}}
	b := []model.Package{{Name: "7zip2408", ID: "b", Source: model.SourceChoco, Status: model.StatusUpgrade}}
	merged := Merge(a, b)
	if len(merged) != 1 {
		t.Fatalf("got %d packages, want 1 (normName dedup)", len(merged))
	}
}

func TestMerge_sortsWindowsUpdatesFirst(t *testing.T) {
	t.Parallel()
	wu := []model.Package{{Name: "KB1", ID: "wu-0", Source: model.SourceWinUpdate, Status: model.StatusUpgrade}}
	wg := []model.Package{{Name: "Git", ID: "Git.Git", Source: model.SourceWinget, Status: model.StatusUpgrade}}
	merged := Merge(wu, wg)
	if len(merged) != 2 {
		t.Fatalf("got %d packages, want 2", len(merged))
	}
	if merged[0].Source != model.SourceWinUpdate {
		t.Fatalf("first package source %q, want winupdate", merged[0].Source)
	}
}
