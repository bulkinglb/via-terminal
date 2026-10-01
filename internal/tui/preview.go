package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bulkinglb/via-terminal/internal/defs"
)

// The lighting preview recomputes RGB matrix effects on the key grid, since
// VIA can't read LED colors back from the board. Effects are found by name
// because firmware numbers them by whichever effects it was built with.
// Random effects use hashes of key and time rather than real randomness, so
// every frame is a pure function of time. Unknown effects show the color.

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
	if m.tab == 0 || m.tab > len(m.def.Menus) {
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

// pixel is one key for the effects that need more than its position.
type pixel struct {
	position
	i      int
	lx, ly float64 // QMK LED coordinates, 0-224 by 0-64
	mod    bool    // a modifier or other non-alpha key
}

// hit is a key press. The preview types by itself every hitEvery, so key
// reactive effects have something to show without real typing, which the
// lighting tab uses for its controls.
type hit struct {
	key    int
	lx, ly float64
	age    float64 // seconds
}

const hitEvery = 0.3 // seconds

type frame struct {
	t     float64 // seconds into the animation
	fade  float64 // seconds a press stays visible; QMK shortens it with speed
	speed int
	hits  []hit // newest first
}

type effectFunc = func(k pixel, f frame, h, s, v float64) (float64, float64, float64)

// Effects that don't fit a single hue, saturation or brightness change get
// the whole color and return it.
var fullEffects = map[string]effectFunc{
	// Gradients don't move; speed sets how far the hue spreads, as in QMK.
	"gradientupdown": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		return h + float64(64*f.speed/256)*math.Floor(k.ly/16)/256, s, v
	},
	"gradientleftright": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		return h + math.Floor(float64(64*f.speed/256)*k.lx/32)/256, s, v
	},
	"alphasmods": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		if k.mod {
			h += float64(f.speed) / 256
		}
		return h, s, v
	},
	"raindrops": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		return h + (reroll(k.i, f, 1)-0.5)/2, s, v
	},
	"jellybeanraindrops": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		return reroll(k.i, f, 1), reroll(k.i, f, 2), v * reroll(k.i, f, 3)
	},
	"pixelrain": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		if reroll(k.i, f, 2) < 0.25 {
			v = 0
		}
		return reroll(k.i, f, 1), s, v
	},
	"pixelflow": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		// Colors enter at the first key and move one key along per step.
		n := int(f.t*float64(2+f.speed/16)) - k.i
		if rnd(n, 2) < 0.3 {
			v = 0
		}
		return rnd(n, 1), s, v
	},
	"pixelfractal": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		// Mirrored left and right, with the pattern moving out from the middle.
		row, out := int(k.y*5+0.5), int(math.Abs(k.x-0.5)*16)
		if rnd(row, out-int(f.t*float64(2+f.speed/32))) < 0.5 {
			v = 0
		}
		return h, s, v
	},
	"digitalrain": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		col := int(k.x * 16)
		head := frac(f.t*0.5+rnd(col, 1))*1.6 - 0.3
		d := head - k.y
		if d < 0 || d > 0.5 {
			return 1.0 / 3, 1, 0
		}
		// Green, with a white head.
		return 1.0 / 3, min(1, d*8), v * (1 - d/0.5)
	},
	"typingheatmap": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		heat := 0.0
		for _, p := range f.hits {
			d := math.Hypot(k.lx-p.lx, k.ly-p.ly)
			heat += max(0, 1-p.age/5) * max(0, 1-d/40) / 2
		}
		heat = min(1, heat)
		return 170.0 / 256 * (1 - heat), 1, v * min(1, heat*3)
	},
	"solidreactivesimple": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		return h, s, v * pressed(k, f)
	},
	"solidreactive": func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		return h + pressed(k, f)/2, s, v
	},
	"solidreactivewide":       spread(false, wide),
	"solidreactivemultiwide":  spread(true, wide),
	"solidreactivecross":      spread(false, cross),
	"solidreactivemulticross": spread(true, cross),
	"solidreactivenexus":      spread(false, nexus),
	"solidreactivemultinexus": spread(true, nexus),
	"splash":                  rainbow(spread(false, ring)),
	"multisplash":             rainbow(spread(true, ring)),
	"solidsplash":             spread(false, ring),
	"solidmultisplash":        spread(true, ring),
}

// pressed is how lit key k still is from its own latest press.
func pressed(k pixel, f frame) float64 {
	for _, p := range f.hits {
		if p.key == k.i {
			return max(0, 1-p.age/f.fade)
		}
	}
	return 0
}

