package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type definition struct {
	Name                string
	VendorID, ProductID uint16
	Rows, Cols          int
	Keys                []key
	Custom              []string // customKeycodes names, QK_KB_0 onwards
}

// key is one physical key in KLE units (1 = 1u). X2/Y2/W2/H2 describe a
// second rectangle relative to X/Y, which is how KLE draws ISO Enter.
type key struct {
	Row, Col       int
	X, Y, W, H     float64
	X2, Y2, W2, H2 float64
}

func loadDefinition(path string) (definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return definition{}, err
	}
	var raw struct {
		Name                string
		VendorID, ProductID string
		Matrix              struct{ Rows, Cols int }
		Layouts             struct{ Keymap [][]json.RawMessage }
		CustomKeycodes      []struct{ Name string }
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return definition{}, fmt.Errorf("%s: %w", path, err)
	}
	vid, err := strconv.ParseUint(raw.VendorID, 0, 16)
	if err != nil {
		return definition{}, fmt.Errorf("%s: vendorId: %w", path, err)
	}
	pid, err := strconv.ParseUint(raw.ProductID, 0, 16)
	if err != nil {
		return definition{}, fmt.Errorf("%s: productId: %w", path, err)
	}
	keys, err := parseKLE(raw.Layouts.Keymap)
	if err != nil {
		return definition{}, fmt.Errorf("%s: %w", path, err)
	}
	for _, k := range keys {
		if k.Row >= raw.Matrix.Rows || k.Col >= raw.Matrix.Cols {
			return definition{}, fmt.Errorf("%s: key %d,%d is outside the %dx%d matrix", path, k.Row, k.Col, raw.Matrix.Rows, raw.Matrix.Cols)
		}
	}
	var custom []string
	for _, c := range raw.CustomKeycodes {
		custom = append(custom, c.Name)
	}
	return definition{raw.Name, uint16(vid), uint16(pid), raw.Matrix.Rows, raw.Matrix.Cols, keys, custom}, nil
}

// parseKLE walks keyboard-layout-editor rows: property objects move the
// cursor and size the next key, strings are keys with "row,col" in legend 0
// and an optional layout option "group,choice" in legend 3.
func parseKLE(rows [][]json.RawMessage) ([]key, error) {
	type props struct{ X, Y, W, H, X2, Y2, W2, H2 float64 }
	var keys []key
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
					return nil, fmt.Errorf("keymap item %s: %w", item, err)
				}
				x += p.X
				y += p.Y
				continue
			}

			labels := strings.Split(legend, "\n")
			k := key{X: x, Y: y, W: p.W, H: p.H, X2: p.X2, Y2: p.Y2, W2: cmp.Or(p.W2, p.W), H2: cmp.Or(p.H2, p.H)}
			if _, err := fmt.Sscanf(labels[0], "%d,%d", &k.Row, &k.Col); err != nil {
				return nil, fmt.Errorf("key legend %q: want \"row,col\"", legend)
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
	return keys, nil
}
