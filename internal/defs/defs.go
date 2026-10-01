// Package defs loads VIA v3 keyboard definitions.
package defs

import (
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Definition struct {
	Name                string
	VendorID, ProductID uint16
	Rows, Cols          int
	Keys                []Key
	Encoders            int      // rotary encoders, from "eN" key legends
	Custom              []string // customKeycodes names, QK_KB_0 onwards
	Menus               []Menu
}

// Key is one physical key in KLE units (1 = 1u). X2/Y2/W2/H2 describe a
// second rectangle relative to X/Y, which is how KLE draws ISO Enter.
type Key struct {
	Row, Col       int
	X, Y, W, H     float64
	X2, Y2, W2, H2 float64
	Knob           bool // the press of a rotary encoder
	Encoder        int  // which encoder, for a knob
}

//go:embed keyboards/*.json
var bundled embed.FS

func Load(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, err
	}
	def, err := parse(data)
	if err != nil {
		return Definition{}, fmt.Errorf("%s: %w", path, err)
	}
	return def, nil
}

// Find looks up the definition for a board, first in the user's config
// folder so a bundled definition can be fixed without waiting for a release,
// then in the bundled set.
func Find(vid, pid uint16) (Definition, error) {
	userDir := "~/.config/via-terminal/keyboards"
	var dirs []fs.FS
	if dir, err := os.UserConfigDir(); err == nil {
		userDir = filepath.Join(dir, "via-terminal", "keyboards")
		dirs = append(dirs, os.DirFS(userDir))
	}
	sub, _ := fs.Sub(bundled, "keyboards")
	for _, fsys := range append(dirs, sub) {
		names, _ := fs.Glob(fsys, "*.json")
		for _, name := range names {
			data, err := fs.ReadFile(fsys, name)
			if err != nil {
				continue
			}
			// Match on the IDs alone first so an unrelated broken file
			// doesn't block the lookup, but a broken match still reports.
			var ids struct{ VendorID, ProductID string }
			if json.Unmarshal(data, &ids) != nil || parseID(ids.VendorID) != vid || parseID(ids.ProductID) != pid {
				continue
			}
			def, err := parse(data)
			if err != nil {
				return Definition{}, fmt.Errorf("%s: %w", name, err)
			}
			return def, nil
		}
	}
	return Definition{}, fmt.Errorf("no definition for %04X:%04X, add one to %s or pass --def", vid, pid, userDir)
}

func parseID(s string) uint16 {
	id, _ := strconv.ParseUint(s, 0, 16)
	return uint16(id)
}

func parse(data []byte) (Definition, error) {
	var raw struct {
		Name                string
		VendorID, ProductID string
		Matrix              struct{ Rows, Cols int }
		Layouts             struct{ Keymap [][]json.RawMessage }
		CustomKeycodes      []struct{ Name string }
		Menus               []json.RawMessage
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Definition{}, err
	}
	vid, err := strconv.ParseUint(raw.VendorID, 0, 16)
	if err != nil {
		return Definition{}, fmt.Errorf("vendorId: %w", err)
	}
	pid, err := strconv.ParseUint(raw.ProductID, 0, 16)
	if err != nil {
		return Definition{}, fmt.Errorf("productId: %w", err)
	}
	keys, encoders, err := parseKLE(raw.Layouts.Keymap)
	if err != nil {
		return Definition{}, err
	}
	for _, k := range keys {
		if k.Row >= raw.Matrix.Rows || k.Col >= raw.Matrix.Cols {
			return Definition{}, fmt.Errorf("key %d,%d is outside the %dx%d matrix", k.Row, k.Col, raw.Matrix.Rows, raw.Matrix.Cols)
		}
	}
	var custom []string
	for _, c := range raw.CustomKeycodes {
		custom = append(custom, c.Name)
	}
	menus, err := parseMenus(raw.Menus)
	if err != nil {
		return Definition{}, err
	}
	return Definition{raw.Name, uint16(vid), uint16(pid), raw.Matrix.Rows, raw.Matrix.Cols, keys, encoders, custom, menus}, nil
}

// parseKLE walks keyboard-layout-editor rows: property objects move the
// cursor and size the next key, strings are keys with "row,col" in legend 0,
// an optional layout option "group,choice" in legend 3 and an optional
// encoder "eN" in legend 9. It returns the keys and the encoder count.
func parseKLE(rows [][]json.RawMessage) ([]Key, int, error) {
	type props struct{ X, Y, W, H, X2, Y2, W2, H2 float64 }
	var keys []Key
	var encoders int
	y := 0.0
	for _, row := range rows {
		x := 0.0
		p := props{W: 1, H: 1}
		for _, item := range row {
			var legend string
			if err := json.Unmarshal(item, &legend); err != nil {
				// Unmarshal only overwrites fields present in the object, so
				// sizes carry over while x/y are fresh offsets each time.
				p.X, p.Y = 0, 0
				if err := json.Unmarshal(item, &p); err != nil {
					return nil, 0, fmt.Errorf("keymap item %s: %w", item, err)
				}
				x += p.X
				y += p.Y
				continue
			}

			labels := strings.Split(legend, "\n")
			k := Key{X: x, Y: y, W: p.W, H: p.H, X2: p.X2, Y2: p.Y2, W2: cmp.Or(p.W2, p.W), H2: cmp.Or(p.H2, p.H)}
			if _, err := fmt.Sscanf(labels[0], "%d,%d", &k.Row, &k.Col); err != nil {
				return nil, 0, fmt.Errorf("key legend %q: want \"row,col\"", legend)
			}
			var encoder int
			if len(labels) > 9 && labels[9] != "" {
				if _, err := fmt.Sscanf(labels[9], "e%d", &encoder); err == nil {
					encoders = max(encoders, encoder+1)
					k.Knob, k.Encoder = true, encoder
				}
			}
			var group, choice int
			if len(labels) > 3 && labels[3] != "" {
				fmt.Sscanf(labels[3], "%d,%d", &group, &choice)
			}
			if choice == 0 {
				keys = append(keys, k)
			}
			x += p.W
			p = props{W: 1, H: 1}
		}
		y++
	}
	return keys, encoders, nil
}
