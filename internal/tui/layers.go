package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bulkinglb/via-terminal/internal/keycodes"
)

// step is one key on the way to a layer: its index in def.Keys and what to
// do with it.
type step struct {
	key  int
	verb string
}

// layerTarget reports which layer a keycode switches to and how.
func layerTarget(code uint16) (layer int, verb string, ok bool) {
	switch {
	case code >= 0x4000 && code < 0x5000:
		return int(code >> 8 & 0xF), "HOLD", true
	case code >= 0x5000 && code < 0x5200:
		return int(code >> 5 & 0xF), "HOLD", true
	case code >= 0x5200 && code < 0x5300:
		verb := map[uint16]string{0x5200: "PRESS", 0x5220: "HOLD", 0x5240: "PRESS", 0x5260: "TAP", 0x5280: "TAP", 0x52C0: "HOLD", 0x52E0: "PRESS"}[code&^0x1F]
		return int(code & 0x1F), verb, verb != ""
	}
	return 0, "", false
}

// route finds the keys that lead to layer target, starting from layer 0 if
// it can and from any other layer if not. from is -1 when no key leads there.
func (m model) route(target int) (from int, steps []step) {
	for from := range m.layers {
		if from == target {
			continue
		}
		type node struct {
			layer int
			path  []step
		}
		queue := []node{{from, nil}}
		seen := map[int]bool{from: true}
		for len(queue) > 0 {
			n := queue[0]
			queue = queue[1:]
			for i, k := range m.def.Keys {
				to, verb, ok := layerTarget(m.keymap[m.index(n.layer, k)])
				if !ok || seen[to] || to >= m.layers {
					continue
				}
				path := append(slices.Clone(n.path), step{i, verb})
				if to == target {
					return from, path
				}
				seen[to] = true
				queue = append(queue, node{to, path})
			}
		}
	}
	return -1, nil
}

// keyLabel names a physical key for the layer help: its layer 0 keycode,
// with keys that only switch layers called FN.
func (m model) keyLabel(i int) string {
	code := m.keymap[m.index(0, m.def.Keys[i])]
	if code >= 0x4000 && code < 0x5000 {
		return keycodes.Name(code&0xFF, nil)
	}
	if _, _, ok := layerTarget(code); ok {
		return "FN"
	}
	return keycodes.Name(code, m.def.Custom)
}

// layerHelp says how to reach the shown layer, like "HOLD FN + RSFT", and
// which keys to mark on the grid for it.
func (m model) layerHelp() (string, map[int]bool) {
	if m.layer == 0 {
		return "BASE LAYER", nil
	}
	from, steps := m.route(m.layer)
	if from < 0 {
		for _, k := range m.def.Keys {
			if code := m.keymap[m.index(m.layer, k)]; code > 0x0001 {
				return "NO KEY LEADS HERE, THE FIRMWARE SWITCHES IT (E.G. A WIN/MAC SWITCH)", nil
			}
		}
		return "UNUSED", nil
	}
	var parts []string
	marked := map[int]bool{}
	for i, s := range steps {
		part := m.keyLabel(s.key)
		if i == 0 || s.verb != steps[i-1].verb {
			part = s.verb + " " + part
		}
		parts = append(parts, part)
		marked[s.key] = true
	}
	text := strings.Join(parts, " + ")
	if from != 0 {
		text = fmt.Sprintf("FROM LAYER %d: %s", from, text)
	}
	return text, marked
}
