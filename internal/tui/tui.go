// Package tui is the interactive keymap editor.
package tui

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"

	"via-terminal/internal/defs"
	"via-terminal/internal/keycodes"
	"via-terminal/internal/via"
)

// Lines above the keyboard; mouse hit-testing counts on it.
const gridTop = 3

const pickRows = 10

type model struct {
	def       defs.Definition
	dev       *via.Device
	layers    int
	keymap    []uint16
	gridLines int

	layer  int
	sel    int
	status string

	picking bool
	query   string
	all     []keycodes.Keycode
	matches []keycodes.Keycode
	pick    int
}

// Run shows the editor until the user quits. keymap is indexed like
// via.Device.Keymap returns it and is updated in place on every write.
func Run(def defs.Definition, dev *via.Device, layers int, keymap []uint16) error {
	_, err := tea.NewProgram(newModel(def, dev, layers, keymap)).Run()
	return err
}

func newModel(def defs.Definition, dev *via.Device, layers int, keymap []uint16) model {
	return model{
		def:       def,
		dev:       dev,
		layers:    layers,
		keymap:    keymap,
		gridLines: strings.Count(render(def.Keys, func(defs.Key) string { return "" }, -1), "\n") + 1,
		all:       keycodes.Picker(layers, def.Custom),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) index(layer int, k defs.Key) int {
	return (layer*m.def.Rows+k.Row)*m.def.Cols + k.Col
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.picking {
			return m.updatePicker(msg), nil
		}
		switch s := msg.String(); s {
		case "q":
			return m, tea.Quit
		case "up", "k":
			m.sel = nearestKey(m.def.Keys, m.sel, 0, -1)
		case "down", "j":
			m.sel = nearestKey(m.def.Keys, m.sel, 0, 1)
		case "left", "h":
			m.sel = nearestKey(m.def.Keys, m.sel, -1, 0)
		case "right", "l":
			m.sel = nearestKey(m.def.Keys, m.sel, 1, 0)
		case "]", "tab", "pgdown":
			m.layer = (m.layer + 1) % m.layers
		case "[", "shift+tab", "pgup":
			m.layer = (m.layer + m.layers - 1) % m.layers
		case "enter", "space":
			m = m.openPicker()
		default:
			if len(s) == 1 && s[0] >= '0' && int(s[0]-'0') < m.layers {
				m.layer = int(s[0] - '0')
			}
		}

	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Y == 1 {
			line := m.layerLine()
			if abs(mouse.X-strings.Index(line, "<")) <= 1 {
				m.layer = (m.layer + m.layers - 1) % m.layers
			} else if abs(mouse.X-strings.Index(line, ">")) <= 1 {
				m.layer = (m.layer + 1) % m.layers
			}
		} else if i := keyAt(m.def.Keys, mouse.X, mouse.Y-gridTop); i >= 0 {
			m.sel = i
			if !m.picking {
				m = m.openPicker()
			}
		} else if m.picking {
			start, _ := m.pickWindow()
			if i := start + mouse.Y - (gridTop + m.gridLines + 2); i >= start && i < len(m.matches) {
				m = m.assign(m.matches[i])
			}
		}
	}
	return m, nil
}

func (m model) openPicker() model {
	m.picking, m.query, m.pick, m.matches = true, "", 0, m.all
	return m
}

func (m model) updatePicker(msg tea.KeyPressMsg) model {
	switch msg.String() {
	case "esc":
		m.picking = false
		return m
	case "enter":
		if len(m.matches) > 0 {
			return m.assign(m.matches[m.pick])
		}
		return m
	case "up":
		m.pick = max(0, m.pick-1)
		return m
	case "down":
		m.pick = max(0, min(len(m.matches)-1, m.pick+1))
		return m
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	default:
		m.query += msg.Text
	}
	m.matches, m.pick = keycodes.Filter(m.all, m.query), 0
	return m
}

func (m model) assign(kc keycodes.Keycode) model {
	m.picking = false
	k := m.def.Keys[m.sel]
	if err := m.dev.SetKeycode(m.layer, k.Row, k.Col, kc.Code); err != nil {
		m.status = "WRITE FAILED: " + err.Error()
		return m
	}
	m.keymap[m.index(m.layer, k)] = kc.Code
	m.status = fmt.Sprintf("SET %d,%d ON LAYER %d TO %s", k.Row, k.Col, m.layer, keycodes.Name(kc.Code, m.def.Custom))
	return m
}

func (m model) pickWindow() (start, end int) {
	start = max(0, m.pick-pickRows+1)
	return start, min(len(m.matches), start+pickRows)
}

func (m model) layerLine() string {
	return fmt.Sprintf("LAYER < %d >", m.layer)
}

// legend shows the layer 0 key above the current mapping on higher layers,
// so it stays clear which physical key is being changed.
func (m model) legend(k defs.Key) string {
	label := func(code uint16) string {
		switch code {
		case 0x0000:
			return ""
		case 0x0001:
			return "▽"
		}
		return keycodes.Name(code, m.def.Custom)
	}
	current := label(m.keymap[m.index(m.layer, k)])
	if m.layer == 0 {
		return current
	}
	return label(m.keymap[m.index(0, k)]) + "\n" + current
}

func (m model) View() tea.View {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1b[1m◆ %s\x1b[22m - VIA v%d\n%s\n\n", strings.ToUpper(m.def.Name), m.dev.Version, m.layerLine())
	b.WriteString(render(m.def.Keys, m.legend, m.sel))
	b.WriteString("\n\n")

	k := m.def.Keys[m.sel]
	if m.picking {
		fmt.Fprintf(&b, "> REMAP %d,%d ON LAYER %d: %s_\n", k.Row, k.Col, m.layer, m.query)
		start, end := m.pickWindow()
		for i := start; i < end; i++ {
			line := fmt.Sprintf("  %-12s %s", m.matches[i].Name, m.matches[i].Long)
			if i == m.pick {
				line = "\x1b[7m" + line + "\x1b[27m"
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\x1b[2mTYPE TO SEARCH · ENTER/CLICK ASSIGN · ESC CANCEL\x1b[22m")
	} else {
		fmt.Fprintf(&b, "> %d,%d  %s\n%s\n", k.Row, k.Col, keycodes.Name(m.keymap[m.index(m.layer, k)], m.def.Custom), m.status)
		b.WriteString("\x1b[2mARROWS MOVE · ENTER/CLICK REMAP · [ ] OR 0-9 LAYER · Q QUIT\x1b[22m")
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// nearestKey steps from key i in direction dx,dy to the closest key whose
// center lies past that edge of key i. Off-axis distance counts double so
// moves stay in their row or column.
func nearestKey(keys []defs.Key, i, dx, dy int) int {
	k := keys[i]
	cx, cy := k.X+k.W/2, k.Y+k.H/2
	best, bestScore := i, math.Inf(1)
	for j, o := range keys {
		x, y := o.X+o.W/2, o.Y+o.H/2
		var along, across float64
		switch {
		case dx > 0:
			along, across = x-(k.X+k.W), math.Abs(y-cy)
		case dx < 0:
			along, across = k.X-x, math.Abs(y-cy)
		case dy > 0:
			along, across = y-(k.Y+k.H), math.Abs(x-cx)
		default:
			along, across = k.Y-y, math.Abs(x-cx)
		}
		if along <= 0 {
			continue
		}
		if score := along + 2*across; score < bestScore {
			best, bestScore = j, score
		}
	}
	return best
}

func abs(n int) int { return max(n, -n) }
