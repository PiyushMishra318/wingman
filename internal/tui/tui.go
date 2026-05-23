package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/piyushmishra318/wingman/internal/discover"
	"github.com/piyushmishra318/wingman/internal/model"
	"github.com/piyushmishra318/wingman/internal/upgrade"
)

type viewMode int

const (
	viewUpdates viewMode = iota
	viewShortcuts
	viewARP
)

type scanStartedMsg struct{}
type scanDoneMsg struct {
	result discover.Result
	err    error
}

type updateStartedMsg struct{}
type updateDoneMsg struct {
	ok, fail int
}

type appModel struct {
	prog      *tea.Program
	packages  []model.Package
	shortcuts []model.Package
	rows      []model.Package
	logs      []string
	table     table.Model
	spinner   spinner.Model
	status    string
	width     int
	height    int
	scanning  bool
	updating  bool
	view      viewMode
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#d29922"))
)

func New() *appModel {
	cols := []table.Column{
		{Title: "✓", Width: 2},
		{Title: "Name", Width: 36},
		{Title: "Current", Width: 12},
		{Title: "Avail", Width: 12},
		{Title: "Source", Width: 10},
		{Title: "Status", Width: 8},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(18),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#30363d"))
	s.Selected = s.Selected.Foreground(lipgloss.Color("#58a6ff")).Bold(true)
	s.Cell = s.Cell.Padding(0, 1)
	t.SetStyles(s)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))

	return &appModel{
		table:   t,
		spinner: sp,
		status:  "Starting scan…",
		view:    viewUpdates,
	}
}

func (m *appModel) Init() tea.Cmd {
	return tea.Batch(m.beginScan(), m.spinner.Tick, tea.EnterAltScreen)
}

func (m *appModel) beginScan() tea.Cmd {
	m.scanning = true
	m.status = "Scanning (parallel) — q to quit"
	return func() tea.Msg {
		go func() {
			r := discover.All(true)
			if m.prog != nil {
				m.prog.Send(scanDoneMsg{result: r})
			}
		}()
		return scanStartedMsg{}
	}
}

func (m *appModel) beginUpdates(pkgs []model.Package) tea.Cmd {
	if len(pkgs) == 0 {
		return nil
	}
	m.updating = true
	m.status = fmt.Sprintf("Updating %d package(s)…", len(pkgs))
	upgrade.SortPackages(pkgs)
	// Copy IDs so goroutine uses stable list
	ids := make([]model.Package, len(pkgs))
	copy(ids, pkgs)

	return func() tea.Msg {
		go func() {
			ok, fail := 0, 0
			for _, p := range ids {
				if upgrade.Package(p, func(string) {}) {
					ok++
				} else {
					fail++
				}
			}
			if m.prog != nil {
				m.prog.Send(updateDoneMsg{ok: ok, fail: fail})
			}
		}()
		return updateStartedMsg{}
	}
}

func (m *appModel) visible() []model.Package {
	switch m.view {
	case viewShortcuts:
		return m.shortcuts
	case viewARP:
		out := make([]model.Package, 0)
		for _, p := range m.packages {
			if p.Source == model.SourceARP {
				out = append(out, p)
			}
		}
		return out
	default:
		out := make([]model.Package, 0, len(m.packages))
		for _, p := range m.packages {
			if p.Source != model.SourceARP {
				out = append(out, p)
			}
		}
		return out
	}
}

func (m *appModel) rebuildTable() {
	m.rows = m.visible()
	rows := make([]table.Row, 0, len(m.rows))
	for _, p := range m.rows {
		mark := "☐"
		if p.Selected {
			mark = "☑"
		}
		if !p.CanAutoUpdate() {
			mark = "·"
		}
		st := string(p.Status)
		if p.Source == model.SourceShortcut {
			st = "launcher"
		}
		if p.Source == model.SourceARP {
			st = "manual"
		}
		name, cur, av := p.Name, p.Current, p.Available
		if len(name) > 36 {
			name = name[:35] + "…"
		}
		if len(cur) > 12 {
			cur = cur[:11] + "…"
		}
		if len(av) > 12 {
			av = av[:11] + "…"
		}
		rows = append(rows, table.Row{mark, name, cur, av, string(p.Source), st})
	}
	m.table.SetRows(rows)
	if len(rows) > 0 {
		cur := m.table.Cursor()
		if cur >= len(rows) {
			m.table.SetCursor(0)
		}
	}
}

func (m *appModel) statsLine() string {
	auto := 0
	by := map[model.Source]int{}
	for _, p := range m.packages {
		if p.CanAutoUpdate() {
			auto++
		}
		by[p.Source]++
	}
	parts := make([]string, 0, len(by))
	for s, n := range by {
		parts = append(parts, fmt.Sprintf("%s:%d", s, n))
	}
	sort.Strings(parts)
	return fmt.Sprintf("%d items · %d auto-upgradable · ARP %d · %s",
		len(m.packages), auto, by[model.SourceARP], strings.Join(parts, " "))
}

