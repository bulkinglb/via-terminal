package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/bulkinglb/via-terminal/internal/keycodes"
)

// newReference builds the keycode reference: how to read keycodes that take
// arguments, then every keycode the picker offers, each with what it does.
func newReference(layers int, custom []string) *picker {
	var all []choice
	for _, f := range keycodes.Forms {
		all = append(all, choice{label: f.Name, value: -1, desc: f.Desc})
	}
	for _, k := range keycodes.Picker(layers, custom) {
		all = append(all, choice{k.Name, k.Long, int(k.Code), k.Desc})
	}
	p := newPicker("SEARCH", all, -2, nil)
	p.rows = 20
	return p
}

// refTab is the reference's tab index, the last one.
func (m model) refTab() int { return len(m.def.Menus) + 2 }

// updateReference types every key into the search, q included, so only
// Ctrl+C quits here.
func (m model) updateReference(msg tea.KeyPressMsg) model {
	if msg.String() == "esc" {
		m.ref.query = ""
		m.ref.filter()
		return m
	}
	m.ref.key(msg)
	return m
}

func (m model) viewReference(b *strings.Builder) {
	m.ref.view(b, "TYPE TO SEARCH · ↑↓ PGUP PGDN SCROLL · ESC CLEAR · TAB SWITCH · CTRL+C QUIT")
}
