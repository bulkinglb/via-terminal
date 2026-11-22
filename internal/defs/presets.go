package defs

import (
	"encoding/json"
	"fmt"
	"slices"
)

// presets are the menus a definition can name instead of spelling them out,
// as in "menus": ["qmk_rgb_matrix"]. Channels and value IDs come from QMK's
// via.h; effect values are positions in QMK's effect order, which is what
// the firmware expects.
var presets = map[string][]Menu{
	"qmk_backlight":          {{"Lighting", backlight()}},
	"qmk_rgblight":           {{"Lighting", rgbSection("Underglow", "id_qmk_rgblight", 2, rgblightEffects, 35)}},
	"qmk_backlight_rgblight": {{"Lighting", slices.Concat(backlight(), rgbSection("Underglow", "id_qmk_rgblight", 2, rgblightEffects, 35))}},
	"qmk_rgb_matrix":         {{"Lighting", rgbSection("Backlight", "id_qmk_rgb_matrix", 3, rgbMatrixEffects, 24, 28, 29, 32)}},
	"qmk_audio": {{"Audio", []Control{
		{Label: "General"},
		{Label: "Audio Enable", Type: "toggle", ID: "id_qmk_audio_enable", Channel: 4, ValueID: 1, Max: 255, Options: onOff},
		{Label: "Audio Clicky Enable", Type: "toggle", ID: "id_qmk_audio_clicky_enable", Channel: 4, ValueID: 2, Max: 255, Options: onOff},
	}}},
}

var onOff = []Option{{"Off", 0}, {"On", 1}}

// v2Lighting turns a v2 definition's "lighting" into menus. It is a preset
// name, or an object that extends one and can list its own effects as
// [name, colors needed] pairs.
// shortcut: wt_rgb_backlight and wt_mono_backlight boards get no lighting tab, add them when someone has one.
func v2Lighting(raw json.RawMessage) []Menu {
	var custom struct {
		Extends                   string
		Effects, UnderglowEffects [][]json.RawMessage
	}
	if json.Unmarshal(raw, &custom.Extends) != nil && json.Unmarshal(raw, &custom) != nil {
		return nil
	}
	var items []Control
	if custom.Extends == "qmk_backlight" || custom.Extends == "qmk_backlight_rgblight" {
		items = backlight()
		if names, _ := effectPairs(custom.Effects); names != nil {
			items[2].Options = nil // the effect dropdown
			for i, name := range names {
				items[2].Options = append(items[2].Options, Option{name, i})
			}
		}
	}
	if custom.Extends == "qmk_rgblight" || custom.Extends == "qmk_backlight_rgblight" {
		effects, noColor := rgblightEffects, []int{35}
		if names, none := effectPairs(custom.UnderglowEffects); names != nil {
			effects, noColor = names, none
		}
		items = append(items, rgbSection("Underglow", "id_qmk_rgblight", 2, effects, noColor...)...)
	}
	if items == nil {
		return nil
	}
	return []Menu{{"Lighting", items}}
}

// effectPairs splits [name, colors needed] pairs into names and the
// positions of effects that use no color.
func effectPairs(pairs [][]json.RawMessage) (names []string, noColor []int) {
	for i, p := range pairs {
		var name string
		var colors int
		if len(p) != 2 || json.Unmarshal(p[0], &name) != nil || json.Unmarshal(p[1], &colors) != nil {
			return nil, nil
		}
		names = append(names, name)
		if colors == 0 {
			noColor = append(noColor, i)
		}
	}
	return names, noColor
}

func backlight() []Control {
	return []Control{
		{Label: "Backlight"},
		{Label: "Backlight Brightness", Type: "range", ID: "id_qmk_backlight_brightness", Channel: 1, ValueID: 1, Max: 255},
		{Label: "Backlight Effect", Type: "dropdown", ID: "id_qmk_backlight_effect", Channel: 1, ValueID: 2, Max: 255, Options: []Option{{"Off", 0}, {"Breathing", 1}}},
	}
}

// rgbSection is the brightness, effect, speed and color block that RGB
// light and RGB matrix share. noColor lists effects that ignore the color.
func rgbSection(heading, prefix string, channel byte, effects []string, noColor ...int) []Control {
	effect := "{" + prefix + "_effect}"
	colorIf := effect + " != 0"
	for _, n := range noColor {
		colorIf += fmt.Sprintf(" && %s != %d", effect, n)
	}
	var options []Option
	for i, name := range effects {
		options = append(options, Option{name, i})
	}
	return []Control{
		{Label: heading},
		{Label: "Brightness", Type: "range", ID: prefix + "_brightness", Channel: channel, ValueID: 1, Max: 255},
		{Label: "Effect", Type: "dropdown", ID: prefix + "_effect", Channel: channel, ValueID: 2, Max: 255, Options: options},
		{Label: "Effect Speed", Type: "range", ID: prefix + "_effect_speed", ShowIf: effect + " != 0", Channel: channel, ValueID: 3, Max: 255},
		{Label: "Color", Type: "color", ID: prefix + "_color", ShowIf: colorIf, Channel: channel, ValueID: 4, Max: 255},
	}
}

var rgblightEffects = slices.Concat(
	[]string{"All Off", "Solid Color"},
	numbered("Breathing", 4),
	numbered("Rainbow Mood", 3),
	numbered("Rainbow Swirl", 6),
	numbered("Snake", 6),
	numbered("Knight", 3),
	[]string{"Christmas"},
	numbered("Gradient", 10),
	[]string{"RGB Test", "Alternating"},
	numbered("Twinkle", 6),
)

var rgbMatrixEffects = []string{
	"All Off", "Solid Color", "Alphas Mods", "Gradient Up Down", "Gradient Left Right",
	"Breathing", "Band Sat", "Band Val", "Band Pinwheel Sat", "Band Pinwheel Val",
	"Band Spiral Sat", "Band Spiral Val", "Cycle All", "Cycle Left Right", "Cycle Up Down",
	"Rainbow Moving Chevron", "Cycle Out In", "Cycle Out In Dual", "Cycle Pinwheel", "Cycle Spiral",
	"Dual Beacon", "Rainbow Beacon", "Rainbow Pinwheels", "Raindrops", "Jellybean Raindrops",
	"Hue Breathing", "Hue Pendulum", "Hue Wave", "Pixel Rain", "Pixel Flow",
	"Pixel Fractal", "Typing Heatmap", "Digital Rain", "Solid Reactive Simple", "Solid Reactive",
	"Solid Reactive Wide", "Solid Reactive Multiwide", "Solid Reactive Cross", "Solid Reactive Multicross", "Solid Reactive Nexus",
	"Solid Reactive Multinexus", "Splash", "Multisplash", "Solid Splash", "Solid Multisplash",
}

func numbered(name string, n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s %d", name, i+1)
	}
	return names
}
