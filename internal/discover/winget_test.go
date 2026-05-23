package discover

import (
	"strings"
	"testing"

	"github.com/piyushmishra318/wingman/internal/model"
)

func TestParseWingetTable_listsUpgradablePackages(t *testing.T) {
	t.Parallel()
	sample := strings.TrimSpace(`
Name                    Id                      Version      Available    Source
--------------------------------------------------------------------------------
Git                     Git.Git                 2.44.0       2.45.0       winget
Microsoft Teams         Microsoft.Teams         24100.1306   24124.2404   msstore
`) + "\n"
	pkgs := parseWingetTable(sample, true)
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	if pkgs[0].Name != "Git" || pkgs[0].Available != "2.45.0" {
		t.Fatalf("unexpected first row: %+v", pkgs[0])
	}
	if pkgs[1].Source != model.SourceMSStore {
		t.Fatalf("second source %q, want msstore", pkgs[1].Source)
	}
}

func TestParseWingetTable_skipsARPAndSteamIDs(t *testing.T) {
	t.Parallel()
	sample := strings.TrimSpace(`
Name                    Id                      Version      Available    Source
--------------------------------------------------------------------------------
ARP Entry               ARP\\Some.Guid          1.0          2.0          winget
Game                    Steam App 12345         1.0          2.0          winget
`) + "\n"
	pkgs := parseWingetTable(sample, true)
	if len(pkgs) != 0 {
		t.Fatalf("got %d packages, want 0 (filtered)", len(pkgs))
	}
}