// spread lights keys around presses on a dark board, shaped by shape: the
// newest press only, or every recent one for the multi variants.
func spread(multi bool, shape func(dx, dy, d, q float64) float64) effectFunc {
	return func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		lit := 0.0
		for i, p := range f.hits {
			if i > 0 && !multi {
				break
			}
			if q := p.age / f.fade; q < 1 {
				dx, dy := math.Abs(k.lx-p.lx), math.Abs(k.ly-p.ly)
				lit += shape(dx, dy, math.Hypot(dx, dy), q)
			}
		}
		return h, s, v * min(1, lit)
	}
}

// rainbow shifts the hue with how far a key has faded, for the rainbow
// splashes.
func rainbow(effect effectFunc) effectFunc {
	return func(k pixel, f frame, h, s, v float64) (float64, float64, float64) {
		h2, s2, v2 := effect(k, f, h, s, v)
		return h2 + (1-v2/max(v, 0.01))/2, s2, v2
	}
}

// Shapes for spread. d, dx and dy are LED units from the press, q runs from
// 0 to 1 over the fade.
func wide(dx, dy, d, q float64) float64 { return max(0, 1-q-d/51) }

func cross(dx, dy, d, q float64) float64 {
	if dx > 8 && dy > 8 {
		return 0
	}
	return max(0, 1-q-d/224)
}

func nexus(dx, dy, d, q float64) float64 {
	if dx > 8 && dy > 8 || d > 72 {
		return 0
	}
	return max(0, 1-math.Abs(q*72-d)/16)
}

func ring(dx, dy, d, q float64) float64 {
	r := q * 255
	if d > r {
		return 0
	}
	return max(0, 1-(r-d)/64)
}

// reroll gives key i a new random number every so often, at a per-key
// moment, faster at higher speeds.
func reroll(i int, f frame, salt int) float64 {
	period := 20 / (float64(f.speed)/16 + 4)
	return rnd(i, int(math.Floor(f.t/period+rnd(i, 0))), salt)
}

// rnd hashes its inputs to a repeatable number in [0, 1), so random effects
// need no state between frames.
func rnd(nums ...int) float64 {
	h := uint64(0x9E3779B97F4A7C15)
	for _, n := range nums {
		h ^= uint64(n)
		h *= 0xBF58476D1CE4E5B9
		h ^= h >> 31
	}
	return float64(h>>11) / (1 << 53)
}

func simulated(effect string) bool {
	name := effectName(effect)
	_, h := hueEffects[name]
	_, s := satEffects[name]
	_, v := valEffects[name]
	_, full := fullEffects[name]
	return h || s || v || full || name == "solidcolor" || name == "alloff"
}

// reactive effects answer key presses, which the preview makes up.
func reactive(effect string) bool {
	name := effectName(effect)
	return strings.HasPrefix(name, "solidreactive") || strings.Contains(name, "splash") || name == "typingheatmap"
}

// lightFrame returns a background color escape for every key at elapsed
// time into the animation. mods marks the keys Alphas Mods colors apart.
func lightFrame(keys []defs.Key, s rgbSettings, elapsed time.Duration, mods []bool) []string {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, k := range keys {
		minX, maxX = min(minX, k.X+k.W/2), max(maxX, k.X+k.W/2)
		minY, maxY = min(minY, k.Y+k.H/2), max(maxY, k.Y+k.H/2)
	}
	pixels := make([]pixel, len(keys))
	for i, k := range keys {
		x := (k.X + k.W/2 - minX) / max(maxX-minX, 1)
		y := (k.Y + k.H/2 - minY) / max(maxY-minY, 1)
		dx, dy := (x-0.5)*224, (y-0.5)*64
		pos := position{x, y, math.Hypot(dx, dy) / 116, math.Atan2(dy, dx)/(2*math.Pi) + 0.5}
		pixels[i] = pixel{pos, i, x * 224, y * 64, i < len(mods) && mods[i]}
	}

	f := frame{t: elapsed.Seconds(), fade: 65.28 / float64(max(s.speed, 1)), speed: s.speed}
	for n := int(f.t / hitEvery); n >= 0 && float64(n)*hitEvery > f.t-5; n-- {
		key := int(rnd(n, 3) * float64(len(keys)))
		f.hits = append(f.hits, hit{key, pixels[key].lx, pixels[key].ly, f.t - float64(n)*hitEvery})
	}
	// QMK's timing: at the default speed of 128 a full cycle takes about 2s.
	c := float64(elapsed.Milliseconds()) * float64(s.speed/4+1) / 65536
	name := effectName(s.effect)

	colors := make([]string, len(keys))
	for i, p := range pixels {
		h, sat, val := float64(s.hue)/256, float64(s.sat)/255, float64(s.val)/255
		if fn, ok := fullEffects[name]; ok {
			h, sat, val = fn(p, f, h, sat, val)
		}
		if fn, ok := hueEffects[name]; ok {
			h += fn(p.position, c)
		}
		if fn, ok := satEffects[name]; ok {
			sat *= fn(p.position, c)
		}
		if fn, ok := valEffects[name]; ok {
			val *= fn(p.position, c)
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
