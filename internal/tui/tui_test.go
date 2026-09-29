package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"via-terminal/internal/defs"
	"via-terminal/internal/keycodes"
)

func TestRender(t *testing.T) {
	keys := []defs.Key{
		{X: 0, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{X: 1, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{X: 0, Y: 1, W: 2, H: 1, W2: 2, H2: 1},
	}
	names := []string{"a", "b", "c"}
	i := 0
	got := render(keys, func(defs.Key) string {
		i++
		return names[i-1]
	}, nil)
	want := strings.Join([]string{
		"┌─────┬─────┐",
		"│  a  │  b  │",
		"│     │     │",
		"├─────┴─────┤",
		"│     c     │",
		"│           │",
		"└───────────┘",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}

	for in, want := range map[string]string{
		"RGB_MOD": "RGB_|MOD",
		"BT DEV1": "BT|DEV1",
		"LT(1,A)": "LT(1,|A)",
		"ABCDEFG": "ABCDE|FG",
		"ESC":     "ESC",
	} {
		var parts []string
		for _, line := range wrap([]rune(in), 5) {
			parts = append(parts, string(line))
		}
		if got := strings.Join(parts, "|"); got != want {
			t.Errorf("wrap(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNearestKey(t *testing.T) {
	keys := []defs.Key{
		{X: 0, Y: 0, W: 1, H: 1},
		{X: 1, Y: 0, W: 1, H: 1},
		{X: 0, Y: 1, W: 2, H: 1},
		{X: 2.5, Y: 1.25, W: 1, H: 1},
	}
	for _, c := range []struct{ from, dx, dy, want int }{
		{0, 1, 0, 1},
		{1, 0, 1, 2},
		{2, 1, 0, 3},
		{0, -1, 0, 0},
	} {
		if got := nearestKey(keys, c.from, c.dx, c.dy); got != c.want {
			t.Errorf("nearestKey(%d, %d, %d) = %d, want %d", c.from, c.dx, c.dy, got, c.want)
		}
	}
}

func TestMenuRows(t *testing.T) {
	def, err := defs.Find(0x342D, 0xE4C2)
	if err != nil {
		t.Fatal(err)
	}
	m := model{def: def, tab: 1, values: map[string]int{
		"id_qmk_rgb_matrix_brightness":   255,
		"id_qmk_rgb_matrix_effect":       7,
		"id_qmk_rgb_matrix_effect_speed": 72,
		"id_qmk_rgb_matrix_color":        0xAA80,
	}}
	var got []string
	for _, r := range m.menuRows() {
		label, value := m.rowText(r)
		got = append(got, label+"="+value)
	}
	want := []string{"BRIGHTNESS=100%", "EFFECT=BAND VAL", "EFFECT SPEED=28%", "COLOR=██ 239°", "COLOR SATURATION=50%"}
	if !slices.Equal(got, want) {
		t.Errorf("rows:\n got %v\nwant %v", got, want)
	}

	m.values["id_qmk_rgb_matrix_effect"] = 0
	if n := len(m.menuRows()); n != 2 {
		t.Errorf("effect All Off should leave brightness and effect, got %d rows", n)
	}
	if swatch(0x00FF) != "\x1b[38;2;255;0;0m██\x1b[39m" || swatch(0x0000) != "\x1b[38;2;255;255;255m██\x1b[39m" {
		t.Errorf("swatch: red %q, white %q", swatch(0x00FF), swatch(0x0000))
	}
}

func TestPickerFilter(t *testing.T) {
	var keys []choice
	for _, k := range keycodes.Picker(2, nil) {
		keys = append(keys, choice{k.Name, k.Long, int(k.Code)})
	}
	for query, want := range map[string]int{"kc_esc": 0x29, "space": 0x2C, "mo(1": 0x5221} {
		p := newPicker("", keys, 0, nil)
		p.query = query
		p.filter()
		if len(p.matches) == 0 || p.matches[0].value != want {
			t.Errorf("%q should rank 0x%04X first, got %v", query, want, p.matches[:min(3, len(p.matches))])
		}
	}

	effects := []choice{{"SOLID COLOR", "", 1}, {"BREATHING", "", 5}, {"SOLID REACTIVE", "", 34}}
	p := newPicker("", effects, 34, nil)
	if p.pick != 2 {
		t.Errorf("the current value should start selected, got %d", p.pick)
	}
	p.query = "solid"
	p.filter()
	if len(p.matches) != 2 || p.pick != 0 {
		t.Errorf("solid: %v, pick %d", p.matches, p.pick)
	}
}

func TestLightFrame(t *testing.T) {
	def, err := defs.Find(0x342D, 0xE4C2)
	if err != nil {
		t.Fatal(err)
	}
	colors := func(effect string, at time.Duration) []string {
		return lightFrame(def.Keys, rgbSettings{effect: effect, hue: 0, sat: 255, val: 255, speed: 128}, at, nil)
	}
	for _, c := range colors("Solid Color", time.Second) {
		if c != "\x1b[48;2;255;0;0m" {
			t.Fatalf("solid red should color every key red, got %q", c)
		}
	}
	for _, c := range colors("All Off", time.Second) {
		if c != "\x1b[48;2;0;0;0m" {
			t.Fatalf("all off should be black, got %q", c)
		}
	}
	cycle := colors("Cycle Left Right", time.Second)
	if cycle[0] == cycle[13] {
		t.Error("cycle left right should color the left and right of the board differently")
	}
	if later := colors("Cycle Left Right", 1500*time.Millisecond); later[0] == cycle[0] {
		t.Error("the animation should move over time")
	}
	if effectName("Pinwheel Sat.") != effectName("Band Pinwheel Sat") {
		t.Error("VIA and QMK effect names should match")
	}

	// Every effect of the board except MonsGeek's own "Close All" is simulated,
	// and the random and reactive ones light something without real input.
	for _, o := range def.Menus[0].Items[2].Options {
		if o.Label == "Close All" {
			continue
		}
		if !simulated(o.Label) {
			t.Errorf("%s isn't simulated", o.Label)
		}
		if o.Label == "All Off" {
			continue
		}
		// 150ms after a simulated press, so single-press splashes have grown.
		a, b := colors(o.Label, 3150*time.Millisecond), colors(o.Label, 3150*time.Millisecond)
		if !slices.Equal(a, b) {
			t.Errorf("%s should draw the same frame for the same time", o.Label)
		}
		if !slices.ContainsFunc(a, func(c string) bool { return c != "\x1b[48;2;0;0;0m" }) {
			t.Errorf("%s leaves the whole board dark", o.Label)
		}
	}
}
