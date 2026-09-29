package defs

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseKLE(t *testing.T) {
	// ISO Enter row from the M1 V5 definition, plus an alternate layout
	// choice that must be dropped.
	rows := [][]json.RawMessage{
		{json.RawMessage(`{"w": 1.5}`), json.RawMessage(`"2,0"`), json.RawMessage(`{"x": 0.25, "w": 1.5, "h": 2, "h2": 1, "x2": -0.25}`), json.RawMessage(`"3,13"`), json.RawMessage(`{"x": 0.5}`), json.RawMessage(`"2,14"`)},
		{json.RawMessage(`{"y": 0.25}`), json.RawMessage(`"4,0\n\n\n0,1"`), json.RawMessage(`"4,1\n\n\n0,0"`)},
	}
	keys, encoders, err := parseKLE(rows)
	if err != nil || encoders != 0 {
		t.Fatal(encoders, err)
	}
	want := []Key{
		{Row: 2, Col: 0, X: 0, Y: 0, W: 1.5, H: 1, W2: 1.5, H2: 1},
		{Row: 3, Col: 13, X: 1.75, Y: 0, W: 1.5, H: 2, X2: -0.25, W2: 1.5, H2: 1},
		{Row: 2, Col: 14, X: 3.75, Y: 0, W: 1, H: 1, W2: 1, H2: 1},
		{Row: 4, Col: 1, X: 1, Y: 1.25, W: 1, H: 1, W2: 1, H2: 1},
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("got  %+v\nwant %+v", keys, want)
	}
}

func TestFind(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sub, _ := fs.Sub(bundled, "keyboards")
	names, _ := fs.Glob(sub, "*.json")
	for _, name := range names {
		data, _ := fs.ReadFile(sub, name)
		if _, err := parse(data); err != nil {
			t.Errorf("bundled %s: %v", name, err)
		}
	}

	def, err := Find(0x342D, 0xE4C2)
	if err != nil || def.Rows != 6 || def.Cols != 15 || def.Encoders != 1 {
		t.Fatalf("bundled M1 V5 ISO: %+v, %v", def, err)
	}

	userDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "via-terminal", "keyboards")
	os.MkdirAll(userDir, 0o755)
	os.WriteFile(filepath.Join(userDir, "broken.json"), []byte("{"), 0o644)
	os.WriteFile(filepath.Join(userDir, "mine.json"), []byte(`{"name": "Mine", "vendorId": "0x342D", "productId": "0xE4C2", "matrix": {"rows": 1, "cols": 1}, "layouts": {"keymap": [["0,0"]]}}`), 0o644)
	if def, err := Find(0x342D, 0xE4C2); err != nil || def.Name != "Mine" {
		t.Errorf("user folder should win: %+v, %v", def, err)
	}
	if _, err := Find(0xFFFF, 0x0001); err == nil {
		t.Error("expected an error for an unknown board")
	}
}

func TestMenus(t *testing.T) {
	def, err := Find(0x342D, 0xE4C2)
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Menus) != 1 || def.Menus[0].Label != "Lighting" {
		t.Fatalf("menus: %+v", def.Menus)
	}
	items := def.Menus[0].Items
	if len(items) != 5 || items[0].Type != "" || items[0].Label != "Backlight" {
		t.Fatalf("want a Backlight heading and 4 controls, got %+v", items)
	}
	effect, speed, color := items[2], items[3], items[4]
	if effect.Channel != 3 || effect.ValueID != 2 || len(effect.Options) != 46 || effect.Options[1] != (Option{"Solid Color", 1}) {
		t.Errorf("effect: %+v", effect)
	}
	if color.Size() != 2 || items[1].Size() != 1 {
		t.Error("color should take two bytes, brightness one")
	}
	for effectValue, want := range map[int][2]bool{0: {false, false}, 7: {true, true}, 24: {true, false}} {
		v := map[string]int{"id_qmk_rgb_matrix_effect": effectValue}
		if got := [2]bool{speed.Visible(v), color.Visible(v)}; got != want {
			t.Errorf("effect %d: speed/color visible = %v, want %v", effectValue, got, want)
		}
	}

	values := map[string]int{"a": 1, "b": 3, "c": 0}
	for cond, want := range map[string]bool{
		"({a} == 1 || {b} > 5) && !{c}": true,
		"{a} == 1 && {b} <= 2":          true && false,
		"{missing} == 0":                true,
		"{a} ==":                        true, // unreadable conditions show the control
	} {
		if got := (Control{ShowIf: cond}).Visible(values); got != want {
			t.Errorf("%q = %v, want %v", cond, got, want)
		}
	}
	if _, err := parse([]byte(`{"vendorId":"0x1","productId":"0x1","matrix":{"rows":1,"cols":1},"layouts":{"keymap":[["0,0"]]},"menus":[{"label":"L","content":[{"label":"x","type":"range","content":["id"]}]}]}`)); err == nil {
		t.Error("short content should be an error, not a panic")
	}
}
