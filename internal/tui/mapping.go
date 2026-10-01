package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/bulkinglb/via-terminal/internal/defs"
	"github.com/bulkinglb/via-terminal/internal/keycodes"
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
		m = m.remap()
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
		m = m.remap()
	} else if m.picker != nil {
		m = m.clickPicker(y, gridTop+m.gridLines+1)
	}
	return m
}

// remap opens the keycode search for the selected key. For a knob it first
// asks whether to change the press or a turn.
func (m model) remap() model {
	k := m.def.Keys[m.sel]
	press := m.keymap[m.index(m.layer, k)]
	prompt := fmt.Sprintf("REMAP %d,%d ON LAYER %d", k.Row, k.Col, m.layer)
	if !k.Knob || m.encoders == nil {
		m.picker = newPicker(prompt, m.keycodeChoices, int(press), func(m model, v int) model {
			return m.assign(uint16(v))
		})
		return m
	}
	left, right := m.knobTurns(k)
	actions := []choice{
		{"PRESS", m.name(press), 0, ""},
		{"TURN LEFT", m.name(left), 1, ""},
		{"TURN RIGHT", m.name(right), 2, ""},
	}
	m.picker = newPicker(fmt.Sprintf("KNOB ON LAYER %d", m.layer), actions, 0, func(m model, action int) model {
		current := []uint16{press, left, right}[action]
		m.picker = newPicker(prompt+" · "+actions[action].label, m.keycodeChoices, int(current), func(m model, v int) model {
			if action == 0 {
				return m.assign(uint16(v))
			}
			return m.setTurn(k, action-1, uint16(v))
		})
		return m
	})
	return m
}

func (m model) name(code uint16) string { return keycodes.Name(code, m.def.Custom) }

// knobTurns returns what turning knob k left and right does on the shown
// layer.
func (m model) knobTurns(k defs.Key) (left, right uint16) {
	i := (m.layer*m.def.Encoders + k.Encoder) * 2
	return m.encoders[i], m.encoders[i+1]
}

// setTurn changes a turn of knob k on the shown layer: cw 0 is left, 1 right.
func (m model) setTurn(k defs.Key, cw int, code uint16) model {
	if err := m.dev.SetEncoder(m.layer, k.Encoder, cw, code); err != nil {
		m.status = "WRITE FAILED: " + err.Error()
		return m
	}
	m.encoders[(m.layer*m.def.Encoders+k.Encoder)*2+cw] = code
	m.status = fmt.Sprintf("SET KNOB TURN %s ON LAYER %d TO %s", []string{"LEFT", "RIGHT"}[cw], m.layer, m.name(code))
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
	help, marked := m.layerHelp()
	fmt.Fprintf(b, "%s  \x1b[2m%s\x1b[22m\n\n", m.layerLine(), help)
	b.WriteString(render(m.def.Keys, m.legend, func(i int) string {
		switch {
		case i == m.sel:
			return "\x1b[7m"
		case marked[i]:
			return "\x1b[100m"
		}
		return ""
	}))
	b.WriteString("\n\n")

	if m.picker != nil {
		m.viewPicker(b)
		return
	}
	k := m.def.Keys[m.sel]
	code := m.keymap[m.index(m.layer, k)]
	knob := ""
	if k.Knob && m.encoders != nil {
		left, right := m.knobTurns(k)
		knob = fmt.Sprintf("  · TURN LEFT %s · TURN RIGHT %s", m.name(left), m.name(right))
	}
	fmt.Fprintf(b, "> %d,%d  %s  \x1b[2m%s\x1b[22m%s\n%s\n", k.Row, k.Col, m.name(code), keycodes.Describe(code), knob, m.status)
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
