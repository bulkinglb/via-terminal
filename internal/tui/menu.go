package tui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bulkinglb/via-terminal/internal/defs"
	"github.com/bulkinglb/via-terminal/internal/keycodes"
)

const labelWidth, valueWidth = 30, 26

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
	case "enter", "space":
		return m.choose(rows)
	}
	return m
}

func (m model) clickMenu(x, y int) model {
	rows := m.menuRows()
	lines, rowLine := m.menuLines()
	if m.picker != nil {
		return m.clickPicker(y, bodyTop+len(lines)+1)
	}
	for i, line := range rowLine {
		if y-bodyTop != line {
			continue
		}
		m.row = i
		label, _ := m.rowText(rows[i])
		lt := 2 + max(labelWidth, len([]rune(label))+2) + 1
		gt := lt + 2 + valueWidth + 1
		switch {
		case abs(x-lt) <= 1:
			return m.adjust(rows, -1, false)
		case abs(x-gt) <= 1:
			return m.adjust(rows, 1, false)
		case x > lt && x < gt:
			return m.choose(rows)
		}
		return m
	}
	return m
}

// choose opens the search list for dropdowns and keycodes, and flips
// toggles.
func (m model) choose(rows []menuRow) model {
	if m.row >= len(rows) {
		return m
	}
	c := rows[m.row].ctl
	if c.Type == "button" {
		return m.setValue(c, c.Options[0].Value)
	}
	v, ok := m.values[c.ID]
	if !ok {
		return m
	}
	apply := func(m model, v int) model { return m.setValue(c, v) }
	switch c.Type {
	case "dropdown":
		var choices []choice
		for _, o := range c.Options {
			choices = append(choices, choice{label: strings.ToUpper(o.Label), value: o.Value})
		}
		m.picker = newPicker(strings.ToUpper(c.Label), choices, v, apply)
	case "keycode":
		m.picker = newPicker(strings.ToUpper(c.Label), m.keycodeChoices, v, apply)
	case "toggle":
		return m.adjust(rows, 1, false)
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
	case "dropdown", "toggle":
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
	return m.setValue(c, v)
}

func (m model) setValue(c defs.Control, v int) model {
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
	switch {
	case c.Type == "button":
		return label, "PRESS"
	case c.Type == "label":
		return label, strings.ToUpper(c.Text)
	case !c.HasValue():
		return label, "UNSUPPORTED"
	}
	v, ok := m.values[c.ID]
	if !ok {
		return label, "N/A"
	}
	switch c.Type {
	case "range":
		return label, percent(v, c.Min, c.Max)
	case "dropdown", "toggle":
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
	case "keycode":
		return label, keycodes.Name(uint16(v), m.def.Custom)
	}
	return label, "UNSUPPORTED"
}

// rowDetail is the raw value, shown in the status line for the selected row.
func (m model) rowDetail(r menuRow) string {
	v, ok := m.values[r.ctl.ID]
	switch {
	case r.ctl.Type == "button":
		return "ENTER OR CLICK TO SEND"
	case !r.ctl.HasValue():
		return ""
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
	if m.picker != nil {
		m.viewPicker(b)
		return
	}
	status := m.status
	if status == "" && m.row < len(rows) {
		label, _ := m.rowText(rows[m.row])
		status = "> " + label + "  " + m.rowDetail(rows[m.row])
	}
	b.WriteString(status + "\n")
	b.WriteString("\x1b[2m↑↓ SELECT · ←→ CHANGE · SHIFT+←→ FINE · ENTER/CLICK LIST · TAB SWITCH · Q QUIT\x1b[22m")

	if s, ok := m.rgbMatrix(); ok {
		colors := lightFrame(m.def.Keys, s, m.now.Sub(m.start), m.mods())
		b.WriteString("\n\n" + render(m.def.Keys, func(defs.Key) string { return "" }, func(i int) string { return colors[i] }))
		switch {
		case !simulated(s.effect):
			b.WriteString("\n\x1b[2mPREVIEW SHOWS THE PLAIN COLOR, THIS EFFECT ISN'T SIMULATED\x1b[22m")
		case reactive(s.effect):
			b.WriteString("\n\x1b[2mPREVIEW TYPES BY ITSELF TO SHOW THIS EFFECT\x1b[22m")
		}
	}
}

// mods marks keys whose layer 0 keycode isn't a letter, digit, punctuation
// or space; QMK boards usually flag those as modifiers for Alphas Mods.
func (m model) mods() []bool {
	mods := make([]bool, len(m.def.Keys))
	for i, k := range m.def.Keys {
		c := m.keymap[m.index(0, k)]
		mods[i] = !(c >= 0x04 && c <= 0x27 || c >= 0x2C && c <= 0x38)
	}
	return mods
}

func percent(v, lo, hi int) string {
	if hi <= lo {
		return strconv.Itoa(v)
	}
	return fmt.Sprintf("%d%%", ((v-lo)*100+(hi-lo)/2)/(hi-lo))
}

// center pads s to width, cutting it if it's longer so the arrows after it
// stay where mouse clicks expect them.
func center(s string, width int) string {
	r := []rune(s)
	r = r[:min(len(r), width)]
	pad := width - len(r)
	return strings.Repeat(" ", pad/2) + string(r) + strings.Repeat(" ", pad-pad/2)
}

// swatch draws a truecolor block for a QMK color value: hue and saturation,
// each 0-255, at full brightness.
func swatch(v int) string {
	r, g, b := hsv(float64(v>>8)/256, float64(v&0xFF)/255, 1)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm██\x1b[39m", r, g, b)
}