func (m *appModel) handleActionKey(key string) (bool, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		return true, tea.Quit
	case "r":
		if m.scanning || m.updating {
			m.logs = append(m.logs, warnStyle.Render("Wait for current operation to finish."))
			return true, nil
		}
		return true, tea.Batch(m.beginScan(), m.spinner.Tick)
	case "s":
		if m.view == viewShortcuts {
			m.view = viewUpdates
		} else {
			m.view = viewShortcuts
		}
		m.rebuildTable()
		return true, nil
	case "i":
		if m.view == viewARP {
			m.view = viewUpdates
		} else {
			m.view = viewARP
		}
		m.rebuildTable()
		return true, nil
	case " ":
		if m.scanning || m.updating || len(m.rows) == 0 {
			return true, nil
		}
		idx := m.table.Cursor()
		if idx >= 0 && idx < len(m.rows) && m.rows[idx].CanAutoUpdate() {
			m.rows[idx].Selected = !m.rows[idx].Selected
			m.syncRowBack(idx)
			m.rebuildTable()
			m.table.SetCursor(idx)
		}
		return true, nil
	case "a":
		if m.scanning || m.updating {
			return true, nil
		}
		return true, m.beginUpdates(m.allAuto())
	case "u":
		if m.scanning || m.updating {
			return true, nil
		}
		return true, m.beginUpdates(m.selectedAuto())
	default:
		return false, nil
	}
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if ok, cmd := m.handleActionKey(msg.String()); ok {
			return m, cmd
		}
		if m.scanning || m.updating {
			return m, nil
		}

	case tea.MouseMsg:
		if m.scanning || m.updating {
			return m, nil
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h := msg.Height - 12
		if h < 4 {
			h = 4
		}
		m.table.SetWidth(max(40, msg.Width-4))
		m.table.SetHeight(h)
		return m, nil

	case scanStartedMsg:
		m.scanning = true
		return m, m.spinner.Tick

	case scanDoneMsg:
		m.scanning = false
		if msg.err != nil {
			m.status = "Scan failed"
			m.logs = append(m.logs, warnStyle.Render("Scan error: "+msg.err.Error()))
			return m, nil
		}
		m.packages = msg.result.Packages
		m.shortcuts = msg.result.Shortcuts
		m.rebuildTable()
		m.status = m.statsLine()
		m.logs = append(m.logs, okStyle.Render("Scan complete.")+"  s shortcuts · i ARP · click rows")
		return m, nil

	case updateStartedMsg:
		return m, m.spinner.Tick

	case updateDoneMsg:
		m.updating = false
		m.logs = append(m.logs, fmt.Sprintf("Done: %d ok, %d failed — press r to rescan", msg.ok, msg.fail))
		m.status = m.statsLine()
		return m, tea.Batch(m.beginScan(), m.spinner.Tick)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.scanning || m.updating {
			return m, cmd
		}
	}

	// Navigation keys → table (only when idle)
	if m.scanning || m.updating {
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *appModel) syncRowBack(idx int) {
	p := m.rows[idx]
	switch m.view {
	case viewShortcuts:
		for i := range m.shortcuts {
			if m.shortcuts[i].ID == p.ID {
				m.shortcuts[i].Selected = p.Selected
				return
			}
		}
	default:
		for i := range m.packages {
			if m.packages[i].ID == p.ID && m.packages[i].Source == p.Source {
				m.packages[i].Selected = p.Selected
				return
			}
		}
	}
}

func (m *appModel) selectedAuto() []model.Package {
	var out []model.Package
	for _, p := range m.packages {
		if p.Selected && p.CanAutoUpdate() {
			out = append(out, p)
		}
	}
	return out
}

func (m *appModel) allAuto() []model.Package {
	var out []model.Package
	for i := range m.packages {
		if m.packages[i].CanAutoUpdate() {
			m.packages[i].Selected = true
			out = append(out, m.packages[i])
		}
	}
	return out
}

func (m *appModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Wingman"))
	b.WriteString(dimStyle.Render("  —  winget · Store · choco · npm · pip · Steam · Windows · ARP\n"))

	status := m.status
	if m.scanning || m.updating {
		status = m.spinner.View() + " " + status
	}
	b.WriteString(dimStyle.Render(status) + "\n\n")
	b.WriteString(m.table.View())
	if len(m.logs) > 0 {
		b.WriteString("\n\n")
		start := 0
		if len(m.logs) > 8 {
			start = len(m.logs) - 8
		}
		b.WriteString(dimStyle.Render(strings.Join(m.logs[start:], "\n")))
	}
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render(
		"↑↓/click navigate · space toggle · r scan · u/a upgrade · s shortcuts · i ARP · q quit",
	))
	return b.String()
}

func Run() error {
	m := New()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	m.prog = p
	_, err := p.Run()
	return err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
