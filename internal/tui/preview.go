package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"via-terminal/internal/defs"
)

// The lighting preview recomputes RGB matrix effects on the key grid, since
// VIA can't read LED colors back from the board. Effects are found by name
// because firmware numbers them by whichever effects it was built with.
// Random and key-reactive effects aren't simulated and show the plain color.

// rgbSettings are the RGB matrix values from a menu tab.
type rgbSettings struct {
	effect               string
	hue, sat, val, speed int
}

// rgbMatrix finds the RGB matrix settings on the current menu tab by their
// protocol address, channel 3 values 1-4, whatever the definition calls them.
func (m model) rgbMatrix() (rgbSettings, bool) {
	s := rgbSettings{sat: 255, val: 255, speed: 128}
	found := false
	if m.tab == 0 {
		return s, false
	}
	for _, c := range m.def.Menus[m.tab-1].Items {
		v, ok := m.values[c.ID]
		if c.Channel != 3 || !ok {
			continue
		}
		switch c.ValueID {
		case 1:
			s.val = v
		case 2:
			found = true
			for _, o := range c.Options {
				if o.Value == v {
					s.effect = o.Label
				}
			}
		case 3:
			s.speed = v
		case 4:
			s.hue, s.sat = v>>8, v&0xFF
		}
	}
	return s, found
}

type tickMsg int // the tick loop it belongs to, see model.tickGen

// tick keeps the preview moving while a tab with RGB matrix settings is
// open. Switching tabs bumps tickGen, which ends the old loop.
func (m model) tick() tea.Cmd {
	if _, ok := m.rgbMatrix(); !ok {
		return nil
	}
	gen := m.tickGen
	return tea.Tick(time.Second/20, func(time.Time) tea.Msg { return tickMsg(gen) })
}

// position is a key center: x and y from 0 to 1 across the board, distance
// and angle (in turns) from its middle. Distances use QMK's 224x64 LED
// space so round effects keep the board's aspect ratio.
type position struct{ x, y, dist, angle float64 }

// Each effect moves one of hue (in turns), saturation or brightness (as a
// 0-1 factor) for a key at phase c, which counts full cycles over time.
var (
	hueEffects = map[string]func(p position, c float64) float64{
		"gradientupdown":       func(p position, c float64) float64 { return p.y / 2 },
		"gradientleftright":    func(p position, c float64) float64 { return p.x / 2 },
		"cycleall":             func(p position, c float64) float64 { return c },
		"cycleleftright":       func(p position, c float64) float64 { return p.x - c },
		"cycleupdown":          func(p position, c float64) float64 { return p.y - c },
		"rainbowmovingchevron": func(p position, c float64) float64 { return p.x + math.Abs(p.y-0.5)/2 - c },
		"cycleoutin":           func(p position, c float64) float64 { return p.dist - c },
		"cycleoutindual":       func(p position, c float64) float64 { return math.Abs(math.Abs(p.x-0.5)-0.25)*2 - c },
		"cyclepinwheel":        func(p position, c float64) float64 { return p.angle - c },
		"cyclespiral":          func(p position, c float64) float64 { return p.dist - p.angle - c },
		"dualbeacon":           func(p position, c float64) float64 { return beacon(p, c) / 2 },
		"rainbowbeacon":        func(p position, c float64) float64 { return beacon(p, c) },
		"rainbowpinwheels":     func(p position, c float64) float64 { return 2*p.angle - c },
		"huebreathing":         func(p position, c float64) float64 { return 0.05 * math.Sin(2*math.Pi*c) },
		"huependulum":          func(p position, c float64) float64 { return 0.05 * math.Sin(2*math.Pi*c) * (2*p.x - 1) },
		"huewave":              func(p position, c float64) float64 { return 0.05 * math.Sin(2*math.Pi*(p.x-c)) },
	}
	satEffects = map[string]func(p position, c float64) float64{
		"bandsat":         func(p position, c float64) float64 { return band(p.x - c) },
		"bandpinwheelsat": func(p position, c float64) float64 { return 1 - frac(p.angle-c) },
		"bandspiralsat":   func(p position, c float64) float64 { return 1 - frac(p.angle+p.dist-c) },
	}
	valEffects = map[string]func(p position, c float64) float64{
		"breathing":       func(p position, c float64) float64 { return (math.Sin(2*math.Pi*c) + 1) / 2 },
		"bandval":         func(p position, c float64) float64 { return band(p.x - c) },
		"bandpinwheelval": func(p position, c float64) float64 { return 1 - frac(p.angle-c) },
		"bandspiralval":   func(p position, c float64) float64 { return 1 - frac(p.angle+p.dist-c) },
	}
)

