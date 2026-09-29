// Package tui is the interactive keymap and lighting editor.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"via-terminal/internal/defs"
	"via-terminal/internal/keycodes"
	"via-terminal/internal/via"
)

// Screen rows above each tab's content: title, tab bar, blank line. Mouse
// hit-testing counts on it.
const bodyTop = 3

type model struct {
	def       defs.Definition
	dev       *via.Device
	layers    int
	keymap    []uint16
	gridLines int

	tab    int     // 0 is the keymap, then one tab per definition menu
	picker *picker // open search list, nil when closed
	status string

	layer          int
	sel            int
	keycodeChoices []choice

	values map[string]int // menu settings by control ID, absent if unreadable
	row    int            // selected row on a menu tab
	dirty  map[byte]bool  // channels changed since the last save

	start, now time.Time // lighting preview clock
	tickGen    int
}

// Run shows the editor until the user quits. keymap is indexed like
// via.Device.Keymap returns it and is updated in place on every write.
func Run(def defs.Definition, dev *via.Device, layers int, keymap []uint16) error {
	_, err := tea.NewProgram(newModel(def, dev, layers, keymap)).Run()
	return err
}

func newModel(def defs.Definition, dev *via.Device, layers int, keymap []uint16) model {
	m := model{
		def:       def,
		dev:       dev,
		layers:    layers,
		keymap:    keymap,
		gridLines: strings.Count(render(def.Keys, func(defs.Key) string { return "" }, nil), "\n") + 1,
		values:    map[string]int{},
		dirty:     map[byte]bool{},
		start:     time.Now(),
	}
	for _, k := range keycodes.Picker(layers, def.Custom) {
		m.keycodeChoices = append(m.keycodeChoices, choice{k.Name, k.Long, int(k.Code)})
	}
	for _, menu := range def.Menus {
		for _, c := range menu.Items {
			if !c.HasValue() {
				continue
			}
			if v, err := dev.CustomValue(c.Channel, c.ValueID, c.Size()); err == nil {
				m.values[c.ID] = v
			}
		}
	}
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		s := msg.String()
		if s == "ctrl+c" || s == "q" && m.picker == nil {
			return m.save(), tea.Quit
		}
		if m.picker != nil {
			return m.updatePicker(msg), nil
		}
		switch s {
		case "tab":
			m = m.switchTab(m.tab + 1)
			return m, m.tick()
		case "shift+tab":
			m = m.switchTab(m.tab - 1)
			return m, m.tick()
		}
		if m.tab == 0 {
			return m.updateMapping(s), nil
		}
		return m.updateMenu(s), nil

	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Y == 1 {
			if t := m.tabAt(mouse.X); t >= 0 {
				m = m.switchTab(t)
				return m, m.tick()
			}
			return m, nil
		}
		if m.tab == 0 {
			return m.clickMapping(mouse.X, mouse.Y), nil
		}
		return m.clickMenu(mouse.X, mouse.Y), nil

	case tickMsg:
		if int(msg) != m.tickGen {
			return m, nil
		}
		m.now = time.Now()
		return m, m.tick()
	}
	return m, nil
}

func (m model) tabNames() []string {
	names := []string{"MAPPING"}
	for _, menu := range m.def.Menus {
		names = append(names, strings.ToUpper(menu.Label))
	}
	return names
}

// tabAt matches the tab bar's layout: each name padded by a space on both
// sides, one space between tabs.
func (m model) tabAt(x int) int {
	start := 0
	for i, name := range m.tabNames() {
		end := start + len([]rune(name)) + 2
		if x >= start && x < end {
			return i
		}
		start = end + 1
	}
	return -1
}

func (m model) switchTab(t int) model {
	m.status = ""
	m = m.save()
	n := len(m.def.Menus) + 1
	m.tab, m.row, m.picker = (t%n+n)%n, 0, nil
	m.tickGen++
	return m
}

// save persists changed menu settings, which the board otherwise only keeps
// in RAM until it's unplugged.
func (m model) save() model {
	for ch := range m.dirty {
		if err := m.dev.SaveCustom(ch); err != nil {
			m.status = "SAVE FAILED: " + err.Error()
			return m
		}
		delete(m.dirty, ch)
	}
	return m
}

func (m model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1b[1m◆ %s\x1b[22m - VIA v%d\n", strings.ToUpper(m.def.Name), m.dev.Version)
	for i, name := range m.tabNames() {
		if i > 0 {
			b.WriteString(" ")
		}
		if i == m.tab {
			b.WriteString("\x1b[7m " + name + " \x1b[27m")
		} else {
			b.WriteString(" " + name + " ")
		}
	}
	b.WriteString("\n\n")
	if m.tab == 0 {
		m.viewMapping(&b)
	} else {
		m.viewMenu(&b)
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
