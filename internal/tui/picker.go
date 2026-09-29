package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const pickRows = 10

// choice is one picker entry: a keycode or a menu option.
type choice struct {
	label, detail string
	value         int
}

// picker is the searchable list for remapping keys and for menu settings
// with many options. apply receives the chosen value.
type picker struct {
	prompt  string
	all     []choice
	matches []choice
	query   string
	pick    int
	width   int
	apply   func(m model, value int) model
}

func newPicker(prompt string, all []choice, current int, apply func(model, int) model) *picker {
	p := &picker{prompt: prompt, all: all, matches: all, apply: apply}
	for i, c := range all {
		p.width = max(p.width, min(28, len([]rune(c.label))))
		if c.value == current {
			p.pick = i
		}
	}
	return p
}

// filter ranks exact matches first, then prefixes, then substrings, keeping
// the original order within each group. KC_ is optional for keycodes.
func (p *picker) filter() {
	p.pick = 0
	q := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(p.query)), "KC_")
	if q == "" {
		p.matches = p.all
		return
	}
	var exact, prefix, rest []choice
	for _, c := range p.all {
		label, detail := strings.ToUpper(c.label), strings.ToUpper(c.detail)
		switch {
		case label == q || detail == q:
			exact = append(exact, c)
		case strings.HasPrefix(label, q) || strings.HasPrefix(detail, q):
			prefix = append(prefix, c)
		case strings.Contains(label, q) || strings.Contains(detail, q):
			rest = append(rest, c)
		}
	}
	p.matches = slices.Concat(exact, prefix, rest)
}

func (p *picker) window() (start, end int) {
	start = max(0, p.pick-pickRows+1)
	return start, min(len(p.matches), start+pickRows)
}

func (m model) updatePicker(msg tea.KeyPressMsg) model {
	p := m.picker
	switch msg.String() {
	case "esc":
		m.picker = nil
	case "enter":
		if len(p.matches) > 0 {
			m.picker = nil
			return p.apply(m, p.matches[p.pick].value)
		}
	case "up":
		p.pick = max(0, p.pick-1)
	case "down":
		p.pick = max(0, min(len(p.matches)-1, p.pick+1))
	case "backspace":
		if r := []rune(p.query); len(r) > 0 {
			p.query = string(r[:len(r)-1])
			p.filter()
		}
	default:
		if msg.Text != "" {
			p.query += msg.Text
			p.filter()
		}
	}
	return m
}

// clickPicker applies the entry at screen row y. The list starts on the
// line after the prompt, which sits at promptY.
func (m model) clickPicker(y, promptY int) model {
	p := m.picker
	start, end := p.window()
	if i := start + y - promptY - 1; i >= start && i < end {
		m.picker = nil
		return p.apply(m, p.matches[i].value)
	}
	return m
}

func (m model) viewPicker(b *strings.Builder) {
	p := m.picker
	fmt.Fprintf(b, "> %s: %s_\n", p.prompt, p.query)
	start, end := p.window()
	for i := start; i < end; i++ {
		line := fmt.Sprintf("  %-*s  %s", p.width, p.matches[i].label, p.matches[i].detail)
		if i == p.pick {
			line = "\x1b[7m" + line + "\x1b[27m"
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\x1b[2mTYPE TO SEARCH · ENTER/CLICK PICK · ESC CANCEL\x1b[22m")
}
