package tui

import (
	"fmt"
	"math"
	"strings"

	"via-terminal/internal/defs"
	"via-terminal/internal/keycodes"
)

// The keyboard sits below the layer switcher and a blank line.
const gridTop = bodyTop + 2

func (m model) index(layer int, k defs.Key) int {
	return (layer*m.def.Rows+k.Row)*m.def.Cols + k.Col
}

func (m model) updateMapping(s string) model {
	switch s {
	case "up", "k":
		m.sel = nearestKey(m.def.Keys, m.sel, 0, -1)
	case "down", "j":
		m.sel = nearestKey(m.def.Keys, m.sel, 0, 1)
	case "left", "h":
		m.sel = nearestKey(m.def.Keys, m.sel, -1, 0)
	case "right", "l":
		m.sel = nearestKey(m.def.Keys, m.sel, 1, 0)
	case "]", "pgdown":
		m.layer = (m.layer + 1) % m.layers
	case "[", "pgup":
		m.layer = (m.layer + m.layers - 1) % m.layers
	case "enter", "space":
		m = m.openKeycodePicker()
	default:
		if len(s) == 1 && s[0] >= '0' && int(s[0]-'0') < m.layers {
			m.layer = int(s[0] - '0')
		}
	}
	return m
}

func (m model) clickMapping(x, y int) model {
	if y == bodyTop {
		line := m.layerLine()
		if abs(x-strings.Index(line, "<")) <= 1 {
			m.layer = (m.layer + m.layers - 1) % m.layers
		} else if abs(x-strings.Index(line, ">")) <= 1 {
			m.layer = (m.layer + 1) % m.layers
		}
	} else if i := keyAt(m.def.Keys, x, y-gridTop); i >= 0 {
		m.sel = i
		m = m.openKeycodePicker()
	} else if m.picker != nil {
		m = m.clickPicker(y, gridTop+m.gridLines+1)
	}
	return m
}

func (m model) openKeycodePicker() model {
	k := m.def.Keys[m.sel]
	prompt := fmt.Sprintf("REMAP %d,%d ON LAYER %d", k.Row, k.Col, m.layer)
	m.picker = newPicker(prompt, m.keycodeChoices, int(m.keymap[m.index(m.layer, k)]), func(m model, v int) model {
		return m.assign(uint16(v))
	})
	return m
}

func (m model) assign(code uint16) model {
	k := m.def.Keys[m.sel]
	if err := m.dev.SetKeycode(m.layer, k.Row, k.Col, code); err != nil {
		m.status = "WRITE FAILED: " + err.Error()
		return m
	}
	m.keymap[m.index(m.layer, k)] = code
	m.status = fmt.Sprintf("SET %d,%d ON LAYER %d TO %s", k.Row, k.Col, m.layer, keycodes.Name(code, m.def.Custom))
	return m
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

func (m model) viewMapping(b *strings.Builder) {
	b.WriteString(m.layerLine() + "\n\n")
	b.WriteString(render(m.def.Keys, m.legend, m.sel))
	b.WriteString("\n\n")

	if m.picker != nil {
		m.viewPicker(b)
		return
	}
	k := m.def.Keys[m.sel]
	fmt.Fprintf(b, "> %d,%d  %s\n%s\n", k.Row, k.Col, keycodes.Name(m.keymap[m.index(m.layer, k)], m.def.Custom), m.status)
	b.WriteString("\x1b[2mARROWS MOVE · ENTER/CLICK REMAP · [ ] OR 0-9 LAYER · TAB SWITCH · Q QUIT\x1b[22m")
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
