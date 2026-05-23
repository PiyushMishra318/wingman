package model

type Source string

const (
	SourceWinget    Source = "winget"
	SourceMSStore   Source = "msstore"
	SourceChoco     Source = "choco"
	SourceNPM       Source = "npm"
	SourcePIP       Source = "pip"
	SourceScoop     Source = "scoop"
	SourceSteam     Source = "steam"
	SourceWinUpdate Source = "winupdate"
	SourceARP       Source = "arp"
	SourceShortcut  Source = "shortcut"
)

type Status string

const (
	StatusUpgrade Status = "upgrade"
	StatusManual  Status = "manual"
	StatusWorking Status = "working"
	StatusOK      Status = "ok"
	StatusFail    Status = "fail"
)

type Package struct {
	Name      string
	ID        string
	Current   string
	Available string
	Source    Source
	Status    Status
	Target    string
	Detail    string
	Selected  bool
}

func (p Package) CanAutoUpdate() bool {
	if p.Source == SourceARP || p.Source == SourceShortcut {
		return false
	}
	switch p.Source {
	case SourceWinget, SourceMSStore, SourceChoco, SourceNPM, SourcePIP,
		SourceScoop, SourceSteam, SourceWinUpdate:
		return p.Status == StatusUpgrade || p.Status == StatusFail
	}
	return false
}

func SourceOrder(s Source) int {
	switch s {
	case SourceWinUpdate:
		return 0
	case SourceWinget:
		return 1
	case SourceMSStore:
		return 2
	case SourceChoco:
		return 3
	case SourceNPM:
		return 4
	case SourcePIP:
		return 5
	case SourceScoop:
		return 6
	case SourceSteam:
		return 7
	case SourceARP:
		return 9
	case SourceShortcut:
		return 10
	default:
		return 99
	}
}

func UpgradeOrder(s Source) int {
	switch s {
	case SourceNPM:
		return 0
	case SourcePIP:
		return 1
	case SourceChoco:
		return 2
	case SourceWinget, SourceMSStore:
		return 3
	case SourceScoop:
		return 4
	case SourceSteam:
		return 5
	case SourceWinUpdate:
		return 6
	default:
		return 50
	}
}
