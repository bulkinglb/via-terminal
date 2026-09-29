package tui

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"via-terminal/internal/defs"
)

const labelWidth, valueWidth = 30, 20

// menuRow is one line on a menu tab. A color control gets two rows, hue
// and saturation.
type menuRow struct {
	ctl     defs.Control
	section string
	sat     bool
}

func (m model) menuRows() []menuRow {
	var rows []menuRow
	section := ""
	for _, c := range m.def.Menus[m.tab-1].Items {
		switch {
		case c.Type == "":
			section = c.Label
		case c.Visible(m.values):
			rows = append(rows, menuRow{c, section, false})
			if c.Type == "color" {
				rows = append(rows, menuRow{c, section, true})
			}
		}
	}
	return rows
}

func (m model) updateMenu(s string) model {
	rows := m.menuRows()
	switch s {
	case "up", "k":
		m.row = max(0, m.row-1)
	case "down", "j":
		m.row = max(0, min(len(rows)-1, m.row+1))
	case "left", "h":
		return m.adjust(rows, -1, false)
	case "right", "l":
		return m.adjust(rows, 1, false)
	case "shift+left", "H":
		return m.adjust(rows, -1, true)
	case "shift+right", "L":
		return m.adjust(rows, 1, true)
	}
	return m
}

func (m model) clickMenu(x, y int) model {
	rows := m.menuRows()
	_, rowLine := m.menuLines()
	for i, line := range rowLine {
		if y-bodyTop != line {
			continue
		}
		m.row = i
		label, _ := m.rowText(rows[i])
		lt := 2 + max(labelWidth, len([]rune(label))+2) + 1
		gt := lt + 2 + valueWidth + 1
		if abs(x-lt) <= 1 {
			return m.adjust(rows, -1, false)
		}
		if abs(x-gt) <= 1 {
			return m.adjust(rows, 1, false)
		}
		return m
	}
	return m
}

// adjust steps the selected row by dir. A coarse step is a twentieth of the
// range, a fine step one unit.
func (m model) adjust(rows []menuRow, dir int, fine bool) model {
	if m.row >= len(rows) {
		return m
	}
	r := rows[m.row]
	c := r.ctl
	v, ok := m.values[c.ID]
	if !ok {
		return m
	}
	step := func(lo, hi int) int {
		if fine {
			return dir
		}
		return dir * max(1, (hi-lo)/20)
	}
	switch c.Type {
	case "range":
		v = min(max(v+step(c.Min, c.Max), c.Min), c.Max)
	case "toggle":
		v = 1 - min(v, 1)
	case "dropdown":
		if len(c.Options) == 0 {
			return m
		}
		i := slices.IndexFunc(c.Options, func(o defs.Option) bool { return o.Value == v })
		i = (max(i, 0) + dir + len(c.Options)) % len(c.Options)
		v = c.Options[i].Value
	case "color":
		hue, sat := v>>8, v&0xFF
		if r.sat {
			sat = min(max(sat+step(0, 255), 0), 255)
		} else {
			hue = (hue + step(0, 255) + 256) % 256
		}
		v = hue<<8 | sat
	default:
		return m
	}

	if err := m.dev.SetCustomValue(c.Channel, c.ValueID, c.Size(), v); err != nil {
		m.status = "WRITE FAILED: " + err.Error()
		return m
	}
	m.values[c.ID] = v
	m.dirty[c.Channel] = true
	m.status = ""
	// A new value can hide rows below it, e.g. effect "All Off" hides speed.
	m.row = max(0, min(m.row, len(m.menuRows())-1))
	return m
}

func (m model) rowText(r menuRow) (label, value string) {
	c := r.ctl
	label = strings.ToUpper(c.Label)
	if r.sat {
		label += " SATURATION"
	}
	v, ok := m.values[c.ID]
	if !ok {
		return label, "N/A"
	}
	switch c.Type {
	case "range":
		return label, percent(v, c.Min, c.Max)
	case "toggle":
		if v != 0 {
			return label, "ON"
		}
		return label, "OFF"
	case "dropdown":
		for _, o := range c.Options {
			if o.Value == v {
				return label, strings.ToUpper(o.Label)
			}
		}
		return label, strconv.Itoa(v)
	case "color":
		if r.sat {
			return label, percent(v&0xFF, 0, 255)
		}
		return label, fmt.Sprintf("██ %d°", (v>>8)*360/256)
	}
	return label, "UNSUPPORTED"
}

// rowDetail is the raw value, shown in the status line for the selected row.
func (m model) rowDetail(r menuRow) string {
	v, ok := m.values[r.ctl.ID]
	switch {
	case !ok:
		return "the board didn't return a value for " + r.ctl.ID
	case r.ctl.Type == "color":
		return fmt.Sprintf("HUE %d  SATURATION %d", v>>8, v&0xFF)
	case r.ctl.Type == "range":
		return fmt.Sprintf("%d (%d-%d)", v, r.ctl.Min, r.ctl.Max)
	}
	return strconv.Itoa(v)
}

// menuLines lays out the current menu tab and reports which line each row
// landed on, for mouse clicks.
func (m model) menuLines() (lines []string, rowLine []int) {
	rows := m.menuRows()
	for i, r := range rows {
		if (i == 0 || r.section != rows[i-1].section) && r.section != "" {
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, "\x1b[1m"+strings.ToUpper(r.section)+"\x1b[22m")
		}
		rowLine = append(rowLine, len(lines))

		label, value := m.rowText(r)
		label += strings.Repeat(".", max(2, labelWidth-len([]rune(label))))
		if i == m.row {
			label = "\x1b[7m" + label + "\x1b[27m"
		}
		cell := center(value, valueWidth)
		if r.ctl.Type == "color" && !r.sat {
			cell = strings.Replace(cell, "██", swatch(m.values[r.ctl.ID]), 1)
		}
		lines = append(lines, fmt.Sprintf("  %s < %s >", label, cell))
	}
	return lines, rowLine
}

func (m model) viewMenu(b *strings.Builder) {
	rows := m.menuRows()
	lines, _ := m.menuLines()
	if len(rows) == 0 {
		lines = []string{"  NOTHING TO SET"}
	}
	b.WriteString(strings.Join(lines, "\n") + "\n\n")
	status := m.status
	if status == "" && m.row < len(rows) {
		label, _ := m.rowText(rows[m.row])
		status = "> " + label + "  " + m.rowDetail(rows[m.row])
	}
	b.WriteString(status + "\n")
	b.WriteString("\x1b[2m↑↓ SELECT · ←→ CHANGE · SHIFT+←→ FINE · TAB SWITCH · Q QUIT\x1b[22m")
}

func percent(v, lo, hi int) string {
	if hi <= lo {
		return strconv.Itoa(v)
	}
	return fmt.Sprintf("%d%%", ((v-lo)*100+(hi-lo)/2)/(hi-lo))
}

func center(s string, width int) string {
	pad := max(0, width-len([]rune(s)))
	return strings.Repeat(" ", pad/2) + s + strings.Repeat(" ", pad-pad/2)
}

// swatch draws a truecolor block for a QMK color value: hue and saturation,
// each 0-255, at full brightness.
func swatch(v int) string {
	h := float64(v>>8) / 256 * 6
	c := float64(v&0xFF) / 255
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	rgb := [6][3]float64{{c, x, 0}, {x, c, 0}, {0, c, x}, {0, x, c}, {x, 0, c}, {c, 0, x}}[int(h)]
	ch := func(f float64) int { return int((f + 1 - c) * 255) }
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm██\x1b[39m", ch(rgb[0]), ch(rgb[1]), ch(rgb[2]))
}