// beacon is the distance along a line through the middle that turns with c.
func beacon(p position, c float64) float64 {
	a := 2 * math.Pi * c
	return (p.x-0.5)*math.Cos(a) + (p.y-0.5)*math.Sin(a)
}

// band is 1 at whole numbers, fading to 0 a quarter away.
func band(d float64) float64 { return max(0, 1-4*math.Abs(d-math.Round(d))) }

func frac(f float64) float64 { return f - math.Floor(f) }

// effectName normalizes labels so "Band Pinwheel Sat" (QMK's name) and
// "Pinwheel Sat." (VIA's) both find the same effect.
func effectName(label string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(label) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	name := b.String()
	if strings.HasPrefix(name, "pinwheel") || strings.HasPrefix(name, "spiral") {
		name = "band" + name
	}
	return name
}

func simulated(effect string) bool {
	name := effectName(effect)
	_, h := hueEffects[name]
	_, s := satEffects[name]
	_, v := valEffects[name]
	return h || s || v || name == "solidcolor" || name == "alloff"
}

// lightFrame returns a background color escape for every key at elapsed
// time into the animation.
func lightFrame(keys []defs.Key, s rgbSettings, elapsed time.Duration) []string {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, k := range keys {
		minX, maxX = min(minX, k.X+k.W/2), max(maxX, k.X+k.W/2)
		minY, maxY = min(minY, k.Y+k.H/2), max(maxY, k.Y+k.H/2)
	}
	// QMK's timing: at the default speed of 128 a full cycle takes about 2s.
	c := float64(elapsed.Milliseconds()) * float64(s.speed/4+1) / 65536
	name := effectName(s.effect)

	colors := make([]string, len(keys))
	for i, k := range keys {
		x := (k.X + k.W/2 - minX) / max(maxX-minX, 1)
		y := (k.Y + k.H/2 - minY) / max(maxY-minY, 1)
		dx, dy := (x-0.5)*224, (y-0.5)*64
		p := position{x, y, math.Hypot(dx, dy) / 116, math.Atan2(dy, dx)/(2*math.Pi) + 0.5}

		h, sat, val := float64(s.hue)/256, float64(s.sat)/255, float64(s.val)/255
		if f, ok := hueEffects[name]; ok {
			h += f(p, c)
		}
		if f, ok := satEffects[name]; ok {
			sat *= f(p, c)
		}
		if f, ok := valEffects[name]; ok {
			val *= f(p, c)
		}
		if name == "alloff" {
			val = 0
		}
		r, g, b := hsv(h, sat, val)
		colors[i] = fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	}
	return colors
}

// hsv converts hue (in turns), saturation and value (0-1) to 8-bit RGB.
func hsv(h, s, v float64) (r, g, b int) {
	h = frac(h) * 6
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	rgb := [6][3]float64{{c, x, 0}, {x, c, 0}, {0, c, x}, {0, x, c}, {x, 0, c}, {c, 0, x}}[int(h)%6]
	m := v - c
	return int((rgb[0] + m) * 255), int((rgb[1] + m) * 255), int((rgb[2] + m) * 255)
}
