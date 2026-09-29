package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/bulkinglb/via-terminal/internal/macros"
)

// macroState is shared between model copies; the buffer is only read when
// the tab is first opened, since that takes a moment.
type macroState struct {
	count   int
	size    int      // buffer bytes
	list    []string // nil until loaded
	err     error
	sel     int
	editing bool
	edit    []rune
	cursor  int
}

// The list starts two lines below the tab bar, after the byte count.
const macroTop = bodyTop + 2

// macroTab is the macros' tab index, after the definition's menus.
func (m model) macroTab() int { return len(m.def.Menus) + 1 }

func (m model) loadMacros() {
	mc := m.macros
	if mc.list != nil || mc.err != nil || mc.count == 0 {
		return
	}
	size, err := m.dev.MacroBufferSize()
	if err == nil {
		var buf []byte
		if buf, err = m.dev.MacroBuffer(size); err == nil {
			mc.size = size
			mc.list, err = macros.Decode(buf, mc.count)
		}
	}
	mc.err = err
}

func (m model) updateMacros(s string) model {
	mc := m.macros
	switch s {
	case "up", "k":
		mc.sel = max(0, mc.sel-1)
	case "down", "j":
		mc.sel = max(0, min(len(mc.list)-1, mc.sel+1))
	case "enter":
		if mc.sel < len(mc.list) {
			mc.editing, mc.edit = true, []rune(mc.list[mc.sel])
			mc.cursor = len(mc.edit)
			m.status = ""
		}
	}
	return m
}

func (m model) clickMacros(y int) model {
	if i := y - macroTop; i >= 0 && i < len(m.macros.list) {
		m.macros.sel = i
	}
	return m
}

// updateMacroEdit is a one-line editor; every key types except the ones
// that move, delete, save or cancel.
func (m model) updateMacroEdit(msg tea.KeyPressMsg) model {
	mc := m.macros
	switch msg.String() {
	case "esc":
		mc.editing = false
		m.status = ""
	case "enter":
		return m.saveMacro()
	case "left":
		mc.cursor = max(0, mc.cursor-1)
	case "right":
		mc.cursor = min(len(mc.edit), mc.cursor+1)
	case "home", "ctrl+a":
		mc.cursor = 0
	case "end", "ctrl+e":
		mc.cursor = len(mc.edit)
	case "backspace":
		if mc.cursor > 0 {
			mc.edit = slices.Delete(mc.edit, mc.cursor-1, mc.cursor)
			mc.cursor--
		}
	case "delete":
		if mc.cursor < len(mc.edit) {
			mc.edit = slices.Delete(mc.edit, mc.cursor, mc.cursor+1)
		}
	default:
		if msg.Text != "" {
			mc.edit = slices.Insert(mc.edit, mc.cursor, []rune(msg.Text)...)
			mc.cursor += len([]rune(msg.Text))
		}
	}
	return m
}

// saveMacro writes every macro, since the board stores them back to back.
func (m model) saveMacro() model {
	mc := m.macros
	list := slices.Clone(mc.list)
	list[mc.sel] = string(mc.edit)
	buf, err := macros.Encode(list)
	switch {
	case err != nil:
		m.status = strings.ToUpper(err.Error())
		return m
	case len(buf) > mc.size:
		m.status = fmt.Sprintf("TOO LONG: THE MACROS NEED %d BYTES, THE BOARD HAS %d", len(buf), mc.size)
		return m
	}
	if err := m.dev.SetMacroBuffer(buf, mc.size); err != nil {
		m.status = "WRITE FAILED: " + err.Error()
		return m
	}
	// Reading back what was written shows the text the way it's stored,
	// e.g. {KC_LCTL, KC_C} as {LCTL,C}.
	mc.list, _ = macros.Decode(buf, mc.count)
	mc.editing = false
	m.status = fmt.Sprintf("SAVED M%d · MAP A KEY TO MC_%d IN MAPPING TO USE IT", mc.sel, mc.sel)
	return m
}

func (m model) viewMacros(b *strings.Builder) {
	mc := m.macros
	switch {
	case mc.count == 0:
		b.WriteString("THIS BOARD HAS NO MACROS\n")
		return
	case mc.err != nil:
		b.WriteString("COULDN'T READ THE MACROS: " + mc.err.Error() + "\n")
		return
	}
	used := 0
	if buf, err := macros.Encode(mc.list); err == nil {
		used = len(buf)
	}
	fmt.Fprintf(b, "\x1b[1mMACROS\x1b[22m  %d OF %d BYTES USED\n", used, mc.size)
	for i, text := range mc.list {
		if r := []rune(text); len(r) > 90 {
			text = string(r[:89]) + "…"
		}
		line := fmt.Sprintf("  M%-2d  %s", i, text)
		if i == mc.sel {
			line = fmt.Sprintf("  \x1b[7mM%-2d\x1b[27m  %s", i, text)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	if mc.editing {
		cur := " "
		if mc.cursor < len(mc.edit) {
			cur = string(mc.edit[mc.cursor])
		}
		after := ""
		if mc.cursor < len(mc.edit) {
			after = string(mc.edit[mc.cursor+1:])
		}
		fmt.Fprintf(b, "> M%d: %s\x1b[7m%s\x1b[27m%s\n%s\n", mc.sel, string(mc.edit[:mc.cursor]), cur, after, m.status)
		b.WriteString("\x1b[2mENTER SAVE · ESC CANCEL · ←→ HOME END MOVE\x1b[22m\n")
	} else {
		status := m.status
		if status == "" {
			status = fmt.Sprintf("> M%d  MAP A KEY TO MC_%d IN MAPPING TO USE IT", mc.sel, mc.sel)
		}
		b.WriteString(status + "\n")
		b.WriteString("\x1b[2m↑↓ SELECT · ENTER EDIT · TAB SWITCH · Q QUIT\x1b[22m\n")
	}
	b.WriteString("\x1b[2mTEXT IS TYPED AS-IS · {A} TAP · {LCTL,C} TOGETHER · {+LSFT} HOLD · {-LSFT} RELEASE · {100} WAIT MS · \\{ BRACE\x1b[22m")
}
