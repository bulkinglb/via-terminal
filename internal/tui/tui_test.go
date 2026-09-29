package tui

import (
	"strings"
	"testing"

	"via-terminal/internal/defs"
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
	}, -1)
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
