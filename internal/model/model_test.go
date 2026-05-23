package model

import "testing"

func TestPackage_CanAutoUpdate_upgradeSources(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source Source
		status Status
		want   bool
	}{
		{SourceWinget, StatusUpgrade, true},
		{SourceChoco, StatusUpgrade, true},
		{SourceARP, StatusUpgrade, false},
		{SourceShortcut, StatusUpgrade, false},
		{SourceWinget, StatusOK, false},
		{SourceWinget, StatusFail, true},
	}
	for _, tc := range cases {
		p := Package{Source: tc.source, Status: tc.status}
		if got := p.CanAutoUpdate(); got != tc.want {
			t.Errorf("%s/%s: got %v want %v", tc.source, tc.status, got, tc.want)
		}
	}
}
