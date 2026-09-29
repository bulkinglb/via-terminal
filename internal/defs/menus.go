package defs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Menu is a top-level entry of a definition's "menus", such as Lighting.
type Menu struct {
	Label string
	Items []Control // in order; an item with an empty Type is a section heading
}

// Control is one setting from a VIA menu. The board stores its value under
// Channel and ValueID; ID is how showIf conditions refer to it.
type Control struct {
	Label, Type, ID, ShowIf string
	Channel, ValueID        byte
	Min, Max                int      // range
	Options                 []Option // dropdown, toggle (off, on), button (the value it sends)
	Text                    string   // label: fixed text
}

// HasValue reports whether the board stores a value for the control that
// can be read and changed.
func (c Control) HasValue() bool {
	switch c.Type {
	case "range", "dropdown", "toggle", "color", "keycode":
		return true
	}
	return false
}

type Option struct {
	Label string
	Value int
}

// Size is how many bytes the board uses for the value: colors are hue and
// saturation, keycodes are 16-bit, ranges past 255 need two bytes.
func (c Control) Size() int {
	if c.Type == "color" || c.Type == "keycode" || c.Max > 255 {
		return 2
	}
	return 1
}

type rawItem struct {
	Label, Type, ShowIf string
	Content             json.RawMessage
	Options             []json.RawMessage
}

// parseMenus expands preset names like "qmk_rgb_matrix" and skips names it
// doesn't know, as VIA does.
func parseMenus(raw []json.RawMessage) ([]Menu, error) {
	var menus []Menu
	for _, r := range raw {
		var preset string
		if json.Unmarshal(r, &preset) == nil {
			menus = append(menus, presets[preset]...)
			continue
		}
		var m rawItem
		if err := json.Unmarshal(r, &m); err != nil {
			return nil, fmt.Errorf("menus: %w", err)
		}
		items, err := parseItems(m.Content)
		if err != nil {
			return nil, fmt.Errorf("menu %q: %w", m.Label, err)
		}
		menus = append(menus, Menu{m.Label, items})
	}
	return menus, nil
}

// parseItems flattens nested sections into headings followed by controls.
func parseItems(content json.RawMessage) ([]Control, error) {
	var raw []rawItem
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, err
	}
	var items []Control
	for _, it := range raw {
		if it.Type == "" {
			sub, err := parseItems(it.Content)
			if err != nil {
				return nil, fmt.Errorf("%q: %w", it.Label, err)
			}
			items = append(append(items, Control{Label: it.Label}), sub...)
			continue
		}
		c, err := parseControl(it)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", it.Label, err)
		}
		items = append(items, c)
	}
	return items, nil
}

// parseControl only insists on [id, channel, value id] content for controls
// that talk to the board, so a label or a type VIA added later still loads.
func parseControl(it rawItem) (Control, error) {
	c := Control{Label: it.Label, Type: it.Type, ShowIf: it.ShowIf, Max: 255}
	var content []any
	json.Unmarshal(it.Content, &content)
	if len(content) == 1 && it.Type == "label" {
		c.Text, _ = content[0].(string)
		return c, nil
	}
	var chOK, idOK bool
	var channel, valueID float64
	if len(content) >= 3 {
		c.ID, _ = content[0].(string)
		channel, chOK = content[1].(float64)
		valueID, idOK = content[2].(float64)
		c.Channel, c.ValueID = byte(channel), byte(valueID)
	}
	if (c.HasValue() || it.Type == "button") && (c.ID == "" || !chOK || !idOK) {
		return Control{}, fmt.Errorf("content should be [id, channel, value id], got %s", it.Content)
	}

	switch it.Type {
	case "range":
		if len(it.Options) == 2 {
			json.Unmarshal(it.Options[0], &c.Min)
			json.Unmarshal(it.Options[1], &c.Max)
		}
	case "dropdown":
		// Options are either labels, valued by position, or [label, value].
		for i, o := range it.Options {
			var label string
			if json.Unmarshal(o, &label) == nil {
				c.Options = append(c.Options, Option{label, i})
				continue
			}
			var pair []any
			if json.Unmarshal(o, &pair) != nil || len(pair) != 2 {
				return Control{}, fmt.Errorf("dropdown option %s: want [label, value]", o)
			}
			label, _ = pair[0].(string)
			value, ok := pair[1].(float64)
			if !ok {
				return Control{}, fmt.Errorf("dropdown option %s: want [label, value]", o)
			}
			c.Options = append(c.Options, Option{label, int(value)})
		}
	case "toggle":
		// A toggle is a two-option list; options can set the off and on values.
		off, on := 0, 1
		if len(it.Options) == 2 {
			json.Unmarshal(it.Options[0], &off)
			json.Unmarshal(it.Options[1], &on)
		}
		c.Options = []Option{{"Off", off}, {"On", on}}
	case "button":
		// A button has no state; pressing it sends options[0], or 1.
		press := 1
		if len(it.Options) == 1 {
			json.Unmarshal(it.Options[0], &press)
		}
		c.Options = []Option{{"Press", press}}
	}
	return c, nil
}

// Visible evaluates the showIf condition, e.g. "{id_effect} != 0 &&
// {id_effect} != 24", against current values by control ID. A condition it
// can't read shows the control rather than hiding it.
func (c Control) Visible(values map[string]int) bool {
	if c.ShowIf == "" {
		return true
	}
	p := &cond{src: c.ShowIf, values: values}
	v, ok := p.or()
	p.space()
	return !ok || p.pos != len(p.src) || v != 0
}

type cond struct {
	src    string
	pos    int
	values map[string]int
}

func (p *cond) space() {
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
}

func (p *cond) eat(tok string) bool {
	p.space()
	if strings.HasPrefix(p.src[p.pos:], tok) {
		p.pos += len(tok)
		return true
	}
	return false
}

func (p *cond) or() (int, bool) {
	v, ok := p.and()
	for ok && p.eat("||") {
		w, wok := p.and()
		v, ok = b2i(v != 0 || w != 0), wok
	}
	return v, ok
}

func (p *cond) and() (int, bool) {
	v, ok := p.cmp()
	for ok && p.eat("&&") {
		w, wok := p.cmp()
		v, ok = b2i(v != 0 && w != 0), wok
	}
	return v, ok
}

func (p *cond) cmp() (int, bool) {
	v, ok := p.atom()
	if !ok {
		return 0, false
	}
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.eat(op) {
			w, wok := p.atom()
			r := map[string]bool{"==": v == w, "!=": v != w, "<=": v <= w, ">=": v >= w, "<": v < w, ">": v > w}[op]
			return b2i(r), wok
		}
	}
	return v, true
}

func (p *cond) atom() (int, bool) {
	switch {
	case p.eat("("):
		v, ok := p.or()
		return v, ok && p.eat(")")
	case p.eat("!"):
		v, ok := p.atom()
		return b2i(v == 0), ok
	case p.eat("{"):
		end := strings.IndexByte(p.src[p.pos:], '}')
		if end < 0 {
			return 0, false
		}
		id := p.src[p.pos : p.pos+end]
		p.pos += end + 1
		return p.values[id], true
	}
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
		p.pos++
	}
	n, err := strconv.Atoi(p.src[start:p.pos])
	return n, err == nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
